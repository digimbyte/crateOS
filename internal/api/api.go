package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/auth"
	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/state"
	"github.com/crateos/crateos/internal/sysinfo"
)

// Server wraps the HTTP server on a Unix socket.
type Server struct {
	srv      *http.Server
	listener net.Listener
}

// Start launches the API server on the CrateOS agent socket.
func Start() (*Server, error) {
	_ = os.Remove(platform.AgentSocket)
	if err := os.MkdirAll(platform.CratePath("runtime"), 0755); err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/status", handleStatus)
	mux.HandleFunc("/diagnostics/actors", handleActorDiagnostics)
	mux.HandleFunc("/services", handleServices)
	mux.HandleFunc("/services/enable", handleServiceEnable)
	mux.HandleFunc("/services/disable", handleServiceDisable)
	mux.HandleFunc("/services/start", handleServiceStart)
	mux.HandleFunc("/services/stop", handleServiceStop)
	mux.HandleFunc("/users", handleUsers)
	mux.HandleFunc("/users/add", handleUserAdd)
	mux.HandleFunc("/users/delete", handleUserDelete)
	mux.HandleFunc("/users/update", handleUserUpdate)
	mux.HandleFunc("/bootstrap", handleBootstrap)
	mux.HandleFunc("/uploads/ftp/complete", handleFTPUploadComplete)

	srv := &http.Server{Handler: mux}
	ln, err := net.Listen("unix", platform.AgentSocket)
	if err != nil {
		return nil, err
	}

	go srv.Serve(ln)

	return &Server{srv: srv, listener: ln}, nil
}

// Stop closes the server.
func (s *Server) Stop() {
	if s == nil {
		return
	}
	_ = s.srv.Close()
	_ = s.listener.Close()
}

// ---- Handlers ------------------------------------------------------

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

