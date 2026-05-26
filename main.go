// imports
package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

var appHome string

func main() {
	var err error
	appHome, err = os.Executable()
	if err != nil {
		fmt.Println("Error getting executable path:", err)
		return
	}
	appHome = filepath.Dir(appHome)
	// fmt.Println("App home directory:", appHome)

	//register routes
	http.HandleFunc("/health", getHealth)
	http.HandleFunc("/platform", getPlatform)
	http.HandleFunc("/clipboard", getClipboard)
	http.HandleFunc("/clipboard/send", sendClipboard)
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/getfile/", getFile)
	http.HandleFunc("/listfiles", listFiles)
	http.HandleFunc("/uploadfile", uploadFile)
	http.HandleFunc("/", root) // Register "/" LAST so specific routes take priority

	//start the server
	fmt.Println("starting crossboard server!")
	tunnelURL, lanURL := handleTunneling()
	if tunnelURL != "" {
		fmt.Println("Tunnel URL:", tunnelURL)
		generateQr(tunnelURL)
	}
	if lanURL != "" {
		fmt.Println("Local network URL:", lanURL)
		generateQr(lanURL)
	}

	// bind to all interfaces so LAN devices can connect
	err = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil)
	if err != nil {
		fmt.Println(err)
	}

}
