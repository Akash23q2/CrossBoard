//imports

package main

import (
	_ "embed"
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
		appLogf("websocket upgrade failed: %v", err)
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
	content, err := readClipboard()
	if err != nil {
		http.Error(w, "could not read clipboard", http.StatusInternalServerError)
		return
	}
	io.WriteString(w, content)
}

func sendClipboard(w http.ResponseWriter, r *http.Request) {
	content, exists := r.URL.Query()["content"]
	if exists && len(content) > 0 && content[0] != "" {
		if err := writeClipboard(content[0]); err != nil {
			http.Error(w, "could not write clipboard", http.StatusInternalServerError)
			return
		}
	}
	io.WriteString(w, "copied content to clipboard")
}
