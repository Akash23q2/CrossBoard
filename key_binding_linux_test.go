//go:build linux

package main

import "testing"

func TestGlobalKeyStateMatchesClipboardShortcuts(t *testing.T) {
	pressed := make(map[uint16]bool)
	if action := updateGlobalKeyState(pressed, evdevKeyC, evdevKeyDown); action != "" {
		t.Fatalf("plain C action = %q, want none", action)
	}
	updateGlobalKeyState(pressed, evdevAltL, evdevKeyDown)
	if action := updateGlobalKeyState(pressed, evdevKeyC, evdevKeyDown); action != "send" {
		t.Fatalf("Alt+C action = %q, want send", action)
	}
	if action := updateGlobalKeyState(pressed, evdevKeyC, 2); action != "" {
		t.Fatalf("key repeat action = %q, want none", action)
	}
	updateGlobalKeyState(pressed, evdevKeyC, evdevKeyUp)
	if action := updateGlobalKeyState(pressed, evdevKeyV, evdevKeyDown); action != "receive" {
		t.Fatalf("Alt+V action = %q, want receive", action)
	}
	updateGlobalKeyState(pressed, evdevAltL, evdevKeyUp)
	if action := updateGlobalKeyState(pressed, evdevKeyC, evdevKeyDown); action != "" {
		t.Fatalf("shortcut without Alt action = %q, want none", action)
	}
	pressed[evdevCtrlL] = true
	pressed[evdevMetaL] = true
	if action := updateGlobalKeyState(pressed, evdevKeyC, evdevKeyDown); action != "" {
		t.Fatalf("Meta+Ctrl+C action = %q, want none", action)
	}
}

func TestGlobalKeyStateMatchesAltScrcpyShortcut(t *testing.T) {
	pressed := map[uint16]bool{evdevAltR: true}
	if got := updateGlobalKeyState(pressed, evdevKeyK, evdevKeyDown); got != "scrcpy" {
		t.Fatalf("Alt+K action = %q, want scrcpy", got)
	}
	delete(pressed, evdevAltR)
	if got := updateGlobalKeyState(pressed, evdevKeyK, evdevKeyDown); got != "" {
		t.Fatalf("plain K action = %q, want none", got)
	}
	pressed[evdevMetaL] = true
	pressed[evdevAltR] = true
	if got := updateGlobalKeyState(pressed, evdevKeyK, evdevKeyDown); got != "scrcpy" {
		t.Fatalf("Meta+Alt+K action = %q, want scrcpy", got)
	}
	delete(pressed, evdevMetaL)
	for _, test := range []struct {
		key  uint16
		want string
	}{{evdevKeyC, "send"}, {evdevKeyV, "receive"}} {
		if got := updateGlobalKeyState(pressed, test.key, evdevKeyDown); got != test.want {
			t.Errorf("Alt+%d action = %q, want %q", test.key, got, test.want)
		}
	}
}
