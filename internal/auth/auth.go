package auth

import (
	"strings"

	"github.com/crateos/crateos/internal/config"
)

// User represents a resolved user with permissions.
type User struct {
	Name        string
	Role        string
	Allow       map[string]bool
	Deny        map[string]bool
}

// Authz holds in-memory auth context.
type Authz struct {
	Roles map[string]map[string]bool
	Users map[string]*User
}

// Load builds an Authz model from config.Users.
func Load(cfg *config.Config) *Authz {
	a := &Authz{
		Roles: make(map[string]map[string]bool),
		Users: make(map[string]*User),
	}

	// Load roles
	for roleName, role := range cfg.Users.Roles {
		perms := make(map[string]bool, len(role.Permissions))
		for _, p := range role.Permissions {
			perms[p] = true
		}
		a.Roles[roleName] = perms
	}

	// Load users
	for _, u := range cfg.Users.Users {
		perms := a.Roles[u.Role]
		if perms == nil {
			perms = map[string]bool{}
		}
		allow, deny := mergeUserOverrides(perms, u.Permissions)
		a.Users[u.Name] = &User{
			Name:  u.Name,
			Role:  u.Role,
			Allow: allow,
			Deny:  deny,
		}
	}
	return a
}

// Check returns true if user has the given permission (supports wildcard "*").
func (a *Authz) Check(userName, perm string) bool {
	u, ok := a.Users[userName]
	if !ok {
		return false
	}
	if matches(u.Deny, perm) {
		return false
	}
	return matches(u.Allow, perm)
}

// UserRole returns the role name for a user.
func (a *Authz) UserRole(userName string) string {
	if u, ok := a.Users[userName]; ok {
		return u.Role
	}
	return ""
}

func mergeUserOverrides(rolePerms map[string]bool, overrides []string) (map[string]bool, map[string]bool) {
	allow := make(map[string]bool)
	deny := make(map[string]bool)
	for k, v := range rolePerms {
		if v {
			allow[k] = true
		}
	}
	for _, p := range overrides {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, "-") {
			key := strings.TrimPrefix(p, "-")
			deny[key] = true
			delete(allow, key)
			continue
		}
		allow[p] = true
	}
	return allow, deny
}

func matches(perms map[string]bool, perm string) bool {
	if perms["*"] {
		return true
	}
	if perms[perm] {
		return true
	}
	for p := range perms {
		if len(p) > 0 && p[len(p)-1] == '*' {
			prefix := p[:len(p)-1]
			if len(prefix) > 0 && strings.HasPrefix(perm, prefix) {
				return true
			}
		}
	}
	return false
}
