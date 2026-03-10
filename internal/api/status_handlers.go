package api

import (
	"net/http"

	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/state"
	"github.com/crateos/crateos/internal/sysinfo"
)

func handleStatus(w http.ResponseWriter, r *http.Request) {
	cfg, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "sys.view") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	info := sysinfo.Gather()
	svcNames := state.CollectServiceNames(cfg)
	actual := state.Probe(svcNames)

	writeJSON(w, map[string]interface{}{
		"version":     platform.Version,
		"sysinfo":     info,
		"state":       actual,
		"platform":    state.LoadPlatformState(),
		"diagnostics": buildDiagnosticsView(),
	})
}

func handleActorDiagnostics(w http.ResponseWriter, r *http.Request) {
	_, authz, user := loadAuth(r)
	if authz == nil {
		http.Error(w, "config load failed", http.StatusInternalServerError)
		return
	}
	if !authz.Check(user, "sys.view") {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	writeJSON(w, map[string]interface{}{
		"ownership": loadOwnershipDiagnostics(),
	})
}
