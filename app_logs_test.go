package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAppLogEndpointReturnsRecentEntries(t *testing.T) {
	appLogStore.Lock()
	previousEntries := appLogStore.entries
	appLogStore.entries = nil
	appLogStore.Unlock()
	t.Cleanup(func() {
		appLogStore.Lock()
		appLogStore.entries = previousEntries
		appLogStore.Unlock()
	})

	appLogf("test log entry")
	request := httptest.NewRequest(http.MethodGet, "/api/logs", nil)
	response := httptest.NewRecorder()
	getAppLogs(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var entries []appLogEntry
	if err := json.NewDecoder(response.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Message != "test log entry" {
		t.Fatalf("entries = %#v, want the test log entry", entries)
	}
}

func TestAppLogStoreKeepsOnlyRecentEntries(t *testing.T) {
	appLogStore.Lock()
	previousEntries := appLogStore.entries
	appLogStore.entries = nil
	appLogStore.Unlock()
	t.Cleanup(func() {
		appLogStore.Lock()
		appLogStore.entries = previousEntries
		appLogStore.Unlock()
	})

	for i := 0; i < maxAppLogEntries+2; i++ {
		appLogf("entry %d", i)
	}
	appLogStore.RLock()
	defer appLogStore.RUnlock()
	if got := len(appLogStore.entries); got != maxAppLogEntries {
		t.Fatalf("log entry count = %d, want %d", got, maxAppLogEntries)
	}
	if got := appLogStore.entries[0].Message; got != "entry 2" {
		t.Fatalf("oldest retained log = %q, want entry 2", got)
	}
}
