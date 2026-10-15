package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const maxAppLogEntries = 300

type appLogEntry struct {
	Time    time.Time `json:"time"`
	Message string    `json:"message"`
}

var appLogStore = struct {
	sync.RWMutex
	entries []appLogEntry
}{}

var terminalDashboardActive atomic.Bool

func appLogf(format string, args ...any) {
	entry := appLogEntry{
		Time:    time.Now(),
		Message: fmt.Sprintf(format, args...),
	}
	appLogStore.Lock()
	appLogStore.entries = append(appLogStore.entries, entry)
	if len(appLogStore.entries) > maxAppLogEntries {
		appLogStore.entries = appLogStore.entries[len(appLogStore.entries)-maxAppLogEntries:]
	}
	appLogStore.Unlock()
	if !terminalDashboardActive.Load() {
		fmt.Printf("%s %s\n", entry.Time.Format(time.DateTime), entry.Message)
	}
}

func recentAppLogs() []appLogEntry {
	appLogStore.RLock()
	defer appLogStore.RUnlock()
	return append([]appLogEntry(nil), appLogStore.entries...)
}

func logsPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/logs" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(logsHTML))
}

func getAppLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	appLogStore.RLock()
	entries := append([]appLogEntry(nil), appLogStore.entries...)
	appLogStore.RUnlock()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(entries)
}

const logsHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>CrossBoard logs</title>
<style>
body{margin:0;padding:24px;background:#111118;color:#e8e8f0;font:14px ui-monospace,monospace}
h1{color:#00ffe0}#logs{white-space:pre-wrap;overflow-wrap:anywhere;line-height:1.5}
.entry{padding:7px 0;border-bottom:1px solid #292936}.time{color:#9292a8;margin-right:12px}
</style>
</head>
<body><h1>CrossBoard logs</h1><div id="logs">Waiting for log entries…</div>
<script>
async function refresh(){try{const response=await fetch('/api/logs');if(!response.ok)throw Error(response.status);const entries=await response.json();const root=document.getElementById('logs');root.replaceChildren(...entries.map(entry=>{const row=document.createElement('div');row.className='entry';const time=document.createElement('span');time.className='time';time.textContent=new Date(entry.time).toLocaleTimeString();row.append(time,document.createTextNode(entry.message));return row;}));window.scrollTo(0,document.body.scrollHeight);}catch(error){document.getElementById('logs').textContent='Could not load logs: '+error;}}
refresh();setInterval(refresh,1500);
</script></body></html>`
