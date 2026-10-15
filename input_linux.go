//go:build linux

package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unsafe"
)

// ── Backend selection ────────────────────────────────────────────────────────
//
// X11 session     -> robotgo (XTest), as before.
// Wayland session -> /dev/uinput virtual keyboard + absolute pointer. This is
//                    compositor independent (KDE, GNOME, Sway, ...) because the
//                    kernel injects the events.
//
// Override with CROSSBOARD_INPUT=uinput or CROSSBOARD_INPUT=x11.

func pickBackend() inputBackend {
	switch strings.ToLower(os.Getenv("CROSSBOARD_INPUT")) {
	case "x11", "robotgo":
		appLogf("input: using X11 (robotgo) backend (forced)")
		return robotgoBackend{}
	case "uinput":
		if b, err := newUinputBackend(); err == nil {
			appLogf("input: using /dev/uinput backend (forced)")
			return b
		} else {
			appLogf("input: forced uinput backend failed: %v", err)
			printUinputHelp()
			return robotgoBackend{}
		}
	}

	if isWaylandSession() {
		b, err := newUinputBackend()
		if err == nil {
			appLogf("input: Wayland session detected, using /dev/uinput backend")
			return b
		}
		appLogf("input: Wayland session detected but /dev/uinput is not usable: %v", err)
		printUinputHelp()
		appLogf("input: falling back to X11 backend (mouse will not work under Wayland)")
		return robotgoBackend{}
	}

	appLogf("input: X11 session, using robotgo backend")
	return robotgoBackend{}
}

func isWaylandSession() bool {
	st := strings.ToLower(os.Getenv("XDG_SESSION_TYPE"))
	if st == "wayland" {
		return true
	}
	return st != "x11" && os.Getenv("WAYLAND_DISPLAY") != ""
}

func printUinputHelp() {
	appLogf(`input: one-time setup so your user can create virtual input devices:
  echo 'KERNEL=="uinput", SUBSYSTEM=="misc", MODE="0660", TAG+="uaccess", OPTIONS+="static_node=uinput"' | sudo tee /etc/udev/rules.d/99-crossboard-uinput.rules
  echo uinput | sudo tee /etc/modules-load.d/uinput.conf
  sudo modprobe uinput && sudo udevadm control --reload-rules && sudo udevadm trigger
  (then log out and back in, and restart crossboard)`)
}

// ── Linux input event constants (linux/input-event-codes.h) ──────────────────

const (
	evSyn uint16 = 0x00
	evKey uint16 = 0x01
	evRel uint16 = 0x02
	evAbs uint16 = 0x03

	relHWheel uint16 = 0x06
	relWheel  uint16 = 0x08
	absX      uint16 = 0x00
	absY      uint16 = 0x01

	btnLeft   uint16 = 0x110
	btnRight  uint16 = 0x111
	btnMiddle uint16 = 0x112

	keyLeftCtrl  uint16 = 29
	keyLeftShift uint16 = 42
	keyLeftAlt   uint16 = 56
	keyLeftMeta  uint16 = 125

	absRange = 32767

	// uinput ioctls (linux/uinput.h)
	uiDevCreate = 0x5501
	uiSetEvBit  = 0x40045564
	uiSetKeyBit = 0x40045565
	uiSetRelBit = 0x40045566
	uiSetAbsBit = 0x40045567
)

type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

type uinputUserDev struct {
	Name         [80]byte
	Bustype      uint16
	Vendor       uint16
	Product      uint16
	Version      uint16
	FFEffectsMax uint32
	Absmax       [64]int32
	Absmin       [64]int32
	Absfuzz      [64]int32
	Absflat      [64]int32
}

func ioctl(fd int, req, arg uintptr) error {
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg); e != 0 {
		return e
	}
	return nil
}

// ── Virtual device ───────────────────────────────────────────────────────────

type uinputDev struct {
	mu sync.Mutex
	fd int
}

