package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
)

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
