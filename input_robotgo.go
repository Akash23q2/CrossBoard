package main

import "github.com/go-vgo/robotgo"

// robotgoBackend is the original behaviour: X11 (XTest) on Linux, native APIs
// on Windows and macOS.
type robotgoBackend struct{}

func (robotgoBackend) Name() string { return "robotgo" }

func (robotgoBackend) MoveAbs(nx, ny float64) {
	sw, sh := robotgo.GetScreenSize()
	robotgo.Move(int(nx*float64(sw-1)), int(ny*float64(sh-1)))
}

func (robotgoBackend) Click(button string) {
	if button == "" {
		button = "left"
	}
	robotgo.Click(button, false)
}

func (robotgoBackend) Scroll(dx, dy int) {
	robotgo.Scroll(0, dy)
	scrollHorizontal(dx)
}

func (robotgoBackend) KeyTap(key string, mods ...string) {
	args := make([]interface{}, len(mods))
	for i, m := range mods {
		args[i] = m
	}
	robotgo.KeyTap(key, args...)
}

func (robotgoBackend) Type(text string) { robotgo.Type(text) }
