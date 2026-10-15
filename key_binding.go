package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.design/x/clipboard"
)

const (
	maxDeviceClipboard    = 1 << 20
	clipboardMarkerPrefix = "CrossBoard clipboard selection pending: "
)

var (
	clipboardShortcutMu         sync.Mutex
	globalHotkeySuppressedUntil atomic.Int64
)

func runClipboardBinding(action func(deviceIdentity) error) {
	clipboardShortcutMu.Lock()
	defer clipboardShortcutMu.Unlock()
	device, ok := activeDevice()
	if !ok {
		return
	}
	if err := action(device); err != nil {
		appLogf("clipboard shortcut failed: %v", err)
	}
}

func sendPCClipboardToDevice(device deviceIdentity) error {
	if err := clipboard.Init(); err != nil {
		return fmt.Errorf("could not initialize PC clipboard: %w", err)
	}
	previousClipboard, err := readClipboard()
	if err != nil {
		return fmt.Errorf("could not save existing PC clipboard: %w", err)
	}
	markerID, err := newDeviceUUID()
	if err != nil {
		return fmt.Errorf("could not prepare clipboard selection: %w", err)
	}
	// The clipboard watcher ignores identical consecutive text; a marker lets it detect selecting the same text twice.
	marker := clipboardMarkerPrefix + markerID
	if err := writeClipboard(marker); err != nil {
		restoreClipboardIfUnchanged(marker, previousClipboard)
		return fmt.Errorf("could not prepare PC clipboard for selected-text copy: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := clipboard.Watch(ctx, clipboard.FmtText)
	timeout := time.Duration(currentAppConfig().DeviceRequestTimeout) * time.Second
	content, err := captureClipboardSelection(timeout, changes, func() error {
		if err := waitClipboardHotkeyRelease(timeout); err != nil {
			return err
		}
		pressHostShortcut("c", "ctrl", "shift")
		return nil
	})
	if err != nil {
		restoreClipboardIfUnchanged(marker, previousClipboard)
		return fmt.Errorf("could not copy selected text from PC: %w", err)
	}
	endpoint, err := deviceEndpoint(device.IP, "postclipboard")
	if err != nil {
		return err
	}
	if err := postClipboardToDevice(deviceRequestClient(), endpoint, content); err != nil {
		return fmt.Errorf("could not send clipboard to device %s: %w", device.IP, err)
	}
	return nil
}

func restoreClipboardIfUnchanged(marker, previous string) {
	current, err := readClipboard()
	if err != nil {
		appLogf("could not check clipboard after failed selection copy: %v", err)
		return
	}
	if current == marker {
		if err := writeClipboard(previous); err != nil {
			appLogf("could not restore PC clipboard after failed selection copy: %v", err)
		}
	}
}

func postClipboardToDevice(client *http.Client, endpoint, content string) error {
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(content))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain; charset=utf-8")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("device rejected clipboard (HTTP %d): %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func receiveDeviceClipboard(device deviceIdentity) error {
	endpoint, err := deviceEndpoint(device.IP, "getclipboard")
	if err != nil {
		return err
	}
	timeout := time.Duration(currentAppConfig().DeviceRequestTimeout) * time.Second
	content, err := getClipboardFromDevice(deviceRequestClient(), endpoint)
	if err != nil {
		return fmt.Errorf("could not receive clipboard from device %s: %w", device.IP, err)
	}
	if err := writeClipboardThenPaste(string(content), writeClipboard, func() error {
		if err := waitClipboardHotkeyRelease(timeout); err != nil {
			return err
		}
		pressHostShortcut("v", "ctrl")
		return nil
	}); err != nil {
		return fmt.Errorf("could not paste device clipboard on PC: %w", err)
	}
	return nil
}

func captureClipboardSelection(timeout time.Duration, changes <-chan []byte, copySelection func() error) (string, error) {
	if timeout <= 0 {
		return "", fmt.Errorf("clipboard selection timeout must be positive")
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	if err := copySelection(); err != nil {
		return "", err
	}
	select {
	case content, ok := <-changes:
		if !ok {
			return "", fmt.Errorf("PC clipboard watcher stopped before selection was copied")
		}
		return string(content), nil
	case <-timer.C:
		return "", fmt.Errorf("timed out waiting for selected text to be copied")
	}
}

func writeClipboardThenPaste(content string, write func(string) error, paste func() error) error {
	if err := write(content); err != nil {
		return fmt.Errorf("could not write device clipboard to PC: %w", err)
	}
	if err := paste(); err != nil {
		return fmt.Errorf("could not trigger paste: %w", err)
	}
	return nil
}

func pressHostShortcut(key string, modifiers ...string) {
	globalHotkeySuppressedUntil.Store(time.Now().Add(300 * time.Millisecond).UnixNano())
	input.KeyTap(key, modifiers...)
}

func globalHotkeysSuppressed() bool {
	return time.Now().UnixNano() < globalHotkeySuppressedUntil.Load()
}

func runDesktopShortcut(action string) {
	switch action {
	case "scrcpy":
		if err := castActiveAppDevice(); err != nil {
			appLogf("scrcpy shortcut failed: %v", err)
		}
	}
}

func openDesktopPage(path string) {
	url := fmt.Sprintf("http://127.0.0.1:%d%s", serverListenPort, path)
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		command = exec.Command("open", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	if err := command.Start(); err != nil {
		appLogf("could not open %s: %v", url, err)
		return
	}
	go func() {
		if err := command.Wait(); err != nil {
			appLogf("browser command for %s failed: %v", url, err)
		}
	}()
}

func getClipboardFromDevice(client *http.Client, endpoint string) ([]byte, error) {
	response, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("device rejected clipboard request (HTTP %d): %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, maxDeviceClipboard+1))
	if err != nil {
		return nil, err
	}
	if len(content) > maxDeviceClipboard {
		return nil, fmt.Errorf("device clipboard response exceeds 1 MiB")
	}
	return content, nil
}

func deviceEndpoint(ip, path string) (string, error) {
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid registered device IP %q", ip)
	}
	port := currentAppConfig().DevicePort
	if path != "getclipboard" && path != "postclipboard" {
		return "", fmt.Errorf("invalid device endpoint %q", path)
	}
	return "http://" + net.JoinHostPort(ip, strconv.Itoa(port)) + "/" + path, nil
}