type svcReq struct {
	Name string `json:"name"`
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

type userReq struct {
	TargetName string   `json:"target_name"`
	Name       string   `json:"name"`
	Role       string   `json:"role"`
	Perms      []string `json:"permissions"`
}
type bootstrapReq struct {
	AdminName string `json:"admin_name"`
}
type ftpUploadCompleteReq struct {
	Path string `json:"path"`
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

// ---- Helpers -------------------------------------------------------

func loadAuth(r *http.Request) (*config.Config, *auth.Authz, string) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, ""
	}
	authz := auth.Load(cfg)
	user := r.Header.Get("X-CrateOS-User")
	if user == "" {
		// default to the first user in config if none provided
		if len(cfg.Users.Users) > 0 {
			user = cfg.Users.Users[0].Name
		}
	}
	return cfg, authz, strings.TrimSpace(user)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func systemctlNoError(action, unit string) {
	if runtime.GOOS != "linux" {
		return
	}
	_ = exec.Command("systemctl", action, unit).Run()
}

type serviceView struct {
	Name                       string     `json:"name"`
	DisplayName                string     `json:"display_name"`
	Category                   string     `json:"category,omitempty"`
	Description                string     `json:"description,omitempty"`
	Runtime                    string     `json:"runtime"`
	InstallMode                string     `json:"install_mode,omitempty"`
	ActorName                  string     `json:"actor_name,omitempty"`
	ActorType                  string     `json:"actor_type,omitempty"`
	ActorID                    string     `json:"actor_id,omitempty"`
	ActorUser                  string     `json:"actor_user,omitempty"`
	ActorGroup                 string     `json:"actor_group,omitempty"`
	ActorHome                  string     `json:"actor_home,omitempty"`
	ActorRuntimeDir            string     `json:"actor_runtime_dir,omitempty"`
	ActorStateDir              string     `json:"actor_state_dir,omitempty"`
	ActorProvisioning          string     `json:"actor_provisioning,omitempty"`
	ActorProvisioningError     string     `json:"actor_provisioning_error,omitempty"`
	ActorProvisioningUpdatedAt string     `json:"actor_provisioning_updated_at,omitempty"`
	ActorProvisioningStatePath string     `json:"actor_provisioning_state_path,omitempty"`
	ActorOwnershipStatus       string     `json:"actor_ownership_status,omitempty"`
	ActorOwnershipUpdatedAt    string     `json:"actor_ownership_updated_at,omitempty"`
	ActorOwnershipRetiredAt    string     `json:"actor_ownership_retired_at,omitempty"`
	DeploySource               string     `json:"deploy_source,omitempty"`
	UploadPath                 string     `json:"upload_path,omitempty"`
	WorkingDir                 string     `json:"working_dir,omitempty"`
	Entry                      string     `json:"entry,omitempty"`
	InstallCommand             string     `json:"install_command,omitempty"`
	EnvironmentFile            string     `json:"environment_file,omitempty"`
	ExecutionMode              string     `json:"execution_mode,omitempty"`
	ExecutionAdapter           string     `json:"execution_adapter,omitempty"`
	ExecutionStatus            string     `json:"execution_status,omitempty"`
	PrimaryUnit                string     `json:"primary_unit,omitempty"`
	CompanionUnit              string     `json:"companion_unit,omitempty"`
	PrimaryUnitPath            string     `json:"primary_unit_path,omitempty"`
	CompanionUnitPath          string     `json:"companion_unit_path,omitempty"`
	ExecutionStatePath         string     `json:"execution_state_path,omitempty"`
	StartCommand               string     `json:"start_command,omitempty"`
	Schedule                   string     `json:"schedule,omitempty"`
	Timeout                    string     `json:"timeout,omitempty"`
	StopTimeout                string     `json:"stop_timeout,omitempty"`
	OnTimeout                  string     `json:"on_timeout,omitempty"`
	KillSignal                 string     `json:"kill_signal,omitempty"`
	ConcurrencyPolicy          string     `json:"concurrency_policy,omitempty"`
	ExecutionSummary           string     `json:"execution_summary,omitempty"`
	Stateful                   bool       `json:"stateful,omitempty"`
	DataPath                   string     `json:"data_path,omitempty"`
	NativeDataPath             string     `json:"native_data_path,omitempty"`
	StorageSummary             string     `json:"storage_summary,omitempty"`
	Desired                    bool       `json:"desired"`
	Autostart                  bool       `json:"autostart"`
	Active                     bool       `json:"active"`
	Enabled                    bool       `json:"enabled"`
	Status                     string     `json:"status"`
	Health                     string     `json:"health"`
	Module                     bool       `json:"module"`
	Ready                      bool       `json:"ready"`
	PackagesInstalled          bool       `json:"packages_installed"`
	MissingPackages            []string   `json:"missing_packages,omitempty"`
	Summary                    string     `json:"summary,omitempty"`
	LastError                  string     `json:"last_error,omitempty"`
	LastAction                 string     `json:"last_action,omitempty"`
	LastActionAt               string     `json:"last_action_at,omitempty"`
	SuggestedRepair            string     `json:"suggested_repair,omitempty"`
	LastGoodStatus             string     `json:"last_good_status,omitempty"`
	LastGoodHealth             string     `json:"last_good_health,omitempty"`
	LastGoodAt                 string     `json:"last_good_at,omitempty"`
	LastGoodSummary            string     `json:"last_good_summary,omitempty"`
	Units                      []unitView `json:"units,omitempty"`
	Packages                   []string   `json:"packages,omitempty"`
}

type unitView struct {
	Name    string `json:"name"`
	Active  bool   `json:"active"`
	Enabled bool   `json:"enabled"`
	Status  string `json:"status"`
	Health  string `json:"health"`
}

type diagnosticsView struct {
	Config       configDiagnosticsView       `json:"config"`
	Verification verificationDiagnosticsView `json:"verification"`
	Ownership    ownershipDiagnosticsView    `json:"ownership"`
}

type configDiagnosticsView struct {
	GeneratedAt   string                     `json:"generated_at,omitempty"`
	Tracked       int                        `json:"tracked"`
	Monitored     int                        `json:"monitored"`
	Unmonitored   int                        `json:"unmonitored"`
	ExternalEdits int                        `json:"external_edits"`
	Files         []configDiagnosticFileView `json:"files,omitempty"`
}

type configDiagnosticFileView struct {
	File          string `json:"file"`
	Path          string `json:"path"`
	Exists        bool   `json:"exists"`
	Monitoring    string `json:"monitoring"`
	LastWriter    string `json:"last_writer,omitempty"`
	LastSeenAt    string `json:"last_seen_at,omitempty"`
	LastChangedAt string `json:"last_changed_at,omitempty"`
}

type verificationDiagnosticsView struct {
	Status         string   `json:"status"`
	Summary        string   `json:"summary,omitempty"`
	Missing        []string `json:"missing,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
	PlatformState  string   `json:"platform_state,omitempty"`
	Readiness      string   `json:"readiness,omitempty"`
	StorageState   string   `json:"storage_state,omitempty"`
	OwnershipState string   `json:"ownership_state,omitempty"`
	AgentSocket    bool     `json:"agent_socket"`
	AdminPresent   bool     `json:"admin_present"`
}

type ownershipDiagnosticsView struct {
	GeneratedAt string                         `json:"generated_at,omitempty"`
	Managed     int                            `json:"managed"`
	Provisioned int                            `json:"provisioned"`
	Pending     int                            `json:"pending"`
	Blocked     int                            `json:"blocked"`
	Active      int                            `json:"active"`
	Retired     int                            `json:"retired"`
	Claims      []ownershipDiagnosticClaimView `json:"claims,omitempty"`
	Workloads   []actorLifecycleDiagnosticView `json:"workloads,omitempty"`
}

type ownershipDiagnosticClaimView struct {
	Crate     string `json:"crate"`
	Name      string `json:"name,omitempty"`
	Type      string `json:"type,omitempty"`
	ID        string `json:"id,omitempty"`
	User      string `json:"user,omitempty"`
	Group     string `json:"group,omitempty"`
	Home      string `json:"home,omitempty"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at,omitempty"`
	RetiredAt string `json:"retired_at,omitempty"`
}

type actorLifecycleDiagnosticView struct {
	Crate                 string                              `json:"crate"`
	ActorName             string                              `json:"actor_name,omitempty"`
	ActorType             string                              `json:"actor_type,omitempty"`
	ActorID               string                              `json:"actor_id,omitempty"`
	ActorUser             string                              `json:"actor_user,omitempty"`
	ActorGroup            string                              `json:"actor_group,omitempty"`
	ActorHome             string                              `json:"actor_home,omitempty"`
	Provisioning          string                              `json:"provisioning,omitempty"`
	ProvisioningError     string                              `json:"provisioning_error,omitempty"`
	ProvisioningUpdatedAt string                              `json:"provisioning_updated_at,omitempty"`
	LastSuccessAt         string                              `json:"last_success_at,omitempty"`
	LastFailureAt         string                              `json:"last_failure_at,omitempty"`
	ProvisioningStatePath string                              `json:"provisioning_state_path,omitempty"`
	OwnershipStatus       string                              `json:"ownership_status,omitempty"`
	OwnershipUpdatedAt    string                              `json:"ownership_updated_at,omitempty"`
	OwnershipRetiredAt    string                              `json:"ownership_retired_at,omitempty"`
	RecentEvents          []actorLifecycleEventDiagnosticView `json:"recent_events,omitempty"`
}

type actorLifecycleEventDiagnosticView struct {
	At           string `json:"at,omitempty"`
	Provisioning string `json:"provisioning,omitempty"`
	Error        string `json:"error,omitempty"`
}

const maxCrateStateAge = 20 * time.Minute

func buildDiagnosticsView() diagnosticsView {
	return diagnosticsView{
		Config:       loadConfigDiagnostics(),
		Verification: loadVerificationDiagnostics(),
		Ownership:    loadOwnershipDiagnostics(),
	}
}

func loadVerificationDiagnostics() verificationDiagnosticsView {
	view := verificationDiagnosticsView{
		Status:   "ready",
		Missing:  []string{},
		Warnings: []string{},
	}
	requiredFiles := []struct {
		path  string
		label string
	}{
		{platform.CratePath("state", "installed.json"), "installed marker"},
		{platform.CratePath("state", "platform-state.json"), "platform state"},
		{platform.CratePath("state", "readiness-report.json"), "readiness report"},
		{platform.CratePath("state", "storage-state.json"), "storage state"},
		{platform.CratePath("state", "actor-ownership-state.json"), "actor ownership state"},
	}
	for _, item := range requiredFiles {
		if _, err := os.Stat(item.path); err != nil {
			view.Missing = append(view.Missing, item.label)
		}
	}
	if _, err := os.Stat(platform.AgentSocket); err == nil {
		view.AgentSocket = true
	}
	if cfg, err := config.Load(); err == nil {
		for _, user := range cfg.Users.Users {
			if strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
				view.AdminPresent = true
				break
			}
		}
	}
	platformState := state.LoadPlatformState()
	view.PlatformState = strings.TrimSpace(platformState.GeneratedAt)
	if view.PlatformState == "" {
		view.Warnings = append(view.Warnings, "platform state not rendered yet")
	}
	if readiness, ok := loadReadinessStatusSummary(); ok {
		view.Readiness = readiness
		if readiness != "ready" {
			view.Warnings = append(view.Warnings, "readiness report is not ready")
		}
	} else {
		view.Warnings = append(view.Warnings, "readiness report unreadable")
	}
	if storage := state.LoadStorageState(); strings.TrimSpace(storage.GeneratedAt) != "" {
		view.StorageState = storage.GeneratedAt
	} else {
		view.Warnings = append(view.Warnings, "storage posture not rendered yet")
	}
	if ownership := state.LoadActorOwnershipState(); strings.TrimSpace(ownership.GeneratedAt) != "" {
		view.OwnershipState = ownership.GeneratedAt
	} else {
		view.Warnings = append(view.Warnings, "actor ownership state not rendered yet")
	}
	if !view.AgentSocket {
		view.Warnings = append(view.Warnings, "agent socket unavailable")
	}
	if !view.AdminPresent {
		view.Missing = append(view.Missing, "admin operator")
	}
	switch {
	case len(view.Missing) > 0:
		view.Status = "failed"
		view.Summary = "verification prerequisites missing"
	case len(view.Warnings) > 0:
		view.Status = "degraded"
		view.Summary = "verification surfaces present with warnings"
	default:
		view.Summary = "verification surfaces present"
	}
	return view
}

