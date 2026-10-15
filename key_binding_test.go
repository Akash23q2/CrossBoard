package main

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCaptureClipboardSelectionWaitsForCopyAndReturnsText(t *testing.T) {
	changes := make(chan []byte, 1)
	copied := false
	got, err := captureClipboardSelection(time.Second, changes, func() error {
		copied = true
		changes <- []byte("selected text")
		return nil
	})
	if err != nil {
		t.Fatalf("captureClipboardSelection() error = %v", err)
	}
	if !copied || got != "selected text" {
		t.Fatalf("captureClipboardSelection() = %q, copied=%v", got, copied)
	}
}

func TestCaptureClipboardSelectionHonorsTimeout(t *testing.T) {
	_, err := captureClipboardSelection(10*time.Millisecond, make(chan []byte), func() error { return nil })
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("captureClipboardSelection() error = %v, want timeout", err)
	}
}

func TestCaptureClipboardSelectionReturnsCopyFailure(t *testing.T) {
	expected := errors.New("injection failed")
	_, err := captureClipboardSelection(time.Second, make(chan []byte), func() error { return expected })
	if !errors.Is(err, expected) {
		t.Fatalf("captureClipboardSelection() error = %v, want %v", err, expected)
	}
}

func TestWriteClipboardThenPasteWritesBeforePasting(t *testing.T) {
	var order []string
	err := writeClipboardThenPaste("device text", func(content string) error {
		if content != "device text" {
			t.Errorf("clipboard write content = %q", content)
		}
		order = append(order, "write")
		return nil
	}, func() error {
		order = append(order, "paste")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(order, ",") != "write,paste" {
		t.Fatalf("operation order = %v, want [write paste]", order)
	}
}

func TestWriteClipboardThenPasteDoesNotPasteAfterWriteFailure(t *testing.T) {
	pasted := false
	expected := errors.New("clipboard write failed")
	err := writeClipboardThenPaste("device text", func(string) error { return expected }, func() error {
		pasted = true
		return nil
	})
	if !errors.Is(err, expected) {
		t.Fatalf("writeClipboardThenPaste() error = %v, want %v", err, expected)
	}
	if pasted {
		t.Fatal("paste was triggered after clipboard write failed")
	}
}

func TestClipboardDeviceHTTPContract(t *testing.T) {
	const clipboardText = "text from the PC"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/postclipboard":
			if r.Method != http.MethodPost {
				t.Errorf("POST endpoint method = %s, want POST", r.Method)
			}
			if got := r.Header.Get("Content-Type"); got != "text/plain; charset=utf-8" {
				t.Errorf("POST content type = %q", got)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read POST body: %v", err)
			}
			if string(body) != clipboardText {
				t.Errorf("POST body = %q, want %q", body, clipboardText)
			}
		case "/getclipboard":
			if r.Method != http.MethodGet {
				t.Errorf("GET endpoint method = %s, want GET", r.Method)
			}
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, "text from the phone")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if err := postClipboardToDevice(server.Client(), server.URL+"/postclipboard", clipboardText); err != nil {
		t.Fatalf("postClipboardToDevice() error = %v", err)
	}
	got, err := getClipboardFromDevice(server.Client(), server.URL+"/getclipboard")
	if err != nil {
		t.Fatalf("getClipboardFromDevice() error = %v", err)
	}
	if string(got) != "text from the phone" {
		t.Fatalf("getClipboardFromDevice() = %q", got)
	}
}

func TestGetClipboardFromDeviceRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", maxDeviceClipboard+1))
	}))
	defer server.Close()

	if _, err := getClipboardFromDevice(server.Client(), server.URL); err == nil {
		t.Fatal("getClipboardFromDevice() accepted a response larger than 1 MiB")
	}
}

func TestDeviceEndpointUsesConfiguredPort(t *testing.T) {
	configState.Lock()
	previous := configState.config
	configState.config.DevicePort = 8080
	configState.Unlock()
	t.Cleanup(func() {
		configState.Lock()
		configState.config = previous
		configState.Unlock()
	})
	got, err := deviceEndpoint("192.0.2.30", "getclipboard")
	if err != nil {
		t.Fatal(err)
	}
	if want := "http://192.0.2.30:8080/getclipboard"; got != want {
		t.Fatalf("deviceEndpoint() = %q, want %q", got, want)
	}
}
