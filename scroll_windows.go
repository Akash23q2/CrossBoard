//go:build windows

package main

import "syscall"

var user32 = syscall.NewLazyDLL("user32.dll")
var mouseEvent = user32.NewProc("mouse_event")

func scrollHorizontal(delta int) {
	if delta == 0 {
		return
	}
	const mouseEventHorizontalWheel = 0x01000
	const wheelDelta = 120
	mouseEvent.Call(mouseEventHorizontalWheel, 0, 0, uintptr(uint32(int32(delta*wheelDelta))), 0)
}
