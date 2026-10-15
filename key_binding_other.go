//go:build !linux

package main

import (
	"fmt"
	"sync"
	"time"

	hook "github.com/robotn/gohook"
)

var clipboardModifierState = struct {
	sync.Mutex
	alt bool
}{}

func startKeyBindingListener() {
	hook.Register(hook.KeyDown, []string{"alt"}, func(hook.Event) {
		setClipboardAltState(true)
	})
	hook.Register(hook.KeyUp, []string{"alt"}, func(hook.Event) {
		setClipboardAltState(false)
	})
	hook.Register(hook.KeyDown, []string{"c", "alt"}, func(hook.Event) {
		if globalHotkeysSuppressed() {
			return
		}
		go runClipboardBinding(sendPCClipboardToDevice)
	})
	hook.Register(hook.KeyDown, []string{"v", "alt"}, func(hook.Event) {
		if globalHotkeysSuppressed() {
			return
		}
		go runClipboardBinding(receiveDeviceClipboard)
	})
	hook.Register(hook.KeyDown, []string{"k", "alt"}, func(hook.Event) {
		if globalHotkeysSuppressed() {
			return
		}
		runDesktopShortcut("scrcpy")
	})
	appLogf("global shortcuts enabled: Alt+C/V clipboard; Alt+K scrcpy")
	go hook.Process(hook.Start())
}

func setClipboardAltState(down bool) {
	clipboardModifierState.Lock()
	clipboardModifierState.alt = down
	clipboardModifierState.Unlock()
}

func waitClipboardHotkeyRelease(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		clipboardModifierState.Lock()
		altDown := clipboardModifierState.alt
		clipboardModifierState.Unlock()
		if !altDown {
			return nil
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			return fmt.Errorf("timed out waiting for Alt to be released")
		}
	}
}
