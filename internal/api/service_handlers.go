package api

import (
	"encoding/json"
	"net/http"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
	"github.com/crateos/crateos/internal/state"
)

type svcReq struct {
	Name string `json:"name"`
}

func handleServices(w http.ResponseWriter, r *http.Request) {
	cfg, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "svc.list") && !authz.Check(user, "svc.*") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	svcNames := state.CollectServiceNames(cfg)
	actual := state.Probe(svcNames)
	mods := modules.LoadAll(".")
	writeJSON(w, map[string]interface{}{
		"desired":  cfg.Services.Services,
		"actual":   actual.Services,
		"services": buildServiceView(cfg, actual, mods),
	})
}

func handleServiceEnable(w http.ResponseWriter, r *http.Request) {
	cfg, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "svc.*") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req svcReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	mods := modules.LoadAll(".")
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == req.Name {
			cfg.Services.Services[i].Enabled = true
			cfg.Services.Services[i].Autostart = shouldAutostartOnEnable(req.Name, mods)
			_ = config.SaveServices(cfg)
			applyServiceAction(req.Name, serviceActionEnableOnly, mods)
			_ = state.RefreshCrateState(req.Name)
			writeJSON(w, map[string]string{"status": "enabled"})
			return
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func handleServiceDisable(w http.ResponseWriter, r *http.Request) {
	cfg, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "svc.*") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req svcReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	mods := modules.LoadAll(".")
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == req.Name {
			cfg.Services.Services[i].Enabled = false
			cfg.Services.Services[i].Autostart = false
			_ = config.SaveServices(cfg)
			applyServiceAction(req.Name, serviceActionDisable, mods)
			_ = state.RefreshCrateState(req.Name)
			writeJSON(w, map[string]string{"status": "disabled"})
			return
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func handleServiceStart(w http.ResponseWriter, r *http.Request) {
	cfg, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "svc.*") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req svcReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	mods := modules.LoadAll(".")
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == req.Name {
			cfg.Services.Services[i].Enabled = true
			cfg.Services.Services[i].Autostart = true
			_ = config.SaveServices(cfg)
			applyServiceAction(req.Name, serviceActionStart, mods)
			_ = state.RefreshCrateState(req.Name)
			writeJSON(w, map[string]string{"status": "started"})
			return
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
}

func handleServiceStop(w http.ResponseWriter, r *http.Request) {
	cfg, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "svc.*") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req svcReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	mods := modules.LoadAll(".")
	for i := range cfg.Services.Services {
		if cfg.Services.Services[i].Name == req.Name {
			cfg.Services.Services[i].Enabled = true
			cfg.Services.Services[i].Autostart = false
			_ = config.SaveServices(cfg)
			applyServiceAction(req.Name, serviceActionStop, mods)
			_ = state.RefreshCrateState(req.Name)
			writeJSON(w, map[string]string{"status": "stopped"})
			return
		}
	}
	http.Error(w, "not found", http.StatusNotFound)
}
