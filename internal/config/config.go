package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/crateos/crateos/internal/platform"
)

// ── Top-level config bundle ─────────────────────────────────────────

// Config holds all parsed YAML configs as a single desired-state snapshot.
type Config struct {
	CrateOS      CrateOSConfig      `json:"crateos"`
	Network      NetworkConfig      `json:"network"`
	Firewall     FirewallConfig     `json:"firewall"`
	Services     ServicesConfig     `json:"services"`
	Users        UsersConfig        `json:"users"`
	ReverseProxy ReverseProxyConfig `json:"reverse_proxy"`
}

// ── Per-file structs ────────────────────────────────────────────────

type CrateOSConfig struct {
	Version  string `yaml:"version"`
	Platform struct {
		Hostname string `yaml:"hostname"`
		Timezone string `yaml:"timezone"`
		Locale   string `yaml:"locale"`
	} `yaml:"platform"`
	Access struct {
		SSH struct {
			Enabled bool   `yaml:"enabled"`
			Landing string `yaml:"landing"`
		} `yaml:"ssh"`
		LocalGUI struct {
			Enabled      bool   `yaml:"enabled"`
			Provider     string `yaml:"provider"`
			Landing      string `yaml:"landing"`
			DefaultShell string `yaml:"default_shell"`
		} `yaml:"local_gui"`
		VirtualDesktop struct {
			Enabled  bool   `yaml:"enabled"`
			Provider string `yaml:"provider"`
			Landing  string `yaml:"landing"`
		} `yaml:"virtual_desktop"`
		BreakGlass struct {
			Enabled         bool     `yaml:"enabled"`
			RequirePerm     string   `yaml:"require_permission"`
			AllowedSurfaces []string `yaml:"allowed_surfaces"`
		} `yaml:"break_glass"`
	} `yaml:"access"`
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
	Name      string                 `yaml:"name"`
	Enabled   bool                   `yaml:"enabled"`
	Runtime   string                 `yaml:"runtime"`
	Autostart bool                   `yaml:"autostart"`
	Actor     ServiceActorConfig     `yaml:"actor"`
	Deploy    ServiceDeployConfig    `yaml:"deploy"`
	Execution ServiceExecutionConfig `yaml:"execution"`
	Options   map[string]string      `yaml:"options"`
}

type ServicesConfig struct {
	Services []ServiceEntry `yaml:"services"`
}

type ServiceActorConfig struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type ServiceDeployConfig struct {
	Source      string `yaml:"source"`
	UploadPath  string `yaml:"upload_path"`
	WorkingDir  string `yaml:"working_dir"`
	Entry       string `yaml:"entry"`
	InstallCmd  string `yaml:"install_cmd"`
	EnvFile     string `yaml:"env_file"`
	ArtifactDir string `yaml:"artifact_dir"`
}

type ServiceExecutionConfig struct {
	Mode          string `yaml:"mode"`
	StartCmd      string `yaml:"start_cmd"`
	Schedule      string `yaml:"schedule"`
	Timeout       string `yaml:"timeout"`
	StopTimeout   string `yaml:"stop_timeout"`
	OnTimeout     string `yaml:"on_timeout"`
	KillSignal    string `yaml:"kill_signal"`
	Concurrency   string `yaml:"concurrency"`
	SuccessWindow string `yaml:"success_window"`
}

type Role struct {
	Description string   `yaml:"description"`
	Permissions []string `yaml:"permissions"`
}

type UserEntry struct {
	Name        string   `yaml:"name"`
	Role        string   `yaml:"role"`
	Permissions []string `yaml:"permissions,omitempty"` // allow/deny overrides, e.g. "svc.foo", "-svc.bar"
	Priority    int      `yaml:"priority,omitempty"`    // higher wins if future merges are needed
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
	configPaths := make([]string, 0, 6)

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
		configPaths = append(configPaths, path)
		if err := loadYAML(path, l.target); err != nil {
			if os.IsNotExist(err) {
				continue // missing config files are acceptable
			}
			return nil, fmt.Errorf("loading %s: %w", l.file, err)
		}
	}

	if err := auditConfigChanges(configPaths); err != nil {
		log.Printf("warning: config change audit failed: %v", err)
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

// SaveServices writes the services config back to services.yaml.
func SaveServices(cfg *Config) error {
	path := filepath.Join(platform.CratePath("config"), "services.yaml")
	b, err := yaml.Marshal(cfg.Services)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return err
	}
	if err := OnWebFormSave(path); err != nil {
		return err
	}
	return trackManagedConfigWrite(path, "crateos")
}

// SaveUsers writes the users config back to users.yaml.
func SaveUsers(cfg *Config) error {
	path := filepath.Join(platform.CratePath("config"), "users.yaml")
	b, err := yaml.Marshal(cfg.Users)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return err
	}
	if _, err = NormalizeFileIfNeeded(path); err != nil {
		return err
	}
	return trackManagedConfigWrite(path, "crateos")
}
