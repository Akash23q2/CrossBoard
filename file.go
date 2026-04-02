// for handling file sharing operations
package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// sanitizePath ensures the path is within appHome and prevents directory traversal
func sanitizePath(reqPath string) (string, error) {
	if reqPath == "" || reqPath == "." {
		return appHome, nil
	}
	// Join with appHome and resolve to absolute clean path
	fullPath := filepath.Clean(filepath.Join(appHome, reqPath))
	appHomeClean := filepath.Clean(appHome)

	// Make sure it's within appHome (check exact match or prefix with separator)
	if fullPath != appHomeClean && !strings.HasPrefix(fullPath+string(os.PathSeparator), appHomeClean+string(os.PathSeparator)) {
		return "", fmt.Errorf("access denied")
	}
	return fullPath, nil
}

func getFile(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path
	filePath = strings.TrimPrefix(filePath, "/getfile/")
	// URL decode the path
	decodedPath, err := url.QueryUnescape(filePath)
	if err != nil {
		http.Error(w, "Invalid path", 400)
		return
	}
	// Convert forward slashes to OS-specific separator
	decodedPath = filepath.FromSlash(decodedPath)
	fullPath, err := sanitizePath(decodedPath)
	if err != nil {
		http.Error(w, "Access denied", 403)
		return
	}
	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil {
		http.Error(w, "File not found", 404)
		return
	}
	_ = info // use info to avoid unused variable warning

	// Set headers to force download
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(fullPath)+"\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, fullPath)
}

func listFiles(w http.ResponseWriter, r *http.Request) {
	folder := r.URL.Query().Get("folder")
	path, err := sanitizePath(folder)
	if err != nil {
		http.Error(w, "Access denied", 403)
		return
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<ul>`))
	// Get relative path from appHome
	relPath, _ := filepath.Rel(filepath.Clean(appHome), filepath.Clean(path))

	for _, entry := range entries {
		name := entry.Name()
		var subPath string

		// If at root, use name directly; otherwise join with relPath
		if relPath == "." {
			subPath = name
		} else {
			subPath = filepath.Join(relPath, name)
		}

		// Convert to forward slashes for web
		subPath = filepath.ToSlash(subPath)

		if entry.IsDir() {
			w.Write([]byte(`<li><a href="#" class="file-link-dir" onclick="browseFiles('` + subPath + `'); return false;">📁 ` + name + `/</a></li>`))
		} else {
			w.Write([]byte(`<li><a href="/getfile/` + subPath + `" class="file-link" download>📄 ` + name + `</a></li>`))
		}
	}
	w.Write([]byte(`</ul>`))
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseMultipartForm(500 << 20) // 500 MB
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	folder := r.FormValue("folder")
	path, err := sanitizePath(folder)
	if err != nil {
		http.Error(w, "Access denied", 403)
		return
	}
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Failed to get file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	destPath := filepath.Join(path, handler.Filename)
	dst, err := os.Create(destPath)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	_, err = dst.ReadFrom(file)
	if err != nil {
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"success": true, "message": "File uploaded: %s"}`, handler.Filename)
}
