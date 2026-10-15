// for handling file sharing operations
package main

import (
	"encoding/json"
	"fmt"
	"html"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// sanitizePath ensures the path is within the configured file-browser directory.
func sanitizePath(reqPath string) (string, error) {
	root, err := fileBrowserRoot()
	if err != nil {
		return "", err
	}
	return sanitizePathInRoot(root, reqPath)
}

func fileBrowserRoot() (string, error) {
	root := currentAppConfig().FileBrowserDir
	if root == "" {
		root = appHome
	}
	root, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", fmt.Errorf("resolve file browser directory: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve file browser directory: %w", err)
	}
	return root, nil
}

func sanitizePathInRoot(root, reqPath string) (string, error) {
	if reqPath == "" || reqPath == "." {
		return root, nil
	}
	fullPath := filepath.Clean(filepath.Join(root, reqPath))

	if !isPathWithin(root, fullPath) {
		return "", fmt.Errorf("access denied")
	}
	resolved, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve requested path: %w", err)
	}
	if !isPathWithin(root, resolved) {
		return "", fmt.Errorf("access denied")
	}
	return resolved, nil
}

func isPathWithin(root, path string) bool {
	relative, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) &&
		!filepath.IsAbs(relative)
}

func getFile(w http.ResponseWriter, r *http.Request) {
	filePath := strings.TrimPrefix(r.URL.EscapedPath(), "/getfile/")
	decodedPath, err := url.PathUnescape(filePath)
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
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filepath.Base(fullPath)}))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, fullPath)
}

func listFiles(w http.ResponseWriter, r *http.Request) {
	root, err := fileBrowserRoot()
	if err != nil {
		http.Error(w, "File browser folder is unavailable", http.StatusInternalServerError)
		return
	}
	folder := r.URL.Query().Get("folder")
	path, err := sanitizePathInRoot(root, folder)
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
	relPath, _ := filepath.Rel(filepath.Clean(root), filepath.Clean(path))

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
		displayName := html.EscapeString(name)

		if entry.IsDir() {
			onclick := html.EscapeString("browseFiles(" + strconv.Quote(subPath) + "); return false;")
			w.Write([]byte(`<li><a href="#" class="file-link-dir" onclick="` + onclick + `">📁 ` + displayName + `/</a></li>`))
		} else {
			escapedPath := html.EscapeString(strings.Join(escapeURLPathSegments(subPath), "/"))
			w.Write([]byte(`<li><a href="/getfile/` + escapedPath + `" class="file-link" download>📄 ` + displayName + `</a></li>`))
		}
	}
	w.Write([]byte(`</ul>`))
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := r.ParseMultipartForm(5 << 30) // 5 GB
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
	filename := filepath.Base(strings.ReplaceAll(handler.Filename, "\\", "/"))
	if filename == "." || filename == string(filepath.Separator) || filename == "" {
		http.Error(w, "Invalid file name", http.StatusBadRequest)
		return
	}
	destPath := filepath.Join(path, filename)
	if info, err := os.Lstat(destPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
		http.Error(w, "Cannot overwrite a symbolic link", http.StatusForbidden)
		return
	} else if err != nil && !os.IsNotExist(err) {
		http.Error(w, "Could not inspect destination", http.StatusInternalServerError)
		return
	}
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
	_ = json.NewEncoder(w).Encode(struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}{true, "File uploaded: " + filename})
}

func escapeURLPathSegments(path string) []string {
	segments := strings.Split(path, "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return segments
}