func newUinputDev(name string, evs, keys, rels, abss []uint16) (*uinputDev, error) {
	fd, err := syscall.Open("/dev/uinput", syscall.O_WRONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	fail := func(e error) (*uinputDev, error) {
		syscall.Close(fd)
		return nil, e
	}
	set := func(req uintptr, vals []uint16) error {
		for _, v := range vals {
			if err := ioctl(fd, req, uintptr(v)); err != nil {
				return err
			}
		}
		return nil
	}
	if err := set(uiSetEvBit, evs); err != nil {
		return fail(err)
	}
	if err := set(uiSetKeyBit, keys); err != nil {
		return fail(err)
	}
	if err := set(uiSetRelBit, rels); err != nil {
		return fail(err)
	}
	if err := set(uiSetAbsBit, abss); err != nil {
		return fail(err)
	}

	var dev uinputUserDev
	copy(dev.Name[:], name)
	dev.Bustype, dev.Vendor, dev.Product, dev.Version = 0x03, 0xc0de, 0xb0a2, 1
	for _, a := range abss {
		dev.Absmax[a] = absRange
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&dev)), unsafe.Sizeof(dev))
	if _, err := syscall.Write(fd, buf); err != nil {
		return fail(err)
	}
	if err := ioctl(fd, uiDevCreate, 0); err != nil {
		return fail(err)
	}
	return &uinputDev{fd: fd}, nil
}

// emit writes one event. Callers must hold d.mu.
func (d *uinputDev) emit(typ, code uint16, val int32) {
	ev := inputEvent{Type: typ, Code: code, Value: val}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&ev)), unsafe.Sizeof(ev))
	syscall.Write(d.fd, buf)
}

func (d *uinputDev) syn() { d.emit(evSyn, 0, 0) }

func (d *uinputDev) key(code uint16, down bool) {
	v := int32(0)
	if down {
		v = 1
	}
	d.emit(evKey, code, v)
	d.syn()
	time.Sleep(2 * time.Millisecond)
}

// ── Backend ──────────────────────────────────────────────────────────────────

type uinputBackend struct {
	kbd *uinputDev
	ptr *uinputDev
}

func newUinputBackend() (*uinputBackend, error) {
	buildKeymaps()

	var keys []uint16
	seen := map[uint16]bool{}
	add := func(c uint16) {
		if !seen[c] {
			seen[c] = true
			keys = append(keys, c)
		}
	}
	for _, c := range keyNames {
		add(c)
	}
	for _, c := range modKeys {
		add(c)
	}
	for _, ks := range asciiKeys {
		add(ks.code)
	}
	add(keyLeftShift)

	kbd, err := newUinputDev("CrossBoard virtual keyboard", []uint16{evKey}, keys, nil, nil)
	if err != nil {
		return nil, err
	}
	// Absolute pointer (like a VM "tablet"): the existing 0..1 protocol maps
	// straight onto it, and compositors place it across the whole desktop.
	ptr, err := newUinputDev("CrossBoard virtual pointer",
		[]uint16{evKey, evAbs, evRel},
		[]uint16{btnLeft, btnRight, btnMiddle},
		[]uint16{relWheel, relHWheel},
		[]uint16{absX, absY})
	if err != nil {
		syscall.Close(kbd.fd)
		return nil, err
	}
	// Give the compositor a moment to pick up the new devices, otherwise the
	// first events are lost.
	time.Sleep(300 * time.Millisecond)
	return &uinputBackend{kbd: kbd, ptr: ptr}, nil
}

func (b *uinputBackend) Name() string { return "uinput" }

func (b *uinputBackend) MoveAbs(nx, ny float64) {
	b.ptr.mu.Lock()
	defer b.ptr.mu.Unlock()
	b.ptr.emit(evAbs, absX, int32(nx*absRange))
	b.ptr.emit(evAbs, absY, int32(ny*absRange))
	b.ptr.syn()
}

func (b *uinputBackend) Click(button string) {
	code := btnLeft
	switch strings.ToLower(button) {
	case "right":
		code = btnRight
	case "middle":
		code = btnMiddle
	}
	b.ptr.mu.Lock()
	defer b.ptr.mu.Unlock()
	b.ptr.emit(evKey, code, 1)
	b.ptr.syn()
	time.Sleep(15 * time.Millisecond)
	b.ptr.emit(evKey, code, 0)
	b.ptr.syn()
}

func (b *uinputBackend) Scroll(dx, dy int) {
	if dx == 0 && dy == 0 {
		return
	}
	b.ptr.mu.Lock()
	defer b.ptr.mu.Unlock()
	if dy != 0 {
		b.ptr.emit(evRel, relWheel, int32(dy)) // + = up
	}
	if dx != 0 {
		b.ptr.emit(evRel, relHWheel, int32(dx)) // + = right (flip here if reversed)
	}
	b.ptr.syn()
}

func (b *uinputBackend) KeyTap(key string, mods ...string) {
	code, ok := lookupKey(key)
	if !ok {
		appLogf("input: unknown key %q", key)
		return
	}
	var modCodes []uint16
	for _, m := range mods {
		mc, ok := modKeys[strings.ToLower(m)]
		if !ok {
			appLogf("input: unknown modifier %q", m)
			return
		}
		modCodes = append(modCodes, mc)
	}
	b.kbd.mu.Lock()
	defer b.kbd.mu.Unlock()
	for _, mc := range modCodes {
		b.kbd.key(mc, true)
	}
	b.kbd.key(code, true)
	b.kbd.key(code, false)
	for i := len(modCodes) - 1; i >= 0; i-- {
		b.kbd.key(modCodes[i], false)
	}
}

