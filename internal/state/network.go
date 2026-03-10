package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

func reconcileNetwork(cfg config.NetworkConfig) ([]Action, PlatformAdapterState) {
	var actions []Action
	var issues []string
	index := make([]map[string]interface{}, 0, len(cfg.Profiles))
	renderedPaths := make([]string, 0, len(cfg.Profiles))
	validationPath := platform.CratePath("state", "rendered", "network.check.json")
	adapter := platformAdapterState("network", "Network", len(cfg.Profiles) > 0 || strings.TrimSpace(cfg.Manager) != "")
	adapter.RenderedPaths = append(adapter.RenderedPaths, validationPath)
	validationIssues := validateNetworkConfig(cfg)
	if len(validationIssues) > 0 {
		adapter.Validation = "failed"
		adapter.ValidationErr = strings.Join(validationIssues, "; ")
		adapter.Apply = "blocked"
		issues = append(issues, adapter.ValidationErr)
	}
	for _, profile := range cfg.Profiles {
		filename := sanitizeName(profile.Name) + ".nmconnection"
		content := renderNetworkProfile(cfg, profile)
		renderedPath := platform.CratePath("state", "rendered", "network", filename)
		if len(validationIssues) == 0 {
			if action, err := writeManagedArtifact(
				filepath.ToSlash(filepath.Join("network", filename)),
				renderedPath,
				content,
				"network",
				fmt.Sprintf("rendered network profile %s", profile.Name),
			); err != nil {
				issues = append(issues, err.Error())
			} else if action != nil {
				actions = append(actions, *action)
			}
		}
		renderedPaths = append(renderedPaths, renderedPath)
		adapter.RenderedPaths = append(adapter.RenderedPaths, renderedPath)
		adapter.NativeTargets = append(adapter.NativeTargets, filepath.Join("/etc/NetworkManager/system-connections", filepath.Base(renderedPath)))
		index = append(index, map[string]interface{}{
			"name":        profile.Name,
			"type":        profile.Type,
			"method":      profile.Method,
			"metric":      profile.Metric,
			"autoconnect": profile.Autoconnect,
			"has_mac":     strings.TrimSpace(profile.MAC) != "",
			"has_ssid":    strings.TrimSpace(profile.SSID) != "",
			"has_static":  strings.EqualFold(strings.TrimSpace(profile.Method), "manual"),
		})
	}
	if adapter.Enabled && len(validationIssues) == 0 {
		adapter.Validation = "ok"
	}

	summary := map[string]interface{}{
		"manager":          cfg.Manager,
		"self_heal":        cfg.SelfHeal,
		"dns_fallback":     cfg.DNS.Fallback,
		"profiles":         index,
		"validation":       adapter.Validation,
		"validation_error": adapter.ValidationErr,
		"apply":            adapter.Apply,
		"apply_error":      adapter.ApplyErr,
	}
	if data, err := json.MarshalIndent(summary, "", "  "); err == nil {
		if action, err := writeManagedArtifact(
			"network/state.json",
			platform.CratePath("state", "rendered", "network.json"),
			string(data)+"\n",
			"network",
			"rendered network state",
		); err != nil {
			issues = append(issues, err.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	}
	adapter.RenderedPaths = append(adapter.RenderedPaths, platform.CratePath("state", "rendered", "network.json"))
	if validationData, err := json.MarshalIndent(map[string]interface{}{
		"manager":          cfg.Manager,
		"profile_count":    len(cfg.Profiles),
		"validation":       adapter.Validation,
		"validation_error": adapter.ValidationErr,
	}, "", "  "); err == nil {
		if action, err := writeManagedArtifact(
			"network/validation.json",
			validationPath,
			string(validationData)+"\n",
			"network",
			"rendered network validation state",
		); err != nil {
			issues = append(issues, err.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	}
	adapter.Summary = fmt.Sprintf("managed %d network profiles via %s", len(cfg.Profiles), strings.TrimSpace(cfg.Manager))
	if len(validationIssues) == 0 {
		applyActions, applyIssues, applyStatus, applyErr := applyNetwork(cfg, renderedPaths)
		actions = append(actions, applyActions...)
		issues = append(issues, applyIssues...)
		adapter.Apply = applyStatus
		adapter.ApplyErr = applyErr
	}
	if !adapter.Enabled {
		adapter.Validation = "disabled"
		adapter.ValidationErr = ""
		if strings.TrimSpace(adapter.Apply) == "" || adapter.Apply == "pending" {
			adapter.Apply = "disabled"
		}
	}

	return actions, finalizePlatformAdapterState(adapter, issues)
}

func applyNetwork(cfg config.NetworkConfig, renderedPaths []string) ([]Action, []string, string, string) {
	if runtime.GOOS != "linux" || !strings.EqualFold(strings.TrimSpace(cfg.Manager), "networkmanager") {
		return nil, nil, "skipped", ""
	}
	var actions []Action
	var issues []string
	targetDir := "/etc/NetworkManager/system-connections"
	desired := make(map[string]bool, len(renderedPaths))
	for _, renderedPath := range renderedPaths {
		targetPath := filepath.Join(targetDir, filepath.Base(renderedPath))
		desired[filepath.Base(renderedPath)] = true
		if action, err := syncManagedArtifact(
			filepath.ToSlash(filepath.Join("native/network", filepath.Base(renderedPath))),
			renderedPath,
			targetPath,
			"network",
			fmt.Sprintf("installed native network profile %s", filepath.Base(renderedPath)),
		); err != nil {
			issues = append(issues, err.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	}
	if cfg.SelfHeal {
		cleanupActions, cleanupIssues := cleanupStaleNetworkProfiles(targetDir, desired)
		actions = append(actions, cleanupActions...)
		issues = append(issues, cleanupIssues...)
	}
	if len(actions) > 0 {
		if exec.Command("nmcli", "connection", "reload").Run() == nil {
			actions = append(actions, Action{
				Description: "reloaded NetworkManager connection profiles",
				Component:   "network",
				Target:      "NetworkManager",
			})
		} else {
			log.Printf("warning: nmcli connection reload failed")
			issues = append(issues, "NetworkManager connection reload failed")
		}
	}
	if len(issues) > 0 {
		return actions, issues, "failed", issues[len(issues)-1]
	}
	if len(actions) == 0 {
		return actions, issues, "unchanged", ""
	}
	return actions, issues, "applied", ""
}

func renderNetworkProfile(cfg config.NetworkConfig, profile config.NetworkProfile) string {
	values := map[string]interface{}{
		"connection": map[string]interface{}{
			"id":             profile.Name,
			"type":           profile.Type,
			"autoconnect":    boolToInt(profile.Autoconnect),
			"interface-name": "",
		},
		"ipv4": map[string]interface{}{
			"method":       profile.Method,
			"route-metric": profile.Metric,
		},
		"ipv6": map[string]interface{}{
			"method": "auto",
		},
	}
	if strings.TrimSpace(profile.MAC) != "" {
		values["match"] = map[string]interface{}{"mac-address": profile.MAC}
	}
	if strings.EqualFold(profile.Type, "wifi") {
		wifi := map[string]interface{}{}
		if strings.TrimSpace(profile.SSID) != "" {
			wifi["ssid"] = profile.SSID
		}
		values["wifi"] = wifi
		if strings.TrimSpace(profile.Password) != "" {
			values["wifi-security"] = map[string]interface{}{
				"key-mgmt": "wpa-psk",
				"psk":      profile.Password,
			}
		}
	}
	if strings.EqualFold(profile.Method, "manual") {
		ipv4 := values["ipv4"].(map[string]interface{})
		if strings.TrimSpace(profile.Static.Address) != "" {
			ipv4["address1"] = profile.Static.Address
		}
		if strings.TrimSpace(profile.Static.Gateway) != "" {
			ipv4["gateway"] = profile.Static.Gateway
		}
		if len(profile.Static.DNS) > 0 {
			ipv4["dns"] = strings.Join(profile.Static.DNS, ";") + ";"
		}
	} else if len(cfg.DNS.Fallback) > 0 {
		values["ipv4"].(map[string]interface{})["dns"] = strings.Join(cfg.DNS.Fallback, ";") + ";"
	}

	var b strings.Builder
	b.WriteString("# Generated by CrateOS. Do not edit manually.\n")
	sections := make([]string, 0, len(values))
	for section := range values {
		sections = append(sections, section)
	}
	sort.Strings(sections)
	for _, section := range sections {
		b.WriteString("[" + section + "]\n")
		if m, ok := values[section].(map[string]interface{}); ok {
			keys := make([]string, 0, len(m))
			for k := range m {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, key := range keys {
				val := fmt.Sprintf("%v", m[key])
				if strings.TrimSpace(val) == "" {
					continue
				}
				b.WriteString(fmt.Sprintf("%s=%s\n", key, val))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}
