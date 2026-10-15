package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultServerPort        = 5050
	defaultDevicePort        = 80
	defaultDeviceTimeoutSec  = 5
	defaultTunnelTimeoutSec  = 20
	defaultADBTimeoutSec     = 5
	minRequestTimeoutSeconds = 1
	maxRequestTimeoutSeconds = 120
	minTunnelTimeoutSeconds  = 5
	maxTunnelTimeoutSeconds  = 180
	minADBTimeoutSeconds     = 1
	maxADBTimeoutSeconds     = 120
	maxConfigRequestBytes    = 16 << 10
)

type AppConfig struct {
	ServerPort           int    `json:"server_port"`
	DevicePort           int    `json:"device_port"`
	DeviceRequestTimeout int    `json:"device_request_timeout_seconds"`
	TunnelStartupTimeout int    `json:"tunnel_startup_timeout_seconds"`
	ADBConnectTimeout    int    `json:"adb_connect_timeout_seconds"`
	FileBrowserDir       string `json:"file_browser_dir"`
}

var configState = struct {
	sync.RWMutex
	config AppConfig
	path   string
}{config: defaultAppConfig()}

var configUpdateMu sync.Mutex
var serverListenPort = defaultServerPort

func defaultAppConfig() AppConfig {
	return AppConfig{
		ServerPort:           defaultServerPort,
		DevicePort:           defaultDevicePort,
		DeviceRequestTimeout: defaultDeviceTimeoutSec,
		TunnelStartupTimeout: defaultTunnelTimeoutSec,
		ADBConnectTimeout:    defaultADBTimeoutSec,
	}
}

func loadAppConfig() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("locate user config directory: %w", err)
	}
	path := filepath.Join(configDir, "CrossBoard", "config.json")
	configState.Lock()
	configState.path = path
	configState.Unlock()

	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		configState.Lock()
		configState.config = defaultAppConfig()
		configState.Unlock()
		return saveAppConfig()
	}
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	config := defaultAppConfig()
	if err := json.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	if config.FileBrowserDir != "" {
		config.FileBrowserDir, err = normalizeFileBrowserDir(config.FileBrowserDir)
		if err != nil {
			return fmt.Errorf("invalid config %s: %w", path, err)
		}
	}
	if err := validateAppConfig(config); err != nil {
		return fmt.Errorf("invalid config %s: %w", path, err)
	}
	configState.Lock()
	configState.config = config
	configState.Unlock()
	return nil
}

func currentAppConfig() AppConfig {
	configState.RLock()
	defer configState.RUnlock()
	return configState.config
}

