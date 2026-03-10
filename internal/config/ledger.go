package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/crateos/crateos/internal/platform"
)

type ConfigChangeLedger struct {
	GeneratedAt string                     `json:"generated_at"`
	Files       map[string]ConfigFileState `json:"files"`
}

type ConfigFileState struct {
	File          string `json:"file"`
	Path          string `json:"path"`
	Exists        bool   `json:"exists"`
	Monitoring    string `json:"monitoring"`
	LastWriter    string `json:"last_writer,omitempty"`
	CurrentHash   string `json:"current_hash,omitempty"`
	LastSeenAt    string `json:"last_seen_at,omitempty"`
	LastChangedAt string `json:"last_changed_at,omitempty"`
}

func configChangeLedgerPath() string {
	return platform.CratePath("state", "config-change-ledger.json")
}

func loadConfigChangeLedger() (ConfigChangeLedger, error) {
	path := configChangeLedgerPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ConfigChangeLedger{Files: map[string]ConfigFileState{}}, nil
		}
		return ConfigChangeLedger{}, err
	}
	var ledger ConfigChangeLedger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return ConfigChangeLedger{}, err
	}
	if ledger.Files == nil {
		ledger.Files = map[string]ConfigFileState{}
	}
	return ledger, nil
}

// LoadConfigChangeLedger returns the persisted config change ledger without mutating it.
func LoadConfigChangeLedger() (ConfigChangeLedger, error) {
	return loadConfigChangeLedger()
}

func saveConfigChangeLedger(ledger ConfigChangeLedger) error {
	ledger.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	if ledger.Files == nil {
		ledger.Files = map[string]ConfigFileState{}
	}
	data, err := json.MarshalIndent(ledger, "", "  ")
	if err != nil {
		return err
	}
	path := configChangeLedgerPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func auditConfigChanges(paths []string) error {
	ledger, err := loadConfigChangeLedger()
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, path := range paths {
		file := filepath.Base(path)
		record := ledger.Files[file]
		record.File = file
		record.Path = path
		record.LastSeenAt = now

		hash, exists, err := configFileHash(path)
		if err != nil {
			return err
		}
		if !exists {
			record.Exists = false
			record.CurrentHash = ""
			if record.LastChangedAt == "" {
				record.LastChangedAt = now
			}
			ledger.Files[file] = record
			continue
		}

		record.Exists = true
		if record.CurrentHash != "" && record.CurrentHash != hash {
			record.Monitoring = "unmonitored"
			record.LastWriter = "external"
			record.LastChangedAt = now
		} else if record.Monitoring == "" {
			record.Monitoring = "monitored"
			if record.LastChangedAt == "" {
				record.LastChangedAt = now
			}
		}
		record.CurrentHash = hash
		ledger.Files[file] = record
	}
	return saveConfigChangeLedger(ledger)
}

func trackManagedConfigWrite(path string, writer string) error {
	ledger, err := loadConfigChangeLedger()
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	hash, exists, err := configFileHash(path)
	if err != nil {
		return err
	}
	record := ledger.Files[filepath.Base(path)]
	record.File = filepath.Base(path)
	record.Path = path
	record.Exists = exists
	record.Monitoring = "monitored"
	record.LastWriter = writer
	record.CurrentHash = hash
	record.LastSeenAt = now
	record.LastChangedAt = now
	ledger.Files[record.File] = record
	return saveConfigChangeLedger(ledger)
}

func configFileHash(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), true, nil
}
