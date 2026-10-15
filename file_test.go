package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizePathUsesConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	configState.Lock()
	previous := configState.config
	configState.config.FileBrowserDir = root
	configState.Unlock()
	t.Cleanup(func() {
		configState.Lock()
		configState.config = previous
		configState.Unlock()
	})

	if got, err := sanitizePath(""); err != nil || got != root {
		t.Fatalf("sanitizePath(root) = %q, %v; want %q, nil", got, err, root)
	}
	if got, err := sanitizePath("nested"); err != nil || got != filepath.Join(root, "nested") {
		t.Fatalf("sanitizePath(nested) = %q, %v", got, err)
	}
	if _, err := sanitizePath("../" + filepath.Base(outside)); err == nil {
		t.Fatal("sanitizePath accepted traversal outside the configured root")
	}
	if err := os.Symlink(outside, filepath.Join(root, "outside-link")); err == nil {
		if _, err := sanitizePath("outside-link/secret.txt"); err == nil {
			t.Fatal("sanitizePath followed a symlink outside the configured root")
		}
	}
}

func TestListFilesUsesConfiguredRootAndEscapesNames(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "notes & draft.txt"), []byte("text"), 0600); err != nil {
		t.Fatal(err)
	}
	configState.Lock()
	previous := configState.config
	configState.config.FileBrowserDir = root
	configState.Unlock()
	t.Cleanup(func() {
		configState.Lock()
		configState.config = previous
		configState.Unlock()
	})

	request := httptest.NewRequest(http.MethodGet, "/listfiles?folder=.", nil)
	response := httptest.NewRecorder()
	listFiles(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("list files status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	for _, want := range []string{"notes &amp; draft.txt", "notes%20&amp;%20draft.txt"} {
		if !strings.Contains(response.Body.String(), want) {
			t.Errorf("file listing is missing %q: %s", want, response.Body.String())
		}
	}
}
