package tui

import (
	"encoding/json"

	"github.com/crateos/crateos/internal/api"
	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/sysinfo"
)

// fetchStatusViaAPI tries the local agent socket; returns nil on failure.
func fetchStatusViaAPI(user string) (*sysinfo.Info, []ServiceInfo, PlatformInfo, DiagnosticsInfo, string) {
	c := api.NewClient(user)
	raw, err := c.Status()
	if err != nil {
		return nil, nil, PlatformInfo{}, DiagnosticsInfo{}, user
	}
	var info sysinfo.Info
	if b, err := json.Marshal(raw["sysinfo"]); err == nil {
		_ = json.Unmarshal(b, &info)
	}
	var platformInfo PlatformInfo
	if b, err := json.Marshal(raw["platform"]); err == nil {
		_ = json.Unmarshal(b, &platformInfo)
	}
	var diagnostics DiagnosticsInfo
	if b, err := json.Marshal(raw["diagnostics"]); err == nil {
		_ = json.Unmarshal(b, &diagnostics)
	}
	if actorRaw, err := c.ActorDiagnostics(); err == nil {
		if ownership, ok := actorRaw["ownership"]; ok {
			if b, err := json.Marshal(ownership); err == nil {
				_ = json.Unmarshal(b, &diagnostics.Ownership)
			}
		}
	}
	svcResp, err := c.Services()
	if err != nil {
		return &info, nil, platformInfo, diagnostics, user
	}
	var svcs []ServiceInfo
	if services, ok := svcResp["services"]; ok {
		if b, err := json.Marshal(services); err == nil {
			_ = json.Unmarshal(b, &svcs)
		}
	}
	return &info, svcs, platformInfo, diagnostics, user
}

func fetchUsersViaAPI(user string) []userRow {
	c := api.NewClient(user)
	raw, err := c.Users()
	if err != nil {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var cfg struct {
		Users []struct {
			Name  string   `json:"name" yaml:"name"`
			Role  string   `json:"role" yaml:"role"`
			Perms []string `json:"permissions" yaml:"permissions"`
		} `json:"users"`
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil
	}
	var rows []userRow
	for _, u := range cfg.Users {
		rows = append(rows, userRow{Name: u.Name, Role: u.Role, Perms: u.Perms})
	}
	return rows
}

func fetchUsersFromConfig() []userRow {
	cfg, err := config.Load()
	if err != nil {
		return nil
	}
	rows := make([]userRow, 0, len(cfg.Users.Users))
	for _, u := range cfg.Users.Users {
		rows = append(rows, userRow{Name: u.Name, Role: u.Role, Perms: u.Permissions})
	}
	return rows
}

func defaultUser() string {
	if rows := fetchUsersViaAPI(""); rows != nil && len(rows) > 0 {
		return rows[0].Name
	}
	return "crate"
}

func selectInitialUser() string {
	if rows := fetchUsersViaAPI(""); rows != nil && len(rows) > 0 {
		return rows[0].Name
	}
	if rows := fetchUsersFromConfig(); len(rows) > 0 {
		return rows[0].Name
	}
	return "crate"
}