func loadOwnershipDiagnostics() ownershipDiagnosticsView {
	snapshot := state.LoadActorOwnershipState()
	view := ownershipDiagnosticsView{
		GeneratedAt: strings.TrimSpace(snapshot.GeneratedAt),
		Active:      snapshot.Active,
		Retired:     snapshot.Retired,
		Claims:      make([]ownershipDiagnosticClaimView, 0, len(snapshot.Claims)),
		Workloads:   []actorLifecycleDiagnosticView{},
	}
	claimsByCrate := map[string]state.ActorOwnershipStateItem{}
	for _, claim := range snapshot.Claims {
		claimsByCrate[strings.TrimSpace(claim.Crate)] = claim
		view.Claims = append(view.Claims, ownershipDiagnosticClaimView{
			Crate:     claim.Crate,
			Name:      claim.Name,
			Type:      claim.Type,
			ID:        claim.ID,
			User:      claim.User,
			Group:     claim.Group,
			Home:      claim.Home,
			Status:    claim.Status,
			UpdatedAt: claim.UpdatedAt,
			RetiredAt: claim.RetiredAt,
		})
	}
	if cfg, err := config.Load(); err == nil && cfg != nil {
		for _, svc := range cfg.Services.Services {
			if strings.TrimSpace(svc.Actor.Name) == "" && strings.TrimSpace(svc.Execution.Mode) == "" {
				continue
			}
			view.Managed++
			provisioning := state.LoadActorProvisioningState(svc.Name)
			item := actorLifecycleDiagnosticView{
				Crate:                 svc.Name,
				ActorName:             strings.TrimSpace(provisioning.Actor.Name),
				ActorType:             strings.TrimSpace(provisioning.Actor.Type),
				ActorID:               strings.TrimSpace(provisioning.Actor.ID),
				ActorUser:             strings.TrimSpace(provisioning.Actor.User),
				ActorGroup:            strings.TrimSpace(provisioning.Actor.Group),
				ActorHome:             strings.TrimSpace(provisioning.Actor.Home),
				Provisioning:          strings.TrimSpace(provisioning.Provisioning),
				ProvisioningError:     strings.TrimSpace(provisioning.Error),
				ProvisioningUpdatedAt: strings.TrimSpace(provisioning.GeneratedAt),
				LastSuccessAt:         strings.TrimSpace(provisioning.LastSuccessAt),
				LastFailureAt:         strings.TrimSpace(provisioning.LastFailureAt),
				RecentEvents:          make([]actorLifecycleEventDiagnosticView, 0, len(provisioning.Events)),
			}
			for _, event := range provisioning.Events {
				item.RecentEvents = append(item.RecentEvents, actorLifecycleEventDiagnosticView{
					At:           strings.TrimSpace(event.At),
					Provisioning: strings.TrimSpace(event.Provisioning),
					Error:        strings.TrimSpace(event.Error),
				})
			}
			if item.ActorName == "" {
				item.ActorName = strings.TrimSpace(svc.Actor.Name)
			}
			item.ProvisioningStatePath = platform.CratePath("services", svc.Name, "runtime", "actor-provisioning.json")
			if claim, ok := claimsByCrate[strings.TrimSpace(svc.Name)]; ok {
				item.OwnershipStatus = strings.TrimSpace(claim.Status)
				item.OwnershipUpdatedAt = strings.TrimSpace(claim.UpdatedAt)
				item.OwnershipRetiredAt = strings.TrimSpace(claim.RetiredAt)
				if item.ActorName == "" {
					item.ActorName = strings.TrimSpace(claim.Name)
				}
				if item.ActorType == "" {
					item.ActorType = strings.TrimSpace(claim.Type)
				}
				if item.ActorID == "" {
					item.ActorID = strings.TrimSpace(claim.ID)
				}
				if item.ActorUser == "" {
					item.ActorUser = strings.TrimSpace(claim.User)
				}
				if item.ActorGroup == "" {
					item.ActorGroup = strings.TrimSpace(claim.Group)
				}
				if item.ActorHome == "" {
					item.ActorHome = strings.TrimSpace(claim.Home)
				}
			}
			switch item.Provisioning {
			case "provisioned":
				view.Provisioned++
			case "blocked":
				view.Blocked++
			default:
				view.Pending++
			}
			view.Workloads = append(view.Workloads, item)
		}
	}
	return view
}

