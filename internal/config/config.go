package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/crateos/crateos/internal/platform"
)

// ── Top-level config bundle ─────────────────────────────────────────

// Config holds all parsed YAML configs as a single desired-state snapshot.
type Config struct {
	CrateOS      CrateOSConfig      `json:"crateos"`
	Network      NetworkConfig       `json:"network"`
	Firewall     FirewallConfig      `json:"firewall"`
	Services     ServicesConfig      `json:"services"`
	Users        UsersConfig         `json:"users"`
	ReverseProxy ReverseProxyConfig  `json:"reverse_proxy"`
}

// ── Per-file structs ────────────────────────────────────────────────

type CrateOSConfig struct {
	Version  string `yaml:"version"`
	Platform struct {
		Hostname string `yaml:"hostname"`
		Timezone string `yaml:"timezone"`
		Locale   string `yaml:"locale"`
	} `yaml:"platform"`
	CrateRoot string `yaml:"crate_root"`
	LogLevel  string `yaml:"log_level"`
	Updates   struct {
		Enabled       bool   `yaml:"enabled"`
		Channel       string `yaml:"channel"`
		AutoApply     bool   `yaml:"auto_apply"`
		CheckInterval string `yaml:"check_interval"`
	} `yaml:"updates"`
	Maintenance struct {
		CleanupEnabled    bool   `yaml:"cleanup_enabled"`
		JournalVacuumSize string `yaml:"journal_vacuum_size"`
		JournalVacuumTime string `yaml:"journal_vacuum_time"`
		DockerPrune       bool   `yaml:"docker_prune"`
		CacheCleanup      bool   `yaml:"cache_cleanup"`
	} `yaml:"maintenance"`
}

type NetworkProfile struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	MAC         string `yaml:"mac"`
	SSID        string `yaml:"ssid"`
	Password    string `yaml:"password"`
	Method      string `yaml:"method"`
	Metric      int    `yaml:"metric"`
	Autoconnect bool   `yaml:"autoconnect"`
	Static      struct {
		Address string   `yaml:"address"`
		Gateway string   `yaml:"gateway"`
		DNS     []string `yaml:"dns"`
	} `yaml:"static"`
}

type NetworkConfig struct {
	Manager  string           `yaml:"manager"`
	Profiles []NetworkProfile `yaml:"profiles"`
	DNS      struct {
		Fallback []string `yaml:"fallback"`
	} `yaml:"dns"`
	SelfHeal bool `yaml:"self_heal"`
}

type FirewallRule struct {
	Name     string `yaml:"name"`
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
	Source   string `yaml:"source"`
}

type FirewallConfig struct {
	Enabled        bool           `yaml:"enabled"`
	Backend        string         `yaml:"backend"`
	DefaultInput   string         `yaml:"default_input"`
	DefaultForward string         `yaml:"default_forward"`
	DefaultOutput  string         `yaml:"default_output"`
	Allow          []FirewallRule `yaml:"allow"`
	RateLimit      struct {
		SSH struct {
			Enabled     bool   `yaml:"enabled"`
			MaxAttempts int    `yaml:"max_attempts"`
			Window      string `yaml:"window"`
		} `yaml:"ssh"`
	} `yaml:"rate_limit"`
	ICMP struct {
		AllowPing bool `yaml:"allow_ping"`
	} `yaml:"icmp"`
}

type ServiceEntry struct {
	Name      string            `yaml:"name"`
	Enabled   bool              `yaml:"enabled"`
	Runtime   string            `yaml:"runtime"`
	Autostart bool              `yaml:"autostart"`
	Options   map[string]string `yaml:"options"`
}

type ServicesConfig struct {
	Services []ServiceEntry `yaml:"services"`
}

type Role struct {
	Description string   `yaml:"description"`
	Permissions []string `yaml:"permissions"`
}

type UserEntry struct {
	Name string `yaml:"name"`
	Role string `yaml:"role"`
}

type UsersConfig struct {
	Roles map[string]Role `yaml:"roles"`
	Users []UserEntry     `yaml:"users"`
}

type ProxyMapping struct {
	Name     string            `yaml:"name"`
	Hostname string            `yaml:"hostname"`
	Target   string            `yaml:"target"`
	Path     string            `yaml:"path"`
	SSL      bool              `yaml:"ssl"`
	Headers  map[string]string `yaml:"headers"`
}

type ReverseProxyConfig struct {
	Enabled  bool `yaml:"enabled"`
	Defaults struct {
		ListenHTTP  int    `yaml:"listen_http"`
		ListenHTTPS int    `yaml:"listen_https"`
		SSL         bool   `yaml:"ssl"`
		SSLCert     string `yaml:"ssl_cert"`
		SSLKey      string `yaml:"ssl_key"`
	} `yaml:"defaults"`
	Mappings    []ProxyMapping `yaml:"mappings"`
	HealthCheck struct {
		Enabled  bool   `yaml:"enabled"`
		Interval string `yaml:"interval"`
		Timeout  string `yaml:"timeout"`
	} `yaml:"health_check"`
	ValidateBeforeApply bool `yaml:"validate_before_apply"`
}

// ── Loader ──────────────────────────────────────────────────────────

// Load reads all YAML configs from the crate config directory.
func Load() (*Config, error) {
	configDir := platform.CratePath("config")
	cfg := &Config{}

	loaders := []struct {
		file   string
		target interface{}
	}{
		{"crateos.yaml", &cfg.CrateOS},
		{"network.yaml", &cfg.Network},
		{"firewall.yaml", &cfg.Firewall},
		{"services.yaml", &cfg.Services},
		{"users.yaml", &cfg.Users},
		{"reverse-proxy.yaml", &cfg.ReverseProxy},
	}

	for _, l := range loaders {
		path := filepath.Join(configDir, l.file)
		if err := loadYAML(path, l.target); err != nil {
			if os.IsNotExist(err) {
				continue // missing config files are acceptable
			}
			return nil, fmt.Errorf("loading %s: %w", l.file, err)
		}
	}

	return cfg, nil
}

func loadYAML(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}
