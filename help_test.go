package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHelpPageRendersGuidesAndConfiguredPort(t *testing.T) {
	configState.Lock()
	previous := configState.config
	configState.config.ServerPort = 6060
	configState.Unlock()
	t.Cleanup(func() {
		configState.Lock()
		configState.config = previous
		configState.Unlock()
	})

	request := httptest.NewRequest(http.MethodGet, "/help", nil)
	response := httptest.NewRecorder()
	helpPage(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("help page status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("help page content type = %q, want HTML", got)
	}
	for _, want := range []string{
		"6060",
		"developers.cloudflare.com",
		"Genymobile/scrcpy",
		"New-NetFirewallRule",
		"sudo ufw allow",
		"Privacy &amp; Security",
		"rel=\"noopener noreferrer\"",
		"Alt+C",
		"Ctrl+Shift+C",
		"onclick=\"window.close()\"",
		">Close</button>",
	} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("help page is missing %q", want)
		}
	}
	if strings.Contains(response.Body.String(), "Open file browser") {
		t.Error("help page should not link to the server-hosted file browser")
	}
}

func TestHelpPageRejectsUnsupportedMethodAndPath(t *testing.T) {
	for _, test := range []struct {
		method string
		path   string
		want   int
	}{
		{method: http.MethodPost, path: "/help", want: http.StatusMethodNotAllowed},
		{method: http.MethodGet, path: "/help/other", want: http.StatusNotFound},
	} {
		request := httptest.NewRequest(test.method, test.path, nil)
		response := httptest.NewRecorder()
		helpPage(response, request)
		if response.Code != test.want {
			t.Errorf("%s %s status = %d, want %d", test.method, test.path, response.Code, test.want)
		}
	}
}
