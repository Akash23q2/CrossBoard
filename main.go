// imports
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/term"
)

var appHome string

func main() {
	var err error
	if err := loadAppConfig(); err != nil {
		appLogf("could not load configuration; using defaults: %v", err)
	}
	serverListenPort = currentAppConfig().ServerPort
	appHome, err = os.Executable()
	if err != nil {
		appLogf("could not get executable path: %v", err)
		return
	}
	appHome = filepath.Dir(appHome)
	initializeFileBrowserDir(appHome)
	// fmt.Println("App home directory:", appHome)
	startKeyBindingListener()

	//register routes
	http.HandleFunc("/health", getHealth)
	http.HandleFunc("/platform", getPlatform)
	http.HandleFunc("/clipboard", getClipboard)
	http.HandleFunc("/clipboard/send", sendClipboard)
	http.HandleFunc("/device/connect", connectDevice)
	http.HandleFunc("/help", helpPage)
	http.HandleFunc("/settings", settingsPage)
	http.HandleFunc("/api/config", appConfigHandler)
	http.HandleFunc("/logs", logsPage)
	http.HandleFunc("/api/logs", getAppLogs)
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/getfile/", getFile)
	http.HandleFunc("/listfiles", listFiles)
	http.HandleFunc("/uploadfile", uploadFile)
	http.HandleFunc("/cast/", castHandler)
	http.HandleFunc("/", root) // Register "/" LAST so specific routes take priority

	//start the server
	appLogf("starting CrossBoard server")
	tunnelURL, lanURL := handleTunneling()
	if tunnelURL != "" {
		appLogf("tunnel URL: %s", tunnelURL)
	}
	if lanURL != "" {
		appLogf("local network URL: %s", lanURL)
	}
	setServerURLs(tunnelURL, lanURL)
	interactiveTerminal := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	if !interactiveTerminal {
		refreshQRCodes()
	}

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", serverListenPort), appIdentityMiddleware(http.DefaultServeMux))
	}()

	if interactiveTerminal {
		terminalDashboardActive.Store(true)
		if err := runTerminalDashboard(serverErrors); err != nil {
			terminalDashboardActive.Store(false)
			appLogf("terminal dashboard stopped: %v", err)
		}
		return
	}

	if err := <-serverErrors; err != nil {
		appLogf("HTTP server stopped: %v", err)
	}

}
