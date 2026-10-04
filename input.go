package main

// inputBackend abstracts how CrossBoard injects keyboard/mouse input on the host.
// The backend is chosen once at startup by pickBackend (see input_linux.go /
// input_other.go).
type inputBackend interface {
	Name() string
	// MoveAbs moves the pointer to a normalized screen position (0..1 on each axis).
	MoveAbs(nx, ny float64)
	// Click presses and releases a button: "left" (default), "right" or "middle".
	Click(button string)
	// Scroll scrolls by whole notches. dy > 0 = up, dx > 0 = right.
	Scroll(dx, dy int)
	// KeyTap taps key while holding mods (e.g. KeyTap("v", "ctrl")).
	KeyTap(key string, mods ...string)
	// Type types literal text.
	Type(text string)
}

var input = pickBackend()

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