func loadReadinessStatusSummary() (string, bool) {
	data, err := os.ReadFile(platform.CratePath("state", "readiness-report.json"))
	if err != nil {
		return "", false
	}
	var report struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return "", false
	}
	status := strings.TrimSpace(report.Status)
	if status == "" {
		status = "unknown"
	}
	return status, true
}

func loadConfigDiagnostics() configDiagnosticsView {
	ledger, err := config.LoadConfigChangeLedger()
	if err != nil {
		return configDiagnosticsView{}
	}
	view := configDiagnosticsView{
		GeneratedAt: ledger.GeneratedAt,
		Files:       make([]configDiagnosticFileView, 0, len(ledger.Files)),
	}
	for _, record := range ledger.Files {
		view.Tracked++
		switch strings.TrimSpace(record.Monitoring) {
		case "unmonitored":
			view.Unmonitored++
		default:
			view.Monitored++
		}
		if strings.TrimSpace(record.LastWriter) == "external" {
			view.ExternalEdits++
		}
		view.Files = append(view.Files, configDiagnosticFileView{
			File:          record.File,
			Path:          record.Path,
			Exists:        record.Exists,
			Monitoring:    record.Monitoring,
			LastWriter:    record.LastWriter,
			LastSeenAt:    record.LastSeenAt,
			LastChangedAt: record.LastChangedAt,
		})
	}
	return view
}

