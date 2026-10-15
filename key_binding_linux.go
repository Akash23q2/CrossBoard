//go:build linux

package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	evdevEventKey = 0x01
	evdevKeyDown  = 1
	evdevKeyUp    = 0
	evdevCtrlL    = 29
	evdevCtrlR    = 97
	evdevAltL     = 56
	evdevAltR     = 100
	evdevMetaL    = 125
	evdevMetaR    = 126
	evdevKeyC     = 46
	evdevKeyV     = 47
	evdevKeyK     = 37
	evdevKeyBits  = 96
)

type evdevKeyEvent struct {
	time  syscall.Timeval
	kind  uint16
	code  uint16
	value int32
}

var globalPressedKeys = struct {
	sync.Mutex
	keys map[uint16]bool
}{keys: make(map[uint16]bool)}

var evdevKeyboardFiles = struct {
	sync.Mutex
	files map[string]bool
}{files: make(map[string]bool)}

func startKeyBindingListener() {
	if count := scanEvdevKeyboards(); count > 0 {
		appLogf("global shortcuts enabled via Linux input devices: Alt+C/V clipboard; Alt+K scrcpy")
	} else {
		appLogf("global shortcuts unavailable: cannot read a keyboard from /dev/input; grant this user read access to the keyboard event device (usually via the input group), then restart CrossBoard")
	}
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			scanEvdevKeyboards()
		}
	}()
}

func scanEvdevKeyboards() int {
	paths, err := filepath.Glob("/dev/input/event*")
	if err != nil {
		appLogf("global shortcut keyboard scan failed: %v", err)
		return 0
	}
	count := 0
	for _, path := range paths {
		evdevKeyboardFiles.Lock()
		alreadyOpen := evdevKeyboardFiles.files[path]
		evdevKeyboardFiles.Unlock()
		if alreadyOpen {
			count++
			continue
		}

		file, err := os.Open(path)
		if err != nil {
			continue
		}
		if !isEvdevKeyboard(file) {
			file.Close()
			continue
		}

		evdevKeyboardFiles.Lock()
		if evdevKeyboardFiles.files[path] {
			evdevKeyboardFiles.Unlock()
			file.Close()
			count++
			continue
		}
		evdevKeyboardFiles.files[path] = true
		evdevKeyboardFiles.Unlock()
		count++
		go readEvdevKeyboard(path, file)
	}
	return count
}

func isEvdevKeyboard(file *os.File) bool {
	name := make([]byte, 256)
	nameRequest := uintptr(0x80000000 | ((len(name) & 0x3fff) << 16) | (int('E') << 8) | 0x06)
	if err := ioctl(int(file.Fd()), nameRequest, uintptr(unsafe.Pointer(&name[0]))); err == nil &&
		strings.HasPrefix(string(name), "CrossBoard virtual keyboard") {
		return false
	}
	bits := make([]byte, evdevKeyBits)
	request := uintptr(0x80000000 | ((len(bits) & 0x3fff) << 16) | (int('E') << 8) | 0x21)
	if err := ioctl(int(file.Fd()), request, uintptr(unsafe.Pointer(&bits[0]))); err != nil {
		return false
	}
	has := func(code uint16) bool {
		return int(code/8) < len(bits) && bits[code/8]&(1<<(code%8)) != 0
	}
	return (has(evdevCtrlL) || has(evdevCtrlR)) &&
		(has(evdevMetaL) || has(evdevMetaR)) &&
		has(evdevKeyC) && has(evdevKeyV) && has(evdevKeyK)
}

func readEvdevKeyboard(path string, file *os.File) {
	defer func() {
		file.Close()
		evdevKeyboardFiles.Lock()
		delete(evdevKeyboardFiles.files, path)
		evdevKeyboardFiles.Unlock()
	}()

	eventSize := int(unsafe.Sizeof(evdevKeyEvent{}))
	buffer := make([]byte, eventSize)
	for {
		if _, err := io.ReadFull(file, buffer); err != nil {
			if err != io.EOF {
				appLogf("global shortcut keyboard %s stopped: %v", path, err)
			}
			return
		}
		event := evdevKeyEvent{
			kind:  binary.NativeEndian.Uint16(buffer[runtimeSizeofTimeval():]),
			code:  binary.NativeEndian.Uint16(buffer[runtimeSizeofTimeval()+2:]),
			value: int32(binary.NativeEndian.Uint32(buffer[runtimeSizeofTimeval()+4:])),
		}
		if event.kind == evdevEventKey {
			processEvdevKey(event.code, event.value)
		}
	}
}

func runtimeSizeofTimeval() int {
	return int(unsafe.Sizeof(syscall.Timeval{}))
}

func processEvdevKey(code uint16, value int32) {
	globalPressedKeys.Lock()
	action := updateGlobalKeyState(globalPressedKeys.keys, code, value)
	globalPressedKeys.Unlock()
	if globalHotkeysSuppressed() {
		return
	}

	switch action {
	case "send":
		go runClipboardBinding(sendPCClipboardToDevice)
	case "receive":
		go runClipboardBinding(receiveDeviceClipboard)
	case "scrcpy":
		go runDesktopShortcut(action)
	}
}

func waitClipboardHotkeyRelease(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		globalPressedKeys.Lock()
		altDown := globalPressedKeys.keys[evdevAltL] || globalPressedKeys.keys[evdevAltR]
		globalPressedKeys.Unlock()
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

func updateGlobalKeyState(pressed map[uint16]bool, code uint16, value int32) string {
	if value == evdevKeyUp {
		delete(pressed, code)
		return ""
	}
	pressed[code] = true
	if value != evdevKeyDown {
		return ""
	}
	alt := pressed[evdevAltL] || pressed[evdevAltR]
	if alt {
		switch code {
		case evdevKeyC:
			return "send"
		case evdevKeyV:
			return "receive"
		}
	}
	if alt && code == evdevKeyK {
		return "scrcpy"
	}
	return ""
}
