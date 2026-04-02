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

	http.HandleFunc("/", root)
	http.HandleFunc("/health", getHealth)
	http.HandleFunc("/platform", getPlatform)
	http.HandleFunc("/clipboard", getClipboard)
	http.HandleFunc("/clipboard/send", sendClipboard)
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/getfile", getFile)
	http.HandleFunc("/listfiles", listFiles)
	http.HandleFunc("/uploadfile", uploadFile)

	//start the server
	fmt.Println("starting crossboard server!")
	url := handleTunneling()
	generateQr(url)
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		fmt.Println(err)
	}

}
