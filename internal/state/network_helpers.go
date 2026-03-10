package state

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/crateos/crateos/internal/config"
)

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

func sanitizeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	if value == "" {
		return "unnamed"
	}
	return value
}
