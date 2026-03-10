package state

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/sysinfo"
	"github.com/crateos/crateos/internal/users"
	"github.com/crateos/crateos/internal/virtualization"
)

func reconcilePlatform(cfg *config.Config) []Action {
	var actions []Action
	accessActions, accessState := reconcileAccess(cfg.CrateOS)
	userActions, userState := reconcileUsers(cfg)
	virtDesktopState := virtualization.ReconcileVirtualDesktop(cfg)
	reverseProxyActions, reverseProxyState := reconcileReverseProxy(cfg.ReverseProxy)
	firewallActions, firewallState := reconcileFirewall(cfg.Firewall)
	networkActions, networkState := reconcileNetwork(cfg.Network)
	storageActions, storageState := reconcileStorage()
	actions = append(actions, accessActions...)
	actions = append(actions, userActions...)
	actions = append(actions, reverseProxyActions...)
	actions = append(actions, firewallActions...)
	actions = append(actions, networkActions...)
	actions = append(actions, storageActions...)
	// Virtual desktop state is informational, no direct actions
	if len(virtDesktopState.Issues) > 0 {
		for _, issue := range virtDesktopState.Issues {
			actions = append(actions, Action{
				Description: fmt.Sprintf("virtual desktop: %s", issue),
				Component:   "virtualization",
				Target:      "sessions",
			})
		}
	}
	writePlatformState(PlatformState{
		Adapters: []PlatformAdapterState{accessState, userState, reverseProxyState, firewallState, networkState, storageState},
	})
	return actions
}

