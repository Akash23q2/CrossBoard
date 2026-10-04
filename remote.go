package main

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type   string   `json:"type"`
	Value  string   `json:"value"` // for key/type
	X      float64  `json:"x"`     // for mouse, 0-1 normalized
	Y      float64  `json:"y"`
	DeltaX int      `json:"deltaX"`
	DeltaY int      `json:"deltaY"`
	Click  bool     `json:"click"` // true = left click
	Button string   `json:"button"`
	Keys   []string `json:"keys"` // for hotkey, e.g. ["ctrl","v"]
	Scale  float64  `json:"scale"`
}

func remoteControlListener(conn *websocket.Conn) {
	defer conn.Close()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var wsMessage WSMessage
		if err := json.Unmarshal(msg, &wsMessage); err != nil {
			fmt.Println("invalid message:", err)
			continue
		}

		switch wsMessage.Type {
		case "key":
			// special keys: "enter", "backspace", "up", "down", "left", "right", "escape", "tab"
			if runtime.GOOS == "windows" && (wsMessage.Value == "brightness_up" || wsMessage.Value == "brightness_down") {
				fmt.Println("brightness control is not supported on Windows")
				continue
			}
			input.KeyTap(wsMessage.Value)

		case "type":
			// bulk text input
			input.Type(wsMessage.Value)

		case "mouse":
			// The client already applies SCALE to its position math, so the
			// server must not scale again.
			input.MoveAbs(clamp01(wsMessage.X), clamp01(wsMessage.Y))
			if wsMessage.Click {
				btn := wsMessage.Button
				if btn == "" {
					btn = "left"
				}
				input.Click(btn)
			}

		case "scroll":
			input.Scroll(wsMessage.DeltaX, wsMessage.DeltaY)

		case "hotkey":
			if len(wsMessage.Keys) > 0 {
				main := wsMessage.Keys[len(wsMessage.Keys)-1]
				mods := wsMessage.Keys[:len(wsMessage.Keys)-1]
				input.KeyTap(main, mods...)
			}

		default:
			fmt.Println("unknown message type:", wsMessage.Type)
			continue
		}
	}
}