func saveAppConfig() error {
	configState.RLock()
	config, path := configState.config, configState.path
	configState.RUnlock()
	if path == "" {
		return fmt.Errorf("configuration path is not initialized")
	}
	if err := validateAppConfig(config); err != nil {
		return err
	}
	content, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".config-*.json")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return fmt.Errorf("secure temporary config: %w", err)
	}
	if _, err := temp.Write(content); err != nil {
		temp.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return fmt.Errorf("sync config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	if err := replaceConfigFile(tempPath, path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}

func validateAppConfig(config AppConfig) error {
	if config.ServerPort < 1 || config.ServerPort > 65535 {
		return fmt.Errorf("server_port must be between 1 and 65535")
	}
	if config.DevicePort < 1 || config.DevicePort > 65535 {
		return fmt.Errorf("device_port must be between 1 and 65535")
	}
	if config.DeviceRequestTimeout < minRequestTimeoutSeconds || config.DeviceRequestTimeout > maxRequestTimeoutSeconds {
		return fmt.Errorf("device_request_timeout_seconds must be between %d and %d", minRequestTimeoutSeconds, maxRequestTimeoutSeconds)
	}
	if config.TunnelStartupTimeout < minTunnelTimeoutSeconds || config.TunnelStartupTimeout > maxTunnelTimeoutSeconds {
		return fmt.Errorf("tunnel_startup_timeout_seconds must be between %d and %d", minTunnelTimeoutSeconds, maxTunnelTimeoutSeconds)
	}
	if config.ADBConnectTimeout < minADBTimeoutSeconds || config.ADBConnectTimeout > maxADBTimeoutSeconds {
		return fmt.Errorf("adb_connect_timeout_seconds must be between %d and %d", minADBTimeoutSeconds, maxADBTimeoutSeconds)
	}
	if config.FileBrowserDir != "" {
		if _, err := normalizeFileBrowserDir(config.FileBrowserDir); err != nil {
			return fmt.Errorf("file_browser_dir: %w", err)
		}
	}
	return nil
}

func normalizeFileBrowserDir(path string) (string, error) {
	absolute, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("resolve directory path: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve directory %q: %w", path, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("inspect directory %q: %w", path, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q is not a directory", path)
	}
	return resolved, nil
}

func initializeFileBrowserDir(defaultDir string) {
	configState.Lock()
	if configState.config.FileBrowserDir == "" {
		configState.config.FileBrowserDir = defaultDir
	}
	configState.Unlock()
}

func updateAppConfig(body io.Reader) (bool, error) {
	var config AppConfig
	decoder := json.NewDecoder(io.LimitReader(body, maxConfigRequestBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return false, fmt.Errorf("invalid config JSON: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return false, fmt.Errorf("invalid config JSON: unexpected trailing data")
	}
	if config.FileBrowserDir == "" {
		config.FileBrowserDir = appHome
	}
	if config.FileBrowserDir != "" {
		var err error
		config.FileBrowserDir, err = normalizeFileBrowserDir(config.FileBrowserDir)
		if err != nil {
			return false, err
		}
	}
	if err := validateAppConfig(config); err != nil {
		return false, err
	}

	configUpdateMu.Lock()
	defer configUpdateMu.Unlock()
	configState.Lock()
	previous := configState.config
	configState.config = config
	configState.Unlock()
	if err := saveAppConfig(); err != nil {
		configState.Lock()
		configState.config = previous
		configState.Unlock()
		return false, err
	}
	return config.ServerPort != serverListenPort, nil
}

func deviceRequestClient() *http.Client {
	return &http.Client{Timeout: time.Duration(currentAppConfig().DeviceRequestTimeout) * time.Second}
}

func settingsPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/settings" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isLocalRequest(r) {
		http.Error(w, "settings are available only from the CrossBoard computer", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(settingsHTML))
}

func getAppConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isLocalRequest(r) {
		http.Error(w, "settings are available only from the CrossBoard computer", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	config := currentAppConfig()
	if config.FileBrowserDir == "" {
		config.FileBrowserDir = appHome
	}
	_ = json.NewEncoder(w).Encode(config)
}

func putAppConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isLocalRequest(r) {
		http.Error(w, "settings are available only from the CrossBoard computer", http.StatusForbidden)
		return
	}
	restartRequired, err := updateAppConfig(http.MaxBytesReader(w, r.Body, maxConfigRequestBytes))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(struct {
		Config          AppConfig `json:"config"`
		RestartRequired bool      `json:"restart_required"`
	}{currentAppConfig(), restartRequired})
}

func appConfigHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getAppConfig(w, r)
	case http.MethodPut:
		putAppConfig(w, r)
	default:
		w.Header().Set("Allow", "GET, PUT")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func isLocalRequest(r *http.Request) bool {
	ip, err := requestIP(r)
	if err != nil || !net.ParseIP(ip).IsLoopback() {
		return false
	}
	host := r.Host
	if name, _, err := net.SplitHostPort(host); err == nil {
		host = name
	}
	host = strings.Trim(host, "[]")
	return strings.EqualFold(host, "localhost") || (net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback())
}

const settingsHTML = `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>CrossBoard settings</title><style>
*{box-sizing:border-box}body{margin:0;padding:24px;background:#111118;color:#e8e8f0;font:15px system-ui,sans-serif}
main{max-width:680px;margin:auto}h1{color:#00ffe0}p,.note{color:#aaaabc}.field{display:grid;grid-template-columns:1fr 150px;gap:16px;align-items:center;padding:14px 0;border-bottom:1px solid #292936}
input{width:100%;padding:10px;background:#1b1b27;border:1px solid #44445a;border-radius:6px;color:white}button{margin-top:22px;padding:11px 18px;background:#00ffe0;border:0;border-radius:6px;font-weight:bold;cursor:pointer}#status{margin-left:12px;color:#00ffe0}
@media(max-width:520px){.field{grid-template-columns:1fr}input{max-width:220px}}
</style></head><body><main><h1>CrossBoard settings</h1>
<p>Settings are saved to your user config. A server-port change takes effect after restarting CrossBoard. The file-browser folder controls the root directory available for browsing, downloads, and uploads.</p>
<form id="settings">
<label class="field"><span>File browser folder (absolute path)</span><input name="file_browser_dir" type="text"></label>
<label class="field"><span>CrossBoard server port</span><input name="server_port" type="number" min="1" max="65535" required></label>
<label class="field"><span>App device HTTP port</span><input name="device_port" type="number" min="1" max="65535" required></label>
<label class="field"><span>Device request timeout (seconds)</span><input name="device_request_timeout_seconds" type="number" min="1" max="120" required></label>
<label class="field"><span>Tunnel startup timeout (seconds)</span><input name="tunnel_startup_timeout_seconds" type="number" min="5" max="180" required></label>
<label class="field"><span>ADB connect timeout (seconds)</span><input name="adb_connect_timeout_seconds" type="number" min="1" max="120" required></label>
<button type="submit">SAVE SETTINGS</button><span id="status" role="status"></span>
</form><p class="note">By default, the file browser uses the folder containing the CrossBoard executable. Leave this field blank to restore that default, or enter an existing absolute path: Linux <code>/home/alex/Share</code>, macOS <code>/Users/alex/Share</code>, Windows <code>C:\Users\Alex\Share</code>. The selected folder and its contents are shared with connected clients. Device and ADB timeout values apply to subsequent operations. Tunnel startup timeout is used when CrossBoard next launches.</p></main>
<script>
const form=document.getElementById('settings'),status=document.getElementById('status');
async function load(){const response=await fetch('/api/config');if(!response.ok)throw Error(await response.text());const config=await response.json();for(const [key,value] of Object.entries(config))form.elements[key].value=value;}
form.addEventListener('submit',async event=>{event.preventDefault();status.textContent='Saving…';const config=Object.fromEntries([...new FormData(form)].map(([key,value])=>[key,key==='file_browser_dir'?value.trim():Number(value)]));try{const response=await fetch('/api/config',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(config)});const text=await response.text();let result={};try{result=JSON.parse(text);}catch{}if(!response.ok)throw Error(result.error||text||'Could not save settings');status.textContent=result.restart_required?'Saved. Restart CrossBoard to apply the server port.':'Settings saved.';}catch(error){status.textContent=error.message;}});
load().catch(error=>{status.textContent='Could not load settings: '+error.message;});
</script></body></html>`
