package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDashboardShowsLANQRAndServerURLsByDefault(t *testing.T) {
	serverURLs.Lock()
	previousTunnel, previousLAN := serverURLs.tunnel, serverURLs.lan
	serverURLs.tunnel = "https://crossboard.example"
	serverURLs.lan = "http://192.0.2.10:5050"
	serverURLs.Unlock()
	t.Cleanup(func() {
		serverURLs.Lock()
		serverURLs.tunnel, serverURLs.lan = previousTunnel, previousLAN
		serverURLs.Unlock()
	})

	view := (dashboardModel{width: 100, height: 30, logs: []appLogEntry{}}).View()
	for _, want := range []string{
		"CROSSBOARD",
		"LAN · http://192.0.2.10:5050",
		"http://192.0.2.10:5050",
		"https://crossboard.example",
		"? see global shortcuts",
		"q quit",
		"█",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("dashboard view is missing %q", want)
		}
	}
	if strings.Contains(view, "Alt+C") || strings.Contains(view, "Alt+V") {
		t.Error("global shortcuts should be hidden until requested")
	}
}

func TestDashboardShortcutGuideShowsGlobalBindings(t *testing.T) {
	model := dashboardModel{width: 100, height: 30}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model = updated.(dashboardModel)
	view := model.View()
	for _, want := range []string{
		"GLOBAL SHORTCUTS",
		"Alt+C",
		"Alt+V",
		"Alt+K",
		"Ctrl+Shift+C",
		"copy selected PC text",
		"fetch app clipboard",
		"scrcpy",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("shortcut guide is missing %q", want)
		}
	}
	for _, excluded := range []string{"Meta+Alt+L", "Meta+Alt+R", "Meta+Alt+S", "Meta+Alt+F"} {
		if strings.Contains(view, excluded) {
			t.Errorf("global shortcut guide should not list %s", excluded)
		}
	}
}

func TestDashboardDoesNotShowHelpContentAndRemovesFileHotkey(t *testing.T) {
	model := dashboardModel{width: 100, height: 30}
	view := model.View()
	if strings.Contains(view, "f files") || strings.Contains(view, "Cloudflare: optional") {
		t.Fatal("dashboard should not display the file hotkey or FAQ content")
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model = updated.(dashboardModel)
	if model.status != "" {
		t.Fatalf("pressing f should not launch a browser action; status = %q", model.status)
	}
}

func TestDashboardCanShowCloudflareQRWithoutReplacingLANDefault(t *testing.T) {
	serverURLs.Lock()
	previousTunnel, previousLAN := serverURLs.tunnel, serverURLs.lan
	serverURLs.tunnel = "https://crossboard.example"
	serverURLs.lan = "http://192.0.2.10:5050"
	serverURLs.Unlock()
	t.Cleanup(func() {
		serverURLs.Lock()
		serverURLs.tunnel, serverURLs.lan = previousTunnel, previousLAN
		serverURLs.Unlock()
	})

	lanDefault := (dashboardModel{width: 100}).View()
	if !strings.Contains(lanDefault, "LAN · http://192.0.2.10:5050") {
		t.Fatal("default dashboard should show LAN QR")
	}
	tunnelView := (dashboardModel{width: 100, showTunnelQR: true}).View()
	if !strings.Contains(tunnelView, "CLOUDFLARE · https://crossboard.example") {
		t.Fatal("tunnel QR view should retain Cloudflare support")
	}
}

func TestDashboardLANQRContainsRenderedCode(t *testing.T) {
	serverURLs.Lock()
	previousTunnel, previousLAN := serverURLs.tunnel, serverURLs.lan
	serverURLs.tunnel = ""
	serverURLs.lan = "http://192.0.2.10:5050"
	serverURLs.Unlock()
	t.Cleanup(func() {
		serverURLs.Lock()
		serverURLs.tunnel, serverURLs.lan = previousTunnel, previousLAN
		serverURLs.Unlock()
	})

	qr := currentLANQRText()
	if !strings.Contains(qr, "LAN · http://192.0.2.10:5050") || !strings.Contains(qr, "█") {
		t.Fatalf("QR view is missing URL or rendered QR blocks: %q", qr)
	}
}

func TestDashboardRefreshShortcutRedrawsWithoutReprintingQR(t *testing.T) {
	serverURLs.Lock()
	previousTunnel, previousLAN := serverURLs.tunnel, serverURLs.lan
	serverURLs.tunnel = ""
	serverURLs.lan = "http://192.0.2.10:5050"
	serverURLs.Unlock()
	t.Cleanup(func() {
		serverURLs.Lock()
		serverURLs.tunnel, serverURLs.lan = previousTunnel, previousLAN
		serverURLs.Unlock()
	})

	model := dashboardModel{width: 100, status: "before"}
	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	model = updated.(dashboardModel)
	if model.status != "Dashboard refreshed" {
		t.Fatalf("refresh status = %q, want dashboard refresh confirmation", model.status)
	}
	if command == nil {
		t.Fatal("refresh shortcut should request a terminal redraw")
	}
	if view := model.View(); !strings.Contains(view, "LAN · http://192.0.2.10:5050") || !strings.Contains(view, "█") {
		t.Fatalf("refreshed dashboard is missing the LAN QR: %q", view)
	}
}
