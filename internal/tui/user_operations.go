package tui

import (
	"fmt"
	"strings"

	"github.com/crateos/crateos/internal/config"
)

func addUserDirect(name, role string, perms []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	for _, u := range cfg.Users.Users {
		if u.Name == name {
			return fmt.Errorf("user already exists")
		}
	}
	cfg.Users.Users = append(cfg.Users.Users, config.UserEntry{
		Name:        name,
		Role:        role,
		Permissions: perms,
	})
	return config.SaveUsers(cfg)
}

func updateUserDirect(targetName, newName, newRole string, newPerms []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	targetName = strings.TrimSpace(targetName)
	if targetName == "" {
		return fmt.Errorf("target name required")
	}
	updated := false
	for i := range cfg.Users.Users {
		if cfg.Users.Users[i].Name == targetName {
			newName = strings.TrimSpace(newName)
			if newName != "" && newName != targetName {
				for j := range cfg.Users.Users {
					if j != i && cfg.Users.Users[j].Name == newName {
						return fmt.Errorf("user already exists")
					}
				}
				cfg.Users.Users[i].Name = newName
			}
			if newRole != "" {
				cfg.Users.Users[i].Role = newRole
			}
			if newPerms != nil {
				cfg.Users.Users[i].Permissions = newPerms
			}
			updated = true
			break
		}
	}
	if !updated {
		return fmt.Errorf("user not found")
	}
	return config.SaveUsers(cfg)
}

func deleteUserDirect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("user name required")
	}
	var filtered []config.UserEntry
	found := false
	for _, u := range cfg.Users.Users {
		if u.Name != name {
			filtered = append(filtered, u)
		} else {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("user not found")
	}
	cfg.Users.Users = filtered
	return config.SaveUsers(cfg)
}
