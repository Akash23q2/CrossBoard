package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateAppConfigPersistsAndReportsRestart(t *testing.T) {
	configState.Lock()
	previousConfig, previousPath := configState.config, configState.path
	configState.config = defaultAppConfig()
	configState.path = filepath.Join(t.TempDir(), "config.json")
	configState.Unlock()
	previousListenPort := serverListenPort
	serverListenPort = defaultServerPort
	t.Cleanup(func() {
		configState.Lock()
		configState.config, configState.path = previousConfig, previousPath
		configState.Unlock()
		serverListenPort = previousListenPort
	})

	config := defaultAppConfig()
	config.ServerPort = 6060
	config.DevicePort = 8080
	config.DeviceRequestTimeout = 12
	config.TunnelStartupTimeout = 45
	config.ADBConnectTimeout = 9
	config.FileBrowserDir = t.TempDir()
	content, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	restartRequired, err := updateAppConfig(strings.NewReader(string(content)))
	if err != nil {
		t.Fatal(err)
	}
	if !restartRequired {
		t.Fatal("changing the server port must require a restart")
	}
	if got := currentAppConfig(); got != config {
		t.Fatalf("current config = %+v, want %+v", got, config)
	}
	saved, err := os.ReadFile(configState.path)
	if err != nil {
		t.Fatal(err)
	}
	var persisted AppConfig
	if err := json.Unmarshal(saved, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != config {
		t.Fatalf("persisted config = %+v, want %+v", persisted, config)
	}
	config.DevicePort = 8081
	updated, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := updateAppConfig(strings.NewReader(string(updated))); err != nil {
		t.Fatalf("replace existing config: %v", err)
	}
}

func TestUpdateAppConfigRejectsInvalidSettingsWithoutChangingCurrentConfig(t *testing.T) {
	configState.Lock()
	previous := configState.config
	configState.config = defaultAppConfig()
	configState.Unlock()
	t.Cleanup(func() {
		configState.Lock()
		configState.config = previous
		configState.Unlock()
	})

	invalid := `{"server_port":0,"device_port":80,"device_request_timeout_seconds":5,"tunnel_startup_timeout_seconds":20,"adb_connect_timeout_seconds":5}`
	if _, err := updateAppConfig(strings.NewReader(invalid)); err == nil {
		t.Fatal("invalid port was accepted")
	}
	if got := currentAppConfig(); got != defaultAppConfig() {
		t.Fatalf("invalid update changed current config: %+v", got)
	}
}

func TestValidateAppConfigRejectsOutOfRangeTimeouts(t *testing.T) {
	config := defaultAppConfig()
	config.DeviceRequestTimeout = maxRequestTimeoutSeconds + 1
	if err := validateAppConfig(config); err == nil {
		t.Fatal("out-of-range device timeout was accepted")
	}
}

func TestNormalizeFileBrowserDirRequiresExistingDirectory(t *testing.T) {
	directory := t.TempDir()
	got, err := normalizeFileBrowserDir(directory)
	if err != nil {
		t.Fatalf("normalizeFileBrowserDir(directory) error = %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("normalized directory %q is not absolute", got)
	}
	file := filepath.Join(directory, "file.txt")
	if err := os.WriteFile(file, []byte("text"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := normalizeFileBrowserDir(file); err == nil {
		t.Fatal("file_browser_dir accepted a regular file")
	}
	if _, err := normalizeFileBrowserDir(filepath.Join(directory, "missing")); err == nil {
		t.Fatal("file_browser_dir accepted a nonexistent directory")
	}
}

func TestSettingsEndpointsAreLoopbackOnly(t *testing.T) {
	for _, path := range []string{"/settings", "/api/config"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.RemoteAddr = "192.0.2.50:1234"
		response := httptest.NewRecorder()
		if path == "/settings" {
			settingsPage(response, request)
		} else {
			getAppConfig(response, request)
		}
		if response.Code != http.StatusForbidden {
			t.Errorf("%s status = %d, want %d", path, response.Code, http.StatusForbidden)
		}
	}
	localRequest := httptest.NewRequest("GET", "http://127.0.0.1:5050/api/config", nil)
	localRequest.RemoteAddr = "127.0.0.1:1234"
	if !isLocalRequest(localRequest) {
		t.Fatal("loopback request was not recognized as local")
	}
	tunnelRequest := httptest.NewRequest("GET", "https://public-tunnel.example/api/config", nil)
	tunnelRequest.RemoteAddr = "127.0.0.1:1234"
	if isLocalRequest(tunnelRequest) {
		t.Fatal("request through a loopback tunnel proxy was treated as local")
	}
}

func TestSettingsPageOffersManualFolderPathExamples(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://localhost:5050/settings", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	response := httptest.NewRecorder()
	settingsPage(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("settings status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, want := range []string{"file_browser_dir", "default", "/home/alex/Share", "/Users/alex/Share", `C:\Users\Alex\Share`} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("settings page is missing %q", want)
		}
	}
	if strings.Contains(response.Body.String(), "CHOOSE FOLDER") {
		t.Error("settings page should not offer a folder picker")
	}
}
