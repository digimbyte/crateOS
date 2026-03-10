package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/config"
)

type ftpUploadCompleteReq struct {
	Path string `json:"path"`
}

func handleFTPUploadComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, authz, user := loadAuth(r)
	if authz == nil || cfg == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "sys.manage") && !authz.Check(user, "sys.*") && !authz.Check(user, "*") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req ftpUploadCompleteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	target := filepath.Clean(req.Path)
	time.Sleep(1 * time.Second)
	info, err := os.Stat(target)
	if err != nil {
		http.Error(w, "upload target not found", http.StatusNotFound)
		return
	}
	targetType := "file"
	if info.IsDir() {
		targetType = "directory"
	}
	normalizedCount, scannedCount, err := config.OnFTPUploadCompleteTarget(target)
	if err != nil {
		http.Error(w, "normalization failed", http.StatusInternalServerError)
		return
	}
	status := "skipped"
	if normalizedCount > 0 {
		status = "normalized"
	}
	writeJSON(w, map[string]interface{}{
		"status":           status,
		"path":             target,
		"target_type":      targetType,
		"normalized_count": normalizedCount,
		"scanned_count":    scannedCount,
	})
}