func buildServiceView(cfg *config.Config, actual *state.ActualState, mods map[string]modules.Module) []serviceView {

	views := make([]serviceView, 0, len(cfg.Services.Services))
	for _, desired := range cfg.Services.Services {
		crate := loadCrateState(desired.Name)
		view := serviceView{
			Name:                       desired.Name,
			DisplayName:                desired.Name,
			Runtime:                    modules.ResolveRuntime(desired.Runtime, mods[desired.Name], false),
			ActorName:                  crate.ActorName,
			ActorType:                  crate.ActorType,
			ActorID:                    crate.ActorID,
			ActorUser:                  crate.ActorUser,
			ActorGroup:                 crate.ActorGroup,
			ActorHome:                  crate.ActorHome,
			ActorRuntimeDir:            crate.ActorRuntimeDir,
			ActorStateDir:              crate.ActorStateDir,
			ActorProvisioning:          crate.ActorProvisioning,
			ActorProvisioningError:     crate.ActorProvisioningError,
			ActorProvisioningUpdatedAt: crate.ActorProvisioningUpdatedAt,
			ActorProvisioningStatePath: crate.ActorProvisioningStatePath,
			ActorOwnershipStatus:       crate.ActorOwnershipStatus,
			ActorOwnershipUpdatedAt:    crate.ActorOwnershipUpdatedAt,
			ActorOwnershipRetiredAt:    crate.ActorOwnershipRetiredAt,
			DeploySource:               crate.DeploySource,
			UploadPath:                 crate.UploadPath,
			WorkingDir:                 crate.WorkingDir,
			Entry:                      crate.Entry,
			InstallCommand:             crate.InstallCommand,
			EnvironmentFile:            crate.EnvironmentFile,
			ExecutionMode:              crate.ExecutionMode,
			ExecutionAdapter:           crate.ExecutionAdapter,
			ExecutionStatus:            crate.ExecutionStatus,
			PrimaryUnit:                crate.PrimaryUnit,
			CompanionUnit:              crate.CompanionUnit,
			PrimaryUnitPath:            crate.PrimaryUnitPath,
			CompanionUnitPath:          crate.CompanionUnitPath,
			ExecutionStatePath:         crate.ExecutionStatePath,
			StartCommand:               crate.StartCommand,
			Schedule:                   crate.Schedule,
			Timeout:                    crate.Timeout,
			StopTimeout:                crate.StopTimeout,
			OnTimeout:                  crate.OnTimeout,
			KillSignal:                 crate.KillSignal,
			ConcurrencyPolicy:          crate.ConcurrencyPolicy,
			ExecutionSummary:           crate.ExecutionSummary,
			Stateful:                   crate.Stateful,
			DataPath:                   crate.DataPath,
			NativeDataPath:             crate.NativeDataPath,
			StorageSummary:             crate.StorageSummary,
			Desired:                    desired.Enabled,
			Autostart:                  desired.Autostart,
			Ready:                      crate.Ready,
			Status:                     crate.Status,
			Health:                     crate.Health,
			PackagesInstalled:          crate.PackagesInstalled,
			MissingPackages:            append([]string(nil), crate.MissingPackages...),
			Summary:                    crate.Summary,
			LastError:                  crate.LastError,
			LastAction:                 crate.LastAction,
			LastActionAt:               crate.LastActionAt,
			SuggestedRepair:            crate.SuggestedRepair,
		}
		if lastGood, ok := loadLastGoodCrateState(desired.Name); ok {
			view.LastGoodStatus = lastGood.Crate.Status
			view.LastGoodHealth = lastGood.Crate.Health
			view.LastGoodAt = lastGood.GeneratedAt
			view.LastGoodSummary = lastGood.Crate.Summary
		}
		if mod, ok := mods[desired.Name]; ok {
			view.Module = true
			view.DisplayName = mod.DisplayName()
			view.Category = mod.Metadata.Category
			view.Description = mod.Metadata.Description
			view.InstallMode = modules.ResolveInstallMode(mod, true)
			view.Packages = append([]string(nil), mod.Spec.Packages...)
			view.Runtime = modules.ResolveRuntime(desired.Runtime, mod, true)
		}
		if len(crate.Units) > 0 {
			view.Active = anyActive(crate.Units)
			view.Enabled = allEnabled(crate.Units)
			for _, unit := range crate.Units {
				view.Units = append(view.Units, unitView{
					Name:    unit.Name,
					Active:  unit.Active,
					Enabled: unit.Enabled,
					Status:  unit.Status,
					Health:  unit.Health,
				})
			}
		}
		if view.DisplayName == "" {
			view.DisplayName = desired.Name
		}
		if view.Status == "" {
			view.Status = "unknown"
		}
		if view.Health == "" {
			view.Health = "unknown"
		}
		views = append(views, view)
	}
	return views
}

