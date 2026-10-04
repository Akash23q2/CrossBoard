package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const scrcpyTCPPort = "2004"

var androidPackageName = regexp.MustCompile(`^[A-Za-z0-9_]+(\.[A-Za-z0-9_]+)+$`)
var scrcpyProcesses = struct {
	sync.Mutex
	active map[int]*os.Process
}{active: make(map[int]*os.Process)}

func castHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/cast/")
	if path == "stop" {
		stopScrcpy(w)
		return
	}

	screenOff, err := castScreenOff(r)
	if err != nil {
		http.Error(w, "invalid screen_off value", http.StatusBadRequest)
		return
	}

	ip, err := requestingIPv4(r)
	if err != nil {
		http.Error(w, "requesting device must connect over IPv4", http.StatusBadRequest)
		return
	}

	var args []string
	switch {
	case path == "screen":
		args = scrcpyArgs(ip, "", screenOff)
	case androidPackageName.MatchString(path):
		args = scrcpyArgs(ip, path, screenOff)
	default:
		http.NotFound(w, r)
		return
	}

	if err := ensureADBConnected(ip); err != nil {
		writeCastError(w, http.StatusServiceUnavailable, "adb_not_connected", err.Error())
		return
	}

	cmd := exec.Command("scrcpy", args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting scrcpy:", err)
		http.Error(w, "could not start scrcpy", http.StatusInternalServerError)
		return
	}
	scrcpyProcesses.Lock()
	scrcpyProcesses.active[cmd.Process.Pid] = cmd.Process
	scrcpyProcesses.Unlock()
	go func() {
		defer func() {
			scrcpyProcesses.Lock()
			delete(scrcpyProcesses.active, cmd.Process.Pid)
			scrcpyProcesses.Unlock()
		}()
		_ = cmd.Wait()
	}()

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte("scrcpy started"))
}

func castScreenOff(r *http.Request) (bool, error) {
	var body struct {
		ScreenOff *bool `json:"screen_off"`
	}
	if r.Body != nil {
		decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
		if err := decoder.Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			return false, err
		}
	}
	if values, exists := r.URL.Query()["screen_off"]; exists {
		if len(values) != 1 {
			return false, fmt.Errorf("screen_off must be specified once")
		}
		return strconv.ParseBool(values[0])
	}
	if body.ScreenOff != nil {
		return *body.ScreenOff, nil
	}
	return false, nil
}

func scrcpyArgs(ip, packageName string, screenOff bool) []string {
	args := []string{"--tcpip=" + net.JoinHostPort(ip, scrcpyTCPPort)}
	if packageName != "" {
		args = append(args, "--new-display=1920x1080/240", "--start-app="+packageName)
	}
	if screenOff {
		args = append(args, "-S")
	}
	return args
}

func ensureADBConnected(ip string) error {
	device := net.JoinHostPort(ip, scrcpyTCPPort)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connect := exec.CommandContext(ctx, "adb", "connect", device)
	if output, err := connect.CombinedOutput(); err != nil {
		return fmt.Errorf("ADB could not connect to %s: %s", device, strings.TrimSpace(string(output)))
	}

	state := exec.CommandContext(ctx, "adb", "-s", device, "get-state")
	if output, err := state.CombinedOutput(); err != nil || strings.TrimSpace(string(output)) != "device" {
		return fmt.Errorf("ADB device %s is not ready: %s", device, strings.TrimSpace(string(output)))
	}
	return nil
}

func stopScrcpy(w http.ResponseWriter) {
	scrcpyProcesses.Lock()
	processes := make([]*os.Process, 0, len(scrcpyProcesses.active))
	for _, process := range scrcpyProcesses.active {
		processes = append(processes, process)
	}
	scrcpyProcesses.Unlock()

	var stopErrors []string
	for _, process := range processes {
		if err := process.Signal(os.Interrupt); err != nil && !errors.Is(err, os.ErrProcessDone) {
			stopErrors = append(stopErrors, err.Error())
		}
	}
	if len(stopErrors) > 0 {
		writeCastError(w, http.StatusInternalServerError, "stop_failed", strings.Join(stopErrors, "; "))
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"message": "scrcpy stop signal sent",
	})
}

func writeCastError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}

func requestingIPv4(r *http.Request) (string, error) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil || ip.To4() == nil {
		return "", fmt.Errorf("invalid IPv4 peer address %q", host)
	}
	return ip.To4().String(), nil
}