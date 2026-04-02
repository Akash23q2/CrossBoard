//imports

package main

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"runtime"

	"github.com/gorilla/websocket"
)

//register ui

//go:embed static/index.html
var indexHtml string

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading:", err)
		return
	}
	remoteControlListener(conn)
}

func root(w http.ResponseWriter, r *http.Request) {
	// Only serve index.html for root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	io.WriteString(w, indexHtml)
}

func getHealth(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "OK")
	// fmt.Println("server is healthy and running!")
}

func getPlatform(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, runtime.GOOS)
}

func getClipboard(w http.ResponseWriter, r *http.Request) {
	content := readClipboard()
	// fmt.Println("pasted clipboard content")
	io.WriteString(w, content)
}

func sendClipboard(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "copied content to clipboard")
	content := r.URL.Query().Get("content")
	if content == "" {
		// fmt.Println("nothing to copy")
	} else {
		// fmt.Println("copied content", content)
		writeClipboard(content)
	}
}