func reconcileAccess(cfg config.CrateOSConfig) ([]Action, PlatformAdapterState) {
	var actions []Action
	var issues []string
	adapter := platformAdapterState("access", "Access", true)
	renderedPath := platform.CratePath("state", "rendered", "access.json")
	adapter.RenderedPaths = append(adapter.RenderedPaths, renderedPath)

	validationIssues, summary := validateAccessConfig(cfg)
	if len(validationIssues) > 0 {
		adapter.Validation = "failed"
		adapter.ValidationErr = strings.Join(validationIssues, "; ")
		adapter.Apply = "blocked"
		issues = append(issues, adapter.ValidationErr)
	} else {
		adapter.Validation = "ok"
		adapter.Apply = "ok"
	}

	localGUIProvider := strings.TrimSpace(cfg.Access.LocalGUI.Provider)
	virtualDesktopProvider := strings.TrimSpace(cfg.Access.VirtualDesktop.Provider)
	if runtime.GOOS == "linux" {
		switch {
		case cfg.Access.LocalGUI.Enabled && strings.EqualFold(localGUIProvider, "lightdm") && !packageInstalled("lightdm"):
			issues = append(issues, "local gui enabled but lightdm package is not installed")
		case cfg.Access.VirtualDesktop.Enabled && virtualDesktopProvider != "" && !isSupportedVirtualDesktopProvider(virtualDesktopProvider):
			issues = append(issues, "virtual desktop provider is not yet supported by CrateOS session adapters")
		}
	}

	stateSummary := map[string]interface{}{
		"ssh": map[string]interface{}{
			"enabled": cfg.Access.SSH.Enabled,
			"landing": normalizeLanding(cfg.Access.SSH.Landing),
		},
		"local_gui": map[string]interface{}{
			"enabled":       cfg.Access.LocalGUI.Enabled,
			"provider":      normalizeLocalGUIProvider(cfg.Access.LocalGUI.Provider),
			"landing":       normalizeLanding(cfg.Access.LocalGUI.Landing),
			"default_shell": normalizeDefaultShell(cfg.Access.LocalGUI.DefaultShell),
		},
		"virtual_desktop": map[string]interface{}{
			"enabled":  cfg.Access.VirtualDesktop.Enabled,
			"provider": normalizeVirtualDesktopProvider(cfg.Access.VirtualDesktop.Provider),
			"landing":  normalizeLanding(cfg.Access.VirtualDesktop.Landing),
		},
		"break_glass": map[string]interface{}{
			"enabled":            cfg.Access.BreakGlass.Enabled,
			"require_permission": normalizeBreakGlassPermission(cfg.Access.BreakGlass.RequirePerm),
			"allowed_surfaces":   normalizeAllowedSurfaces(cfg.Access.BreakGlass.AllowedSurfaces),
		},
		"summary": summary,
		"validation": map[string]interface{}{
			"status": adapter.Validation,
			"error":  adapter.ValidationErr,
		},
	}
	if data, err := json.MarshalIndent(stateSummary, "", "  "); err == nil {
		if action, err := writeManagedArtifact(
			"access/state.json",
			renderedPath,
			string(data)+"\n",
			"access",
			"rendered access and session state",
		); err != nil {
			issues = append(issues, err.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	}

	adapter.Summary = summary
	return actions, finalizePlatformAdapterState(adapter, issues)
}

func validateAccessConfig(cfg config.CrateOSConfig) ([]string, string) {
	issues := []string{}
	enabledSurfaces := 0
	if cfg.Access.SSH.Enabled {
		enabledSurfaces++
	}
	if cfg.Access.LocalGUI.Enabled {
		enabledSurfaces++
	}
	if cfg.Access.VirtualDesktop.Enabled {
		enabledSurfaces++
	}
	if enabledSurfaces == 0 {
		issues = append(issues, "at least one controlled entry surface must remain enabled")
	}

	sshLanding := normalizeLanding(cfg.Access.SSH.Landing)
	if cfg.Access.SSH.Enabled && sshLanding != "console" {
		issues = append(issues, "ssh landing must remain console")
	}

	localLanding := normalizeLanding(cfg.Access.LocalGUI.Landing)
	if cfg.Access.LocalGUI.Enabled {
		if normalizeLocalGUIProvider(cfg.Access.LocalGUI.Provider) != "lightdm" {
			issues = append(issues, "local_gui provider must be lightdm when enabled")
		}
		if localLanding == "shell" || localLanding == "desktop" {
			issues = append(issues, "local_gui landing must stay inside CrateOS-owned surfaces")
		}
		if normalizeDefaultShell(cfg.Access.LocalGUI.DefaultShell) != "crateos-session" {
			issues = append(issues, "local_gui default_shell must be crateos-session")
		}
	}

	virtualLanding := normalizeLanding(cfg.Access.VirtualDesktop.Landing)
	if cfg.Access.VirtualDesktop.Enabled && (virtualLanding == "shell" || virtualLanding == "desktop") {
		issues = append(issues, "virtual_desktop landing must stay inside CrateOS-owned surfaces")
	}

	breakGlassPerm := normalizeBreakGlassPermission(cfg.Access.BreakGlass.RequirePerm)
	if cfg.Access.BreakGlass.Enabled && breakGlassPerm == "" {
		issues = append(issues, "break_glass enabled requires require_permission")
	}

	allowedSurfaces := normalizeAllowedSurfaces(cfg.Access.BreakGlass.AllowedSurfaces)
	for _, surface := range allowedSurfaces {
		switch surface {
		case "ssh", "local_gui", "virtual_desktop":
		default:
			issues = append(issues, fmt.Sprintf("break_glass allowed surface %q is unsupported", surface))
		}
	}

	summaryParts := []string{}
	if cfg.Access.SSH.Enabled {
		summaryParts = append(summaryParts, "ssh→"+sshLanding)
	}
	if cfg.Access.LocalGUI.Enabled {
		summaryParts = append(summaryParts, "local_gui→"+localLanding)
	}
	if cfg.Access.VirtualDesktop.Enabled {
		summaryParts = append(summaryParts, "virtual_desktop→"+virtualLanding)
	}
	summary := "controlled entry surfaces: " + strings.Join(summaryParts, ", ")
	if cfg.Access.BreakGlass.Enabled {
		summary += "; break-glass gated by " + breakGlassPerm
	} else {
		summary += "; break-glass disabled"
	}
	return issues, summary
}

func normalizeLanding(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "console":
		return "console"
	case "panel":
		return "panel"
	case "workspace":
		return "workspace"
	case "recovery":
		return "recovery"
	case "shell":
		return "shell"
	case "desktop":
		return "desktop"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeLocalGUIProvider(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "lightdm":
		return "lightdm"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeVirtualDesktopProvider(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeDefaultShell(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "crateos-session":
		return "crateos-session"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeBreakGlassPermission(value string) string {
	return strings.TrimSpace(value)
}

func normalizeAllowedSurfaces(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func isSupportedVirtualDesktopProvider(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "none":
		return true
	default:
		return false
	}
}

func reconcileStorage() ([]Action, PlatformAdapterState) {
	adapter := platformAdapterState("storage", "Storage", true)
	adapter.RenderedPaths = append(adapter.RenderedPaths, platform.CratePath("state", "storage-state.json"))
	devices := sysinfo.StorageDevices()
	storageState := normalizeStorageDevices(devices)
	writeStorageState(storageState)
	adapter.Validation = "ok"
	adapter.Apply = "ok"
	switch {
	case runtime.GOOS != "linux":
		adapter.Status = "unknown"
		adapter.Health = "unknown"
		adapter.Summary = "storage posture only probes linux hosts"
	case len(storageState.Devices) == 0:
		adapter.Status = "failed"
		adapter.Health = "degraded"
		adapter.Validation = "failed"
		adapter.Apply = "skipped"
		adapter.LastError = "no storage devices detected from lsblk"
		adapter.ValidationErr = adapter.LastError
		adapter.Summary = adapter.LastError
	case len(storageState.SafeTargets) == 0:
		adapter.Status = "ready"
		adapter.Health = "degraded"
		adapter.Summary = fmt.Sprintf("detected %d storage devices; no dedicated safe data target mounted yet", len(storageState.Devices))
		adapter.LastError = "stateful crates still share the system disk unless a target is mounted under /srv, /mnt, or /media"
	default:
		adapter.Status = "ready"
		adapter.Health = "ok"
		adapter.Summary = fmt.Sprintf("detected %d storage devices; %d safe data target(s) available", len(storageState.Devices), len(storageState.SafeTargets))
	}
	return nil, adapter
}

func validateNetworkConfig(cfg config.NetworkConfig) []string {
	var issues []string
	manager := strings.TrimSpace(strings.ToLower(cfg.Manager))
	if manager != "" && manager != "networkmanager" {
		issues = append(issues, "network manager must be networkmanager")
	}
	seen := make(map[string]string, len(cfg.Profiles))
	for i, profile := range cfg.Profiles {
		label := networkProfileLabel(profile, i)
		profileType := strings.TrimSpace(strings.ToLower(profile.Type))
		if profileType != "ethernet" && profileType != "wifi" {
			issues = append(issues, fmt.Sprintf("%s type must be ethernet or wifi", label))
		}
		method := strings.TrimSpace(strings.ToLower(profile.Method))
		if method == "" {
			method = "auto"
		}
		if method != "auto" && method != "manual" {
			issues = append(issues, fmt.Sprintf("%s method must be auto or manual", label))
		}
		if strings.TrimSpace(profile.Name) == "" {
			issues = append(issues, fmt.Sprintf("%s requires a name", label))
		}
		if strings.TrimSpace(profile.MAC) != "" && !isValidMAC(profile.MAC) {
			issues = append(issues, fmt.Sprintf("%s has invalid mac", label))
		}
		if profileType == "wifi" {
			if strings.TrimSpace(profile.SSID) == "" {
				issues = append(issues, fmt.Sprintf("%s wifi profile requires ssid", label))
			}
			if strings.TrimSpace(profile.Password) == "" {
				issues = append(issues, fmt.Sprintf("%s wifi profile requires password", label))
			}
		}
		if method == "manual" {
			if strings.TrimSpace(profile.Static.Address) == "" {
				issues = append(issues, fmt.Sprintf("%s manual profile requires static.address", label))
			} else if _, _, err := net.ParseCIDR(strings.TrimSpace(profile.Static.Address)); err != nil {
				issues = append(issues, fmt.Sprintf("%s static.address must be CIDR", label))
			}
			if strings.TrimSpace(profile.Static.Gateway) == "" {
				issues = append(issues, fmt.Sprintf("%s manual profile requires static.gateway", label))
			} else if ip := net.ParseIP(strings.TrimSpace(profile.Static.Gateway)); ip == nil {
				issues = append(issues, fmt.Sprintf("%s static.gateway must be an IP", label))
			}
			for _, dns := range profile.Static.DNS {
				if strings.TrimSpace(dns) == "" {
					continue
				}
				if ip := net.ParseIP(strings.TrimSpace(dns)); ip == nil {
					issues = append(issues, fmt.Sprintf("%s static dns must be IPs", label))
					break
				}
			}
		}
		key := strings.ToLower(strings.TrimSpace(profile.Name))
		if previous, exists := seen[key]; exists {
			issues = append(issues, fmt.Sprintf("%s duplicates %s", label, previous))
		} else {
			seen[key] = label
		}
	}
	for _, dns := range cfg.DNS.Fallback {
		if strings.TrimSpace(dns) == "" {
			continue
		}
		if ip := net.ParseIP(strings.TrimSpace(dns)); ip == nil {
			issues = append(issues, "network dns fallback must contain IPs")
			break
		}
	}
	return issues
}

func networkProfileLabel(profile config.NetworkProfile, index int) string {
	if strings.TrimSpace(profile.Name) != "" {
		return fmt.Sprintf("network profile %q", profile.Name)
	}
	return fmt.Sprintf("network profile #%d", index+1)
}

func isValidMAC(value string) bool {
	_, err := net.ParseMAC(strings.TrimSpace(value))
	return err == nil
}

func cleanupStaleNetworkProfiles(targetDir string, desired map[string]bool) ([]Action, []string) {
	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, []string{fmt.Sprintf("read network profile dir %s: %v", targetDir, err)}
	}
	var actions []Action
	var issues []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if desired[name] {
			continue
		}
		targetPath := filepath.Join(targetDir, name)
		data, err := os.ReadFile(targetPath)
		if err != nil {
			issues = append(issues, fmt.Sprintf("read network profile %s: %v", targetPath, err))
			continue
		}
		if !isCrateManagedArtifact(data) {
			continue
		}
		if action, err := removeManagedArtifact(filepath.ToSlash(filepath.Join("native/network", name)), targetPath, "network", fmt.Sprintf("removed stale managed network profile %s", name)); err != nil {
			issues = append(issues, err.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	}
	return actions, issues
}

func isCrateManagedArtifact(data []byte) bool {
	return strings.HasPrefix(string(data), "# Generated by CrateOS. Do not edit manually.")
}

func reconcileReverseProxy(cfg config.ReverseProxyConfig) ([]Action, PlatformAdapterState) {
	var actions []Action
	var issues []string
	renderedPath := platform.CratePath("services", "nginx", "config", "crateos-generated.conf")
	validationPath := platform.CratePath("state", "rendered", "reverse-proxy.nginx-check.conf")
	adapter := platformAdapterState("reverse-proxy", "Reverse Proxy", cfg.Enabled)
	adapter.RenderedPaths = append(adapter.RenderedPaths, renderedPath, platform.CratePath("state", "rendered", "reverse-proxy.json"))
	adapter.NativeTargets = append(adapter.NativeTargets, "/etc/nginx/conf.d/crateos-generated.conf")
	if cfg.ValidateBeforeApply {
		adapter.RenderedPaths = append(adapter.RenderedPaths, validationPath)
	}

	validationIssues := validateReverseProxyConfig(cfg)
	if len(validationIssues) > 0 {
		adapter.Validation = "failed"
		adapter.ValidationErr = strings.Join(validationIssues, "; ")
		adapter.Apply = "blocked"
		issues = append(issues, adapter.ValidationErr)
	}

	nginxConf := renderReverseProxyNginx(cfg)
	if len(validationIssues) == 0 {
		if action, err := writeManagedArtifact(
			"reverse-proxy/nginx.conf",
			renderedPath,
			nginxConf,
			"proxy",
			"rendered reverse proxy config",
		); err != nil {
			issues = append(issues, err.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	}

	if len(cfg.Mappings) == 0 {
		adapter.Summary = "no reverse proxy mappings defined"
	} else {
		httpsMappings := countReverseProxyHTTPSMappings(cfg)
		if httpsMappings > 0 {
			adapter.Summary = fmt.Sprintf("managed %d reverse proxy mappings (%d with https)", len(cfg.Mappings), httpsMappings)
		} else {
			adapter.Summary = fmt.Sprintf("managed %d reverse proxy mappings", len(cfg.Mappings))
		}
	}
	if len(validationIssues) == 0 {
		if validationIssue := validateRenderedReverseProxy(cfg, renderedPath, validationPath); validationIssue != "" {
			adapter.Validation = "failed"
			adapter.ValidationErr = validationIssue
			adapter.Apply = "blocked"
			issues = append(issues, validationIssue)
		} else {
			if adapter.Enabled {
				if cfg.ValidateBeforeApply {
					adapter.Validation = "ok"
				} else {
					adapter.Validation = "skipped"
				}
			}
			applyActions, applyIssues, applyStatus, applyErr := applyReverseProxy(cfg, renderedPath)
			actions = append(actions, applyActions...)
			issues = append(issues, applyIssues...)
			adapter.Apply = applyStatus
			adapter.ApplyErr = applyErr
		}
	}
	if !cfg.Enabled {
		adapter.Validation = "disabled"
		adapter.ValidationErr = ""
		adapter.ApplyErr = ""
	}
	summary := map[string]interface{}{
		"enabled":               cfg.Enabled,
		"validate_before_apply": cfg.ValidateBeforeApply,
		"listen_http":           cfg.Defaults.ListenHTTP,
		"listen_https":          cfg.Defaults.ListenHTTPS,
		"ssl_default":           cfg.Defaults.SSL,
		"mapping_count":         len(cfg.Mappings),
		"mappings":              cfg.Mappings,
		"validation":            adapter.Validation,
		"validation_error":      adapter.ValidationErr,
		"apply":                 adapter.Apply,
		"apply_error":           adapter.ApplyErr,
	}
	if data, err := json.MarshalIndent(summary, "", "  "); err == nil {
		if action, err := writeManagedArtifact(
			"reverse-proxy/state.json",
			platform.CratePath("state", "rendered", "reverse-proxy.json"),
			string(data)+"\n",
			"proxy",
			"rendered reverse proxy state",
		); err != nil {
			issues = append(issues, err.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	}

	return actions, finalizePlatformAdapterState(adapter, issues)
}

func reconcileFirewall(cfg config.FirewallConfig) ([]Action, PlatformAdapterState) {
	var actions []Action
	var issues []string
	renderedPath := platform.CratePath("state", "rendered", "firewall.nft")
	validationPath := platform.CratePath("state", "rendered", "firewall.nft.check")
	rendered := renderFirewall(cfg)
	adapter := platformAdapterState("firewall", "Firewall", cfg.Enabled)
	adapter.RenderedPaths = append(adapter.RenderedPaths, renderedPath)
	adapter.NativeTargets = append(adapter.NativeTargets, "/etc/nftables.conf")
	adapter.RenderedPaths = append(adapter.RenderedPaths, validationPath, platform.CratePath("state", "rendered", "firewall.json"))

	validationIssues := validateFirewallConfig(cfg)
	if len(validationIssues) > 0 {
		adapter.Validation = "failed"
		adapter.ValidationErr = strings.Join(validationIssues, "; ")
		adapter.Apply = "blocked"
		issues = append(issues, adapter.ValidationErr)
	}
	if len(validationIssues) == 0 {
		if action, err := writeManagedArtifact(
			"firewall/nftables.conf",
			renderedPath,
			rendered,
			"firewall",
			"rendered firewall rules",
		); err != nil {
			issues = append(issues, err.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	}
	if !cfg.Enabled {
		adapter.Summary = "firewall disabled; rendered permissive fallback ruleset"
	} else {
		adapter.Summary = fmt.Sprintf("managed %d firewall allow rules", len(cfg.Allow))
	}
	if len(validationIssues) == 0 {
		if validationIssue := validateRenderedFirewall(cfg, renderedPath, validationPath); validationIssue != "" {
			adapter.Validation = "failed"
			adapter.ValidationErr = validationIssue
			adapter.Apply = "blocked"
			issues = append(issues, validationIssue)
		} else {
			if cfg.Enabled {
				adapter.Validation = "ok"
			} else {
				adapter.Validation = "disabled"
			}
			applyActions, applyIssues, applyStatus, applyErr := applyFirewall(cfg, renderedPath)
			actions = append(actions, applyActions...)
			issues = append(issues, applyIssues...)
			adapter.Apply = applyStatus
			adapter.ApplyErr = applyErr
		}
	}
	if !cfg.Enabled {
		adapter.ValidationErr = ""
		if strings.TrimSpace(adapter.Apply) == "" || adapter.Apply == "pending" {
			adapter.Apply = "disabled"
		}
	}
	effectiveInput := normalizePolicy(cfg.DefaultInput, "drop")
	effectiveForward := normalizePolicy(cfg.DefaultForward, "drop")
	effectiveOutput := normalizePolicy(cfg.DefaultOutput, "accept")
	if !cfg.Enabled {
		effectiveInput = "accept"
		effectiveForward = "accept"
		effectiveOutput = "accept"
	}
	summary := map[string]interface{}{
		"enabled":         cfg.Enabled,
		"backend":         cfg.Backend,
		"default_input":   effectiveInput,
		"default_forward": effectiveForward,
		"default_output":  effectiveOutput,
		"allow_count":     len(cfg.Allow),
		"allow":           cfg.Allow,
		"ssh_rate_limit": map[string]interface{}{
			"enabled":      cfg.RateLimit.SSH.Enabled,
			"max_attempts": maxInt(cfg.RateLimit.SSH.MaxAttempts, 1),
			"window":       strings.TrimSpace(cfg.RateLimit.SSH.Window),
		},
		"icmp_allow_ping":  cfg.ICMP.AllowPing,
		"validation":       adapter.Validation,
		"validation_error": adapter.ValidationErr,
		"apply":            adapter.Apply,
		"apply_error":      adapter.ApplyErr,
	}
	if data, err := json.MarshalIndent(summary, "", "  "); err == nil {
		if action, err := writeManagedArtifact(
			"firewall/state.json",
			platform.CratePath("state", "rendered", "firewall.json"),
			string(data)+"\n",
			"firewall",
			"rendered firewall state",
		); err != nil {
			issues = append(issues, err.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	}
	return actions, finalizePlatformAdapterState(adapter, issues)
}

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

func writeManagedArtifact(logicalName, path, content, component, description string) (*Action, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		log.Printf("warning: ensure managed artifact dir %s: %v", path, err)
		return nil, fmt.Errorf("ensure managed artifact dir %s: %w", path, err)
	}

	current, err := os.ReadFile(path)
	if err == nil && string(current) == content {
		return nil, nil
	}

	if err := snapshotManagedFile(logicalName, path); err != nil {
		log.Printf("warning: snapshot %s: %v", logicalName, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		log.Printf("warning: write managed artifact %s: %v", path, err)
		return nil, fmt.Errorf("write managed artifact %s: %w", path, err)
	}

	return &Action{
		Description: description,
		Component:   component,
		Target:      path,
	}, nil
}

func snapshotManagedFile(logicalName, sourcePath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	snapshotPath := platform.CratePath("state", "last-good", filepath.FromSlash(logicalName))
	if err := os.MkdirAll(filepath.Dir(snapshotPath), 0755); err != nil {
		return err
	}

	metadataPath := snapshotPath + ".meta.json"
	meta := map[string]string{
		"logical_name": logicalName,
		"source_path":  sourcePath,
		"saved_at":     time.Now().UTC().Format(time.RFC3339),
	}
	if encoded, err := json.MarshalIndent(meta, "", "  "); err == nil {
		_ = os.WriteFile(metadataPath, append(encoded, '\n'), 0644)
	}

	return os.WriteFile(snapshotPath, data, 0644)
}

func applyReverseProxy(cfg config.ReverseProxyConfig, renderedPath string) ([]Action, []string, string, string) {
	if runtime.GOOS != "linux" {
		return nil, nil, "skipped", ""
	}
	nativePath := "/etc/nginx/conf.d/crateos-generated.conf"
	if !cfg.Enabled {
		if action, err := removeManagedArtifact("native/reverse-proxy/nginx.conf", nativePath, "proxy", "removed native reverse proxy config"); err != nil {
			return nil, []string{err.Error()}, "failed", err.Error()
		} else if action != nil {
			return []Action{*action}, nil, "removed", ""
		}
		return nil, nil, "disabled", ""
	}
	if action, err := syncManagedArtifact("native/reverse-proxy/nginx.conf", renderedPath, nativePath, "proxy", "applied native reverse proxy config"); err != nil {
		return nil, []string{err.Error()}, "failed", err.Error()
	} else if action != nil {
		actions := []Action{*action}
		if err := systemctl("reload", "nginx"); err == nil {
			actions = append(actions, Action{
				Description: "reloaded nginx after reverse proxy apply",
				Component:   "proxy",
				Target:      "nginx.service",
			})
		} else {
			log.Printf("warning: nginx reload failed")
			return actions, []string{"reload nginx after reverse proxy apply failed"}, "failed", "reload nginx after reverse proxy apply failed"
		}
		return actions, nil, "applied", ""
	}
	return nil, nil, "unchanged", ""
}

func applyFirewall(cfg config.FirewallConfig, renderedPath string) ([]Action, []string, string, string) {
	if runtime.GOOS != "linux" {
		return nil, nil, "skipped", ""
	}
	nativePath := "/etc/nftables.conf"
	if action, err := syncManagedArtifact("native/firewall/nftables.conf", renderedPath, nativePath, "firewall", "applied native firewall config"); err != nil {
		return nil, []string{err.Error()}, "failed", err.Error()
	} else if action != nil {
		actions := []Action{*action}
		if exec.Command("nft", "-f", nativePath).Run() == nil {
			actions = append(actions, Action{
				Description: "applied nftables rules from native config",
				Component:   "firewall",
				Target:      nativePath,
			})
		} else {
			log.Printf("warning: nft apply failed for %s", nativePath)
			return actions, []string{"nft apply failed for rendered firewall config"}, "failed", "nft apply failed for rendered firewall config"
		}
		if cfg.Enabled {
			return actions, nil, "applied", ""
		}
		return actions, nil, "disabled", ""
	}
	if cfg.Enabled {
		return nil, nil, "unchanged", ""
	}
	return nil, nil, "disabled", ""
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

func syncManagedArtifact(logicalName, sourcePath, targetPath, component, description string) (*Action, error) {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		log.Printf("warning: read managed source %s: %v", sourcePath, err)
		return nil, fmt.Errorf("read managed source %s: %w", sourcePath, err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		log.Printf("warning: ensure native target dir %s: %v", targetPath, err)
		return nil, fmt.Errorf("ensure native target dir %s: %w", targetPath, err)
	}
	if current, err := os.ReadFile(targetPath); err == nil && string(current) == string(data) {
		return nil, nil
	}
	if err := snapshotManagedFile(logicalName, targetPath); err != nil {
		log.Printf("warning: snapshot native target %s: %v", targetPath, err)
	}
	mode := os.FileMode(0644)
	if strings.HasSuffix(strings.ToLower(targetPath), ".nmconnection") {
		mode = 0600
	}
	if err := os.WriteFile(targetPath, data, mode); err != nil {
		log.Printf("warning: write native target %s: %v", targetPath, err)
		return nil, fmt.Errorf("write native target %s: %w", targetPath, err)
	}
	return &Action{
		Description: description,
		Component:   component,
		Target:      targetPath,
	}, nil
}

func removeManagedArtifact(logicalName, targetPath, component, description string) (*Action, error) {
	if _, err := os.Stat(targetPath); err != nil {
		return nil, nil
	}
	if err := snapshotManagedFile(logicalName, targetPath); err != nil {
		log.Printf("warning: snapshot native target %s: %v", targetPath, err)
	}
	if err := os.Remove(targetPath); err != nil {
		log.Printf("warning: remove native target %s: %v", targetPath, err)
		return nil, fmt.Errorf("remove native target %s: %w", targetPath, err)
	}
	return &Action{
		Description: description,
		Component:   component,
		Target:      targetPath,
	}, nil
}

func renderReverseProxyNginx(cfg config.ReverseProxyConfig) string {
	var b strings.Builder
	b.WriteString("# Generated by CrateOS. Do not edit manually.\n")
	b.WriteString(fmt.Sprintf("# enabled=%t mappings=%d\n\n", cfg.Enabled, len(cfg.Mappings)))
	if !cfg.Enabled {
		b.WriteString("# Reverse proxy disabled.\n")
		return b.String()
	}

	for _, mapping := range sortedMappings(cfg.Mappings) {
		path := mapping.Path
		if strings.TrimSpace(path) == "" {
			path = "/"
		}
		hostname := strings.TrimSpace(mapping.Hostname)
		if hostname == "" {
			hostname = "_"
		}
		httpsEnabled := mapping.SSL || cfg.Defaults.SSL
		b.WriteString("server {\n")
		b.WriteString(fmt.Sprintf("    listen %d;\n", cfg.Defaults.ListenHTTP))
		b.WriteString(fmt.Sprintf("    server_name %s;\n", hostname))
		b.WriteString(renderReverseProxyLocation(path, mapping))
		if httpsEnabled && cfg.Defaults.ListenHTTPS > 0 {
			b.WriteString(fmt.Sprintf("    return 301 https://$host:%d$request_uri;\n", cfg.Defaults.ListenHTTPS))
		}
		b.WriteString("}\n\n")
		if httpsEnabled && cfg.Defaults.ListenHTTPS > 0 {
			b.WriteString("server {\n")
			b.WriteString(fmt.Sprintf("    listen %d ssl;\n", cfg.Defaults.ListenHTTPS))
			b.WriteString(fmt.Sprintf("    server_name %s;\n", hostname))
			b.WriteString(fmt.Sprintf("    ssl_certificate %s;\n", cfg.Defaults.SSLCert))
			b.WriteString(fmt.Sprintf("    ssl_certificate_key %s;\n", cfg.Defaults.SSLKey))
			b.WriteString(renderReverseProxyLocation(path, mapping))
			b.WriteString("}\n\n")
		}
	}
	if len(cfg.Mappings) == 0 {
		b.WriteString("# No reverse proxy mappings defined.\n")
	}
	return b.String()
}

func renderReverseProxyLocation(path string, mapping config.ProxyMapping) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("    location %s {\n", path))
	b.WriteString(fmt.Sprintf("        proxy_pass %s;\n", mapping.Target))
	headers := sortedHeaderKeys(mapping.Headers)
	for _, key := range headers {
		b.WriteString(fmt.Sprintf("        proxy_set_header %s %s;\n", key, mapping.Headers[key]))
	}
	b.WriteString("    }\n")
	return b.String()
}

func renderFirewall(cfg config.FirewallConfig) string {
	var b strings.Builder
	b.WriteString("# Generated by CrateOS. Do not edit manually.\n")
	b.WriteString("table inet crateos {\n")
	b.WriteString("    chain input {\n")
	b.WriteString("        type filter hook input priority 0;\n")
	defaultInput := normalizePolicy(cfg.DefaultInput, "drop")
	defaultForward := normalizePolicy(cfg.DefaultForward, "drop")
	defaultOutput := normalizePolicy(cfg.DefaultOutput, "accept")
	if !cfg.Enabled {
		defaultInput = "accept"
		defaultForward = "accept"
		defaultOutput = "accept"
	}
	b.WriteString(fmt.Sprintf("        policy %s;\n", defaultInput))
	b.WriteString("        iif lo accept\n")
	b.WriteString("        ct state established,related accept\n")
	if cfg.ICMP.AllowPing || !cfg.Enabled {
		b.WriteString("        ip protocol icmp accept\n")
		b.WriteString("        ip6 nexthdr ipv6-icmp accept\n")
	}
	if cfg.Enabled {
		for _, rule := range cfg.Allow {
			comment := sanitizeComment(rule.Name)
			if expr := firewallSourceExpression(rule.Source); expr != "" {
				b.WriteString(fmt.Sprintf("        %s %s dport %d accept comment \"%s\"\n", expr, normalizeProto(rule.Protocol), rule.Port, comment))
			} else {
				b.WriteString(fmt.Sprintf("        %s dport %d accept comment \"%s\"\n", normalizeProto(rule.Protocol), rule.Port, comment))
			}
		}
		if cfg.RateLimit.SSH.Enabled {
			b.WriteString(fmt.Sprintf("        tcp dport 22 ct state new limit rate %d/minute accept comment \"ssh-rate-limit\"\n", maxInt(cfg.RateLimit.SSH.MaxAttempts, 1)))
		}
	}
	b.WriteString("    }\n")
	b.WriteString("    chain forward {\n")
	b.WriteString("        type filter hook forward priority 0;\n")
	b.WriteString(fmt.Sprintf("        policy %s;\n", defaultForward))
	b.WriteString("    }\n")
	b.WriteString("    chain output {\n")
	b.WriteString("        type filter hook output priority 0;\n")
	b.WriteString(fmt.Sprintf("        policy %s;\n", defaultOutput))
	b.WriteString("    }\n")
	b.WriteString("}\n")
	return b.String()
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

func sortedMappings(mappings []config.ProxyMapping) []config.ProxyMapping {
	sorted := append([]config.ProxyMapping(nil), mappings...)
	sort.Slice(sorted, func(i, j int) bool {
		left := sorted[i].Hostname + "/" + sorted[i].Path + "/" + sorted[i].Name
		right := sorted[j].Hostname + "/" + sorted[j].Path + "/" + sorted[j].Name
		return left < right
	})
	return sorted
}

func sortedHeaderKeys(headers map[string]string) []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func validateFirewallConfig(cfg config.FirewallConfig) []string {
	var issues []string
	if backend := strings.TrimSpace(strings.ToLower(cfg.Backend)); backend != "" && backend != "nftables" {
		issues = append(issues, "firewall backend must be nftables")
	}
	for _, policy := range []struct {
		name  string
		value string
		def   string
	}{
		{name: "default_input", value: cfg.DefaultInput, def: "drop"},
		{name: "default_forward", value: cfg.DefaultForward, def: "drop"},
		{name: "default_output", value: cfg.DefaultOutput, def: "accept"},
	} {
		normalized := normalizePolicy(policy.value, "")
		if normalized == "" {
			issues = append(issues, fmt.Sprintf("%s must be accept or drop", policy.name))
		}
	}
	if cfg.RateLimit.SSH.Enabled {
		if maxInt(cfg.RateLimit.SSH.MaxAttempts, 0) <= 0 {
			issues = append(issues, "firewall ssh rate limit requires max_attempts > 0")
		}
		if strings.TrimSpace(cfg.RateLimit.SSH.Window) != "" {
			if _, err := time.ParseDuration(strings.TrimSpace(cfg.RateLimit.SSH.Window)); err != nil {
				issues = append(issues, "firewall ssh rate limit window must be a valid duration")
			}
		}
	}
	seen := make(map[string]string, len(cfg.Allow))
	for i, rule := range cfg.Allow {
		label := firewallRuleLabel(rule, i)
		if rule.Port < 1 || rule.Port > 65535 {
			issues = append(issues, fmt.Sprintf("%s port must be between 1 and 65535", label))
		}
		protocol := strings.ToLower(strings.TrimSpace(rule.Protocol))
		if protocol == "" {
			protocol = "tcp"
		}
		if protocol != "tcp" && protocol != "udp" {
			issues = append(issues, fmt.Sprintf("%s protocol must be tcp or udp", label))
		}
		if err := validateFirewallSource(rule.Source); err != nil {
			issues = append(issues, fmt.Sprintf("%s %s", label, err.Error()))
		}
		key := fmt.Sprintf("%s|%d|%s", strings.ToLower(strings.TrimSpace(rule.Source)), rule.Port, protocol)
		if previous, exists := seen[key]; exists {
			issues = append(issues, fmt.Sprintf("%s duplicates %s", label, previous))
		} else {
			seen[key] = label
		}
	}
	return issues
}

func validateRenderedFirewall(cfg config.FirewallConfig, renderedPath string, validationPath string) string {
	if runtime.GOOS != "linux" {
		return ""
	}
	if _, err := writeManagedArtifact(
		"firewall/nftables.check",
		validationPath,
		renderFirewallValidationConfig(renderedPath),
		"firewall",
		"rendered firewall validation config",
	); err != nil {
		return err.Error()
	}
	cmd := exec.Command("nft", "-c", "-f", validationPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("warning: firewall validation failed for %s: %v", renderedPath, err)
		return summarizeCommandFailure("nft validation failed for rendered firewall config", output)
	}
	if cfg.Enabled {
		return ""
	}
	return ""
}

func renderFirewallValidationConfig(renderedPath string) string {
	return strings.Join([]string{
		"# Generated by CrateOS for firewall validation.",
		fmt.Sprintf("include \"%s\"", filepath.ToSlash(renderedPath)),
		"",
	}, "\n")
}

func firewallRuleLabel(rule config.FirewallRule, index int) string {
	if strings.TrimSpace(rule.Name) != "" {
		return fmt.Sprintf("firewall rule %q", rule.Name)
	}
	return fmt.Sprintf("firewall rule #%d", index+1)
}

func validateFirewallSource(source string) error {
	source = strings.TrimSpace(strings.ToLower(source))
	switch source {
	case "", "any", "lan", "vpn", "local":
		return nil
	}
	if _, _, err := net.ParseCIDR(source); err == nil {
		return nil
	}
	if ip := net.ParseIP(source); ip != nil {
		return nil
	}
	return fmt.Errorf("source must be any, lan, vpn, local, an IP, or CIDR")
}

func firewallSourceExpression(source string) string {
	source = strings.TrimSpace(strings.ToLower(source))
	switch source {
	case "", "any":
		return ""
	case "lan":
		return "ip saddr { 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 } ip6 saddr { fc00::/7, fe80::/10 }"
	case "vpn":
		return "ip saddr { 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 100.64.0.0/10 } ip6 saddr fc00::/7"
	case "local":
		return "ip saddr 127.0.0.0/8 ip6 saddr ::1"
	default:
		if strings.Contains(source, ":") {
			return fmt.Sprintf("ip6 saddr %s", source)
		}
		return fmt.Sprintf("ip saddr %s", source)
	}
}

func validateReverseProxyConfig(cfg config.ReverseProxyConfig) []string {
	if !cfg.Enabled {
		return nil
	}
	var issues []string
	if cfg.Defaults.ListenHTTP <= 0 && cfg.Defaults.ListenHTTPS <= 0 {
		issues = append(issues, "reverse proxy must expose at least one listen port")
	}
	if countReverseProxyHTTPSMappings(cfg) > 0 {
		if cfg.Defaults.ListenHTTPS <= 0 {
			issues = append(issues, "https mappings require defaults.listen_https")
		}
		if strings.TrimSpace(cfg.Defaults.SSLCert) == "" {
			issues = append(issues, "https mappings require defaults.ssl_cert")
		}
		if strings.TrimSpace(cfg.Defaults.SSLKey) == "" {
			issues = append(issues, "https mappings require defaults.ssl_key")
		}
	}
	seen := make(map[string]string, len(cfg.Mappings))
	for i, mapping := range cfg.Mappings {
		label := reverseProxyMappingLabel(mapping, i)
		if strings.TrimSpace(mapping.Target) == "" {
			issues = append(issues, fmt.Sprintf("%s missing target", label))
		} else if !strings.HasPrefix(mapping.Target, "http://") && !strings.HasPrefix(mapping.Target, "https://") {
			issues = append(issues, fmt.Sprintf("%s target must start with http:// or https://", label))
		}
		if strings.TrimSpace(mapping.Path) != "" && !strings.HasPrefix(mapping.Path, "/") {
			issues = append(issues, fmt.Sprintf("%s path must start with /", label))
		}
		if strings.TrimSpace(mapping.Hostname) != "" && !isValidReverseProxyHostname(mapping.Hostname) {
			issues = append(issues, fmt.Sprintf("%s hostname is invalid", label))
		}
		key := strings.ToLower(strings.TrimSpace(mapping.Hostname)) + "|" + normalizeReverseProxyPath(mapping.Path)
		if previous, exists := seen[key]; exists {
			issues = append(issues, fmt.Sprintf("%s duplicates host/path already used by %s", label, previous))
		} else {
			seen[key] = label
		}
	}
	return issues
}

func validateRenderedReverseProxy(cfg config.ReverseProxyConfig, renderedPath string, validationPath string) string {
	if !cfg.Enabled || !cfg.ValidateBeforeApply || runtime.GOOS != "linux" {
		return ""
	}
	validationConfig := renderReverseProxyValidationConfig(renderedPath)
	if _, err := writeManagedArtifact(
		"reverse-proxy/nginx-validation.conf",
		validationPath,
		validationConfig,
		"proxy",
		"rendered reverse proxy validation config",
	); err != nil {
		return err.Error()
	}
	cmd := exec.Command("nginx", "-t", "-c", validationPath, "-p", "/")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("warning: nginx validation failed for %s: %v", renderedPath, err)
		return summarizeCommandFailure("nginx validation failed for rendered reverse proxy config", output)
	}
	return ""
}

func renderReverseProxyValidationConfig(renderedPath string) string {
	return strings.Join([]string{
		"# Generated by CrateOS for reverse proxy validation.",
		"events {}",
		"http {",
		"    include /etc/nginx/mime.types;",
		fmt.Sprintf("    include %s;", renderedPath),
		"}",
		"",
	}, "\n")
}

func summarizeCommandFailure(prefix string, output []byte) string {
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return prefix
	}
	trimmed = strings.ReplaceAll(trimmed, "\n", " | ")
	return prefix + ": " + trimmed
}

func countReverseProxyHTTPSMappings(cfg config.ReverseProxyConfig) int {
	count := 0
	for _, mapping := range cfg.Mappings {
		if mapping.SSL || cfg.Defaults.SSL {
			count++
		}
	}
	return count
}

func reverseProxyMappingLabel(mapping config.ProxyMapping, index int) string {
	if strings.TrimSpace(mapping.Name) != "" {
		return fmt.Sprintf("mapping %q", mapping.Name)
	}
	if strings.TrimSpace(mapping.Hostname) != "" {
		return fmt.Sprintf("mapping %q", mapping.Hostname)
	}
	return fmt.Sprintf("mapping #%d", index+1)
}

func normalizeReverseProxyPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return "/"
	}
	return path
}

var reverseProxyHostnamePattern = regexp.MustCompile(`^[A-Za-z0-9*.-]+$`)

func isValidReverseProxyHostname(hostname string) bool {
	hostname = strings.TrimSpace(hostname)
	if hostname == "" {
		return true
	}
	return reverseProxyHostnamePattern.MatchString(hostname)
}

func sanitizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	if value == "" {
		return "unnamed"
	}
	return value
}

func sanitizeComment(value string) string {
	value = strings.ReplaceAll(value, "\"", "")
	if strings.TrimSpace(value) == "" {
		return "crateos-rule"
	}
	return value
}

func normalizeProto(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "udp":
		return "udp"
	default:
		return "tcp"
	}
}

func normalizePolicy(value string, fallback string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "accept":
		return "accept"
	case "drop":
		return "drop"
	default:
		return fallback
	}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func maxInt(v int, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}

func reconcileUsers(cfg *config.Config) ([]Action, PlatformAdapterState) {
	var actions []Action
	var issues []string
	adapter := platformAdapterState("users", "User Provisioning", true)

	if runtime.GOOS != "linux" {
		adapter.Summary = "user provisioning not supported on non-Linux platforms"
		return actions, finalizePlatformAdapterState(adapter, issues)
	}

	// Perform user provisioning
	reconciled, provState, err := users.ProvisionUsers(cfg)
	if err != nil {
		issues = append(issues, fmt.Sprintf("user provisioning failed: %v", err))
		adapter.Validation = "failed"
		adapter.ValidationErr = err.Error()
	} else {
		adapter.Validation = "ok"
	}

	// Collect issues from provisioning
	for _, issue := range provState.Issues {
		issues = append(issues, issue)
	}

	// Write provisioning state
	renderedPath := platform.CratePath("state", "rendered", "user-provisioning.json")
	adapter.RenderedPaths = append(adapter.RenderedPaths, renderedPath)

	if data, marshalErr := json.MarshalIndent(provState, "", "  "); marshalErr == nil {
		if action, writeErr := writeManagedArtifact(
			"users/provisioning.json",
			renderedPath,
			string(data)+"\n",
			"users",
			"rendered user provisioning state",
		); writeErr != nil {
			issues = append(issues, writeErr.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	} else {
		issues = append(issues, fmt.Sprintf("failed to marshal provisioning state: %v", marshalErr))
	}

	adapter.Summary = provState.Summary
	if len(reconciled) > 0 {
		adapter.Apply = "ok"
	}

	return actions, finalizePlatformAdapterState(adapter, issues)
}