// Type types US-layout ASCII. Other characters (e.g. Devanagari) cannot be
// expressed as raw key codes and are skipped.
func (b *uinputBackend) Type(text string) {
	buildKeymaps()
	b.kbd.mu.Lock()
	defer b.kbd.mu.Unlock()
	for _, r := range text {
		ks, ok := asciiKeys[r]
		if !ok {
			appLogf("input: cannot type %q with the uinput backend (ASCII only)", r)
			continue
		}
		if ks.shift {
			b.kbd.key(keyLeftShift, true)
		}
		b.kbd.key(ks.code, true)
		b.kbd.key(ks.code, false)
		if ks.shift {
			b.kbd.key(keyLeftShift, false)
		}
	}
}

// ── Key maps (US layout) ─────────────────────────────────────────────────────

type keyStroke struct {
	code  uint16
	shift bool
}

var (
	keymapOnce sync.Once
	keyNames   = map[string]uint16{}
	asciiKeys  = map[rune]keyStroke{}
	modKeys    = map[string]uint16{
		"ctrl": keyLeftCtrl, "control": keyLeftCtrl, "lctrl": keyLeftCtrl,
		"shift": keyLeftShift, "lshift": keyLeftShift,
		"alt": keyLeftAlt, "lalt": keyLeftAlt,
		"cmd": keyLeftMeta, "command": keyLeftMeta, "win": keyLeftMeta,
		"windows": keyLeftMeta, "super": keyLeftMeta, "meta": keyLeftMeta,
	}
)

// buildKeymaps is lazy (not init()) because the backend is created during
// package variable initialisation, before any init() function would run.
func buildKeymaps() {
	keymapOnce.Do(func() {
		rows := []struct {
			chars string
			start uint16
		}{{"1234567890", 2}, {"qwertyuiop", 16}, {"asdfghjkl", 30}, {"zxcvbnm", 44}}
		for _, row := range rows {
			for i, ch := range row.chars {
				code := row.start + uint16(i)
				asciiKeys[ch] = keyStroke{code, false}
				keyNames[string(ch)] = code
				if unicode.IsLetter(ch) {
					asciiKeys[unicode.ToUpper(ch)] = keyStroke{code, true}
				}
			}
		}

		plain := "-=[];'`\\,./"
		shifted := "_+{}:\"~|<>?"
		codes := []uint16{12, 13, 26, 27, 39, 40, 41, 43, 51, 52, 53}
		for i, c := range codes {
			asciiKeys[rune(plain[i])] = keyStroke{c, false}
			asciiKeys[rune(shifted[i])] = keyStroke{c, true}
			keyNames[string(plain[i])] = c
		}
		for i, ch := range "!@#$%^&*()" {
			asciiKeys[ch] = keyStroke{2 + uint16(i), true}
		}

		asciiKeys[' '] = keyStroke{57, false}
		asciiKeys['\n'] = keyStroke{28, false}
		asciiKeys['\t'] = keyStroke{15, false}

		named := map[string]uint16{
			"escape": 1, "esc": 1, "backspace": 14, "tab": 15,
			"enter": 28, "return": 28, "space": 57, "capslock": 58,
			"home": 102, "up": 103, "pageup": 104, "left": 105, "right": 106,
			"end": 107, "down": 108, "pagedown": 109, "insert": 110, "delete": 111,
			"audio_mute": 113, "audio_vol_down": 114, "audio_vol_up": 115,
			"audio_next": 163, "audio_play": 164, "audio_prev": 165,
			"brightness_down": 224, "brightness_up": 225,
			"f11": 87, "f12": 88,
		}
		for i := 1; i <= 10; i++ {
			named[fmt.Sprintf("f%d", i)] = uint16(58 + i)
		}
		for k, v := range named {
			keyNames[k] = v
		}
	})
}

func lookupKey(name string) (uint16, bool) {
	buildKeymaps()
	n := strings.ToLower(name)
	if c, ok := keyNames[n]; ok {
		return c, true
	}
	if c, ok := modKeys[n]; ok {
		return c, true
	}
	if r := []rune(name); len(r) == 1 {
		if ks, ok := asciiKeys[r[0]]; ok {
			return ks.code, true
		}
	}
	return 0, false
}