func anyActive(units []state.ServiceState) bool {
	for _, unit := range units {
		if unit.Active {
			return true
		}
	}
	return false
}

func allEnabled(units []state.ServiceState) bool {
	if len(units) == 0 {
		return false
	}
	for _, unit := range units {
		if !unit.Enabled {
			return false
		}
	}
	return true
}

func loadCrateState(name string) state.CrateState {
	path := platform.CratePath("services", name, "crate-state.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return state.CrateState{Name: name, DisplayName: name, Status: "unknown", Health: "unknown"}
	}
	var stored state.StoredCrateState
	if err := json.Unmarshal(b, &stored); err != nil {
		return state.CrateState{Name: name, DisplayName: name, Status: "unknown", Health: "unknown"}
	}
	applyStoredCrateStateFreshness(&stored, time.Now().UTC())
	if stored.Crate.Name == "" {
		stored.Crate.Name = name
	}
	if stored.Crate.DisplayName == "" {
		stored.Crate.DisplayName = name
	}
	if stored.Crate.Status == "" {
		stored.Crate.Status = "unknown"
	}
	if stored.Crate.Health == "" {
		stored.Crate.Health = "unknown"
	}
	return stored.Crate
}

func loadLastGoodCrateState(name string) (state.StoredCrateState, bool) {
	path := platform.CratePath("services", name, "runtime", "last-good", "crate-state.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return state.StoredCrateState{}, false
	}
	var stored state.StoredCrateState
	if err := json.Unmarshal(b, &stored); err != nil {
		return state.StoredCrateState{}, false
	}
	if stored.Crate.Name == "" {
		stored.Crate.Name = name
	}
	if stored.Crate.DisplayName == "" {
		stored.Crate.DisplayName = name
	}
	return stored, true
}

