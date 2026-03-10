package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/crateos/crateos/internal/config"
)

type userReq struct {
	TargetName string   `json:"target_name"`
	Name       string   `json:"name"`
	Role       string   `json:"role"`
	Perms      []string `json:"permissions"`
}

type bootstrapReq struct {
	AdminName string `json:"admin_name"`
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	cfg, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "users.view") && !authz.Check(user, "users.*") && !authz.Check(user, "sys.manage") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	writeJSON(w, cfg.Users)
}

func handleUserAdd(w http.ResponseWriter, r *http.Request) {
	cfg, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "users.edit") && !authz.Check(user, "users.*") && !authz.Check(user, "sys.manage") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req userReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Role == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	for _, u := range cfg.Users.Users {
		if u.Name == req.Name {
			http.Error(w, "user already exists", http.StatusConflict)
			return
		}
	}
	cfg.Users.Users = append(cfg.Users.Users, config.UserEntry{
		Name:        req.Name,
		Role:        req.Role,
		Permissions: req.Perms,
	})
	if err := config.SaveUsers(cfg); err != nil {
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "added", "name": req.Name})
}

func handleUserDelete(w http.ResponseWriter, r *http.Request) {
	cfg, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "users.edit") && !authz.Check(user, "users.*") && !authz.Check(user, "sys.manage") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req userReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	var filtered []config.UserEntry
	for _, u := range cfg.Users.Users {
		if u.Name != req.Name {
			filtered = append(filtered, u)
		}
	}
	cfg.Users.Users = filtered
	if err := config.SaveUsers(cfg); err != nil {
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "deleted"})
}

func handleUserUpdate(w http.ResponseWriter, r *http.Request) {
	cfg, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "users.edit") && !authz.Check(user, "users.*") && !authz.Check(user, "sys.manage") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req userReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	target := strings.TrimSpace(req.TargetName)
	if target == "" {
		target = strings.TrimSpace(req.Name)
	}
	if target == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	nextName := strings.TrimSpace(req.Name)
	finalName := target
	updated := false
	for i := range cfg.Users.Users {
		if cfg.Users.Users[i].Name == target {
			if nextName != "" && nextName != target {
				for j := range cfg.Users.Users {
					if j != i && cfg.Users.Users[j].Name == nextName {
						http.Error(w, "user already exists", http.StatusConflict)
						return
					}
				}
				cfg.Users.Users[i].Name = nextName
				finalName = nextName
			}
			if req.Role != "" {
				cfg.Users.Users[i].Role = req.Role
			}
			if req.Perms != nil {
				cfg.Users.Users[i].Permissions = req.Perms
			}
			updated = true
			break
		}
	}
	if !updated {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err := config.SaveUsers(cfg); err != nil {
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "updated", "name": finalName})
}

func handleBootstrap(w http.ResponseWriter, r *http.Request) {
	cfg, _, _ := loadAuth(r)
	if cfg == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if len(cfg.Users.Users) > 0 {
		http.Error(w, "already initialized", http.StatusConflict)
		return
	}
	var req bootstrapReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.AdminName) == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if cfg.Users.Roles == nil {
		cfg.Users.Roles = map[string]config.Role{}
	}
	if _, ok := cfg.Users.Roles["admin"]; !ok {
		cfg.Users.Roles["admin"] = config.Role{
			Description: "Full platform access including break-glass shell",
			Permissions: []string{"*"},
		}
	}
	cfg.Users.Users = append(cfg.Users.Users, config.UserEntry{
		Name: req.AdminName,
		Role: "admin",
	})
	if err := config.SaveUsers(cfg); err != nil {
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "bootstrapped", "admin": req.AdminName})
}
