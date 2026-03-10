package state

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

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
