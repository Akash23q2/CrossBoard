package main

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/go-vgo/robotgo"
	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type   string   `json:"type"`
	Value  string   `json:"value"` // for key/type
	X      float64  `json:"x"`     // for mouse, 0-1 normalized
	Y      float64  `json:"y"`
	Click  bool     `json:"click"` // true = left click
	Button string   `json:"button"`
	Keys   []string `json:"keys"` // for hotkey, e.g. ["ctrl","v"]
	Scale  float64  `json:"scale"`
}

func remoteControlListener(conn *websocket.Conn) {
	defer conn.Close()
	// fmt.Println("remote control client connected")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			// fmt.Println("remote control client disconnected")
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
			robotgo.KeyTap(wsMessage.Value)

		case "type":
			// bulk text input
			robotgo.Type(wsMessage.Value)

		case "mouse":
			sw, sh := robotgo.GetScreenSize()
			x := int(wsMessage.X * float64(sw-1) * wsMessage.Scale)
			y := int(wsMessage.Y * float64(sh-1) * wsMessage.Scale)
			// fmt.Printf("screen: %dx%d, cursor: %d,%d\n", sw, sh, x, y)
			robotgo.Move(x, y)
			if wsMessage.Click {
				btn := wsMessage.Button
				if btn == "" {
					btn = "left"
				}
				robotgo.Click(btn, false)
			}

		case "hotkey":
			if len(wsMessage.Keys) > 0 {
				main := wsMessage.Keys[len(wsMessage.Keys)-1]
				mods := wsMessage.Keys[:len(wsMessage.Keys)-1]
				args := make([]interface{}, len(mods))
				for i, k := range mods {
					args[i] = k
				}
				robotgo.KeyTap(main, args...)
			}

		default:
			fmt.Println("unknown message type:", wsMessage.Type)
			continue
		}
	}
}