func applyStoredCrateStateFreshness(stored *state.StoredCrateState, now time.Time) {
	generatedAtRaw := strings.TrimSpace(stored.GeneratedAt)
	if generatedAtRaw == "" {
		markStoredCrateStateStale(stored, "crate state missing generated_at")
		return
	}
	generatedAt, err := time.Parse(time.RFC3339, generatedAtRaw)
	if err != nil {
		markStoredCrateStateStale(stored, "crate state has invalid generated_at")
		return
	}
	age := now.Sub(generatedAt)
	if age > maxCrateStateAge {
		markStoredCrateStateStale(stored, fmt.Sprintf("crate state stale: last agent render %s ago", age.Round(time.Second)))
	}
}

func markStoredCrateStateStale(stored *state.StoredCrateState, reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "crate state stale"
	}
	stored.Crate.Status = "failed"
	stored.Crate.Health = "degraded"
	stored.Crate.Ready = false
	stored.Crate.LastError = reason
	if strings.TrimSpace(stored.Crate.Summary) == "" || strings.TrimSpace(stored.Crate.Summary) == "rendered desired state successfully" {
		stored.Crate.Summary = reason
	}
}

type serviceAction string

const (
	serviceActionEnableOnly serviceAction = "enable-only"
	serviceActionDisable    serviceAction = "disable"
	serviceActionStart      serviceAction = "start"
	serviceActionStop       serviceAction = "stop"
)

func applyServiceAction(name string, action serviceAction, mods map[string]modules.Module) {
	targets := []string{name}
	if mod, ok := mods[name]; ok {
		if units := modules.ResolveUnits(name, mod, true); len(units) > 0 {
			targets = units
		}
	}
	for _, target := range targets {
		switch action {
		case serviceActionEnableOnly:
			systemctlNoError("enable", target)
		case serviceActionDisable:
			systemctlNoError("stop", target)
			systemctlNoError("disable", target)
		case serviceActionStart:
			systemctlNoError("enable", target)
			systemctlNoError("start", target)
		case serviceActionStop:
			systemctlNoError("stop", target)
		}
	}
}

func shouldAutostartOnEnable(name string, mods map[string]modules.Module) bool {
	if mod, ok := mods[name]; ok {
		return mod.InstallMode() != "staged"
	}
	return true
}
