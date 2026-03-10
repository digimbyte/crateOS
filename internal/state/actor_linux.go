package state

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/crateos/crateos/internal/config"
)

func linuxCommandExists(name string) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := exec.LookPath(name)
	return err == nil
}

func linuxAccountExists(name string) bool {
	name = strings.TrimSpace(name)
	if runtime.GOOS != "linux" || name == "" {
		return false
	}
	return exec.Command("id", "-u", name).Run() == nil
}

func linuxGroupExists(name string) bool {
	name = strings.TrimSpace(name)
	if runtime.GOOS != "linux" || name == "" {
		return false
	}
	return exec.Command("getent", "group", name).Run() == nil
}

type linuxPasswdEntry struct {
	Name  string
	UID   string
	GID   string
	Group string
	Home  string
	Shell string
}

func readLinuxPasswdEntry(name string) (linuxPasswdEntry, bool) {
	name = strings.TrimSpace(name)
	if runtime.GOOS != "linux" || name == "" {
		return linuxPasswdEntry{}, false
	}
	out, err := exec.Command("getent", "passwd", name).Output()
	if err != nil {
		return linuxPasswdEntry{}, false
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ":")
	if len(parts) < 7 {
		return linuxPasswdEntry{}, false
	}
	groupName := strings.TrimSpace(parts[0])
	if gid := strings.TrimSpace(parts[3]); gid != "" {
		if groupOut, groupErr := exec.Command("getent", "group", gid).Output(); groupErr == nil {
			groupParts := strings.Split(strings.TrimSpace(string(groupOut)), ":")
			if len(groupParts) > 0 && strings.TrimSpace(groupParts[0]) != "" {
				groupName = strings.TrimSpace(groupParts[0])
			}
		}
	}
	return linuxPasswdEntry{
		Name:  strings.TrimSpace(parts[0]),
		UID:   strings.TrimSpace(parts[2]),
		GID:   strings.TrimSpace(parts[3]),
		Group: groupName,
		Home:  strings.TrimSpace(parts[5]),
		Shell: strings.TrimSpace(parts[6]),
	}, true
}

func readLinuxGroupID(name string) string {
	name = strings.TrimSpace(name)
	if runtime.GOOS != "linux" || name == "" {
		return ""
	}
	out, err := exec.Command("getent", "group", name).Output()
	if err != nil {
		return ""
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ":")
	if len(parts) < 3 {
		return ""
	}
	return strings.TrimSpace(parts[2])
}

func readLinuxSupplementaryGroups(name string) []string {
	name = strings.TrimSpace(name)
	if runtime.GOOS != "linux" || name == "" {
		return nil
	}
	out, err := exec.Command("id", "-nG", name).Output()
	if err != nil {
		return nil
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) == 0 {
		return nil
	}
	groups := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		groups = append(groups, part)
	}
	return groups
}

func linuxPathOwnershipIssue(path, expectedUser, expectedGroup string) string {
	path = strings.TrimSpace(path)
	expectedUser = strings.TrimSpace(expectedUser)
	expectedGroup = strings.TrimSpace(expectedGroup)
	if runtime.GOOS != "linux" || path == "" {
		return ""
	}
	info, err := os.Lstat(path)
	if err != nil {
		return ""
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Sprintf("managed actor path %s must not be a symlink", path)
	}
	if !info.IsDir() {
		return fmt.Sprintf("managed actor path %s exists but is not a directory", path)
	}
	if perms := info.Mode().Perm(); perms != 0755 {
		return fmt.Sprintf("managed actor path %s has mode %04o instead of 0755", path, perms)
	}
	return linuxPathEntryOwnershipIssue(path, info, expectedUser, expectedGroup)
}

func linuxPathEntryOwnershipIssue(path string, info os.FileInfo, expectedUser, expectedGroup string) string {
	path = strings.TrimSpace(path)
	expectedUser = strings.TrimSpace(expectedUser)
	expectedGroup = strings.TrimSpace(expectedGroup)
	if runtime.GOOS != "linux" || path == "" || info == nil {
		return ""
	}
	uid, gid, ok := fileInfoOwnershipIDs(info)
	if !ok {
		return fmt.Sprintf("managed actor path %s ownership could not be inspected", path)
	}
	if expectedUser != "" {
		if passwd, exists := readLinuxPasswdEntry(expectedUser); exists && passwd.UID != "" && passwd.UID != fmt.Sprintf("%d", uid) {
			return fmt.Sprintf("managed actor path %s is owned by uid %d instead of actor %s", path, uid, expectedUser)
		}
	}
	if expectedGroup != "" {
		if expectedGID := readLinuxGroupID(expectedGroup); expectedGID != "" && expectedGID != fmt.Sprintf("%d", gid) {
			return fmt.Sprintf("managed actor path %s is owned by gid %d instead of group %s", path, gid, expectedGroup)
		}
	}
	return ""
}

func fileInfoOwnershipIDs(info os.FileInfo) (uint32, uint32, bool) {
	if info == nil || info.Sys() == nil {
		return 0, 0, false
	}
	value := reflect.ValueOf(info.Sys())
	if !value.IsValid() {
		return 0, 0, false
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0, 0, false
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return 0, 0, false
	}
	uidField := value.FieldByName("Uid")
	gidField := value.FieldByName("Gid")
	if !uidField.IsValid() || !gidField.IsValid() {
		return 0, 0, false
	}
	uid, uidOK := numericFieldToUint32(uidField)
	gid, gidOK := numericFieldToUint32(gidField)
	if !uidOK || !gidOK {
		return 0, 0, false
	}
	return uid, gid, true
}

func numericFieldToUint32(field reflect.Value) (uint32, bool) {
	switch field.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uint32(field.Uint()), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value := field.Int()
		if value < 0 {
			return 0, false
		}
		return uint32(value), true
	default:
		return 0, false
	}
}

func linuxPathEntryPermissionIssue(path string, info os.FileInfo) string {
	path = strings.TrimSpace(path)
	if runtime.GOOS != "linux" || path == "" || info == nil {
		return ""
	}
	perms := info.Mode().Perm()
	if info.IsDir() {
		if perms&0022 != 0 {
			return fmt.Sprintf("managed actor directory %s has writable mode %04o beyond owner-only write", path, perms)
		}
		return ""
	}
	if perms&0022 != 0 {
		return fmt.Sprintf("managed actor file %s has writable mode %04o beyond owner-only write", path, perms)
	}
	return ""
}

func linuxPathRecursiveOwnershipIssue(rootPath, expectedUser, expectedGroup string) string {
	rootPath = strings.TrimSpace(rootPath)
	if runtime.GOOS != "linux" || rootPath == "" {
		return ""
	}
	issueCount := 0
	walkErr := filepath.Walk(rootPath, func(current string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if current == rootPath {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			issueCount++
			return fmt.Errorf("managed actor path %s contains symlink %s", rootPath, current)
		}
		if issue := linuxPathEntryOwnershipIssue(current, info, expectedUser, expectedGroup); issue != "" {
			issueCount++
			return fmt.Errorf("%s", issue)
		}
		if issue := linuxPathEntryPermissionIssue(current, info); issue != "" {
			issueCount++
			return fmt.Errorf("%s", issue)
		}
		if issueCount >= 1 {
			return filepath.SkipAll
		}
		return nil
	})
	if walkErr != nil {
		return walkErr.Error()
	}
	return ""
}

func managedActorAdoptionIssue(crate CrateState) string {
	if runtime.GOOS != "linux" || strings.TrimSpace(crate.ActorUser) == "" {
		return ""
	}
	if linuxGroupExists(crate.ActorGroup) && strings.TrimSpace(crate.ActorGroup) != "" {
		if passwd, ok := readLinuxPasswdEntry(crate.ActorGroup); ok && strings.TrimSpace(passwd.Name) != "" {
			return fmt.Sprintf("managed actor runtime group %s collides with an existing user account", crate.ActorGroup)
		}
	}
	if passwd, ok := readLinuxPasswdEntry(crate.ActorUser); ok {
		if expectedGroup := strings.TrimSpace(crate.ActorGroup); expectedGroup != "" && strings.TrimSpace(passwd.Group) != "" && !strings.EqualFold(strings.TrimSpace(passwd.Group), expectedGroup) {
			return fmt.Sprintf("managed actor runtime account %s is bound to existing group %s instead of %s", crate.ActorUser, passwd.Group, expectedGroup)
		}
		if expectedHome := strings.TrimSpace(crate.ActorHome); expectedHome != "" && strings.TrimSpace(passwd.Home) != "" && strings.TrimSpace(passwd.Home) != expectedHome {
			return fmt.Sprintf("managed actor runtime account %s is bound to existing home %s instead of %s", crate.ActorUser, passwd.Home, expectedHome)
		}
		if shell := strings.TrimSpace(passwd.Shell); shell != "" && shell != "/usr/sbin/nologin" && shell != "/usr/bin/nologin" && shell != "/bin/false" {
			return fmt.Sprintf("managed actor runtime account %s is bound to existing shell %s instead of a non-login shell", crate.ActorUser, shell)
		}
	}
	if groups := readLinuxSupplementaryGroups(crate.ActorUser); len(groups) > 0 {
		expectedGroup := strings.TrimSpace(crate.ActorGroup)
		for _, group := range groups {
			group = strings.TrimSpace(group)
			if group == "" {
				continue
			}
			if expectedGroup != "" && strings.EqualFold(group, expectedGroup) {
				continue
			}
			if strings.EqualFold(group, strings.TrimSpace(crate.ActorUser)) {
				continue
			}
			return fmt.Sprintf("managed actor runtime account %s has unexpected supplementary group %s", crate.ActorUser, group)
		}
	}
	for _, actorPath := range []string{crate.ActorHome, crate.ActorRuntimeDir, crate.ActorStateDir} {
		if issue := linuxPathOwnershipIssue(actorPath, crate.ActorUser, crate.ActorGroup); issue != "" {
			return issue
		}
		if issue := linuxPathRecursiveOwnershipIssue(actorPath, crate.ActorUser, crate.ActorGroup); issue != "" {
			return issue
		}
	}
	return ""
}

func ensureLinuxGroup(name string) error {
	name = strings.TrimSpace(name)
	if runtime.GOOS != "linux" || name == "" {
		return nil
	}
	if linuxGroupExists(name) {
		return nil
	}
	if !linuxCommandExists("groupadd") {
		return fmt.Errorf("groupadd not available while provisioning managed group %s", name)
	}
	return exec.Command("groupadd", "--system", name).Run()
}

func ensureLinuxAccount(user, group, home string) error {
	user = strings.TrimSpace(user)
	group = strings.TrimSpace(group)
	home = strings.TrimSpace(home)
	if runtime.GOOS != "linux" || user == "" {
		return nil
	}
	if linuxAccountExists(user) {
		return nil
	}
	if !linuxCommandExists("useradd") {
		return fmt.Errorf("useradd not available while provisioning managed actor %s", user)
	}
	args := []string{"--system", "--no-create-home"}
	if group != "" {
		args = append(args, "--gid", group)
	}
	if home != "" {
		args = append(args, "--home-dir", home)
	}
	args = append(args, "--shell", "/usr/sbin/nologin", user)
	return exec.Command("useradd", args...).Run()
}

func ensureOwnedDirectory(path string, perm os.FileMode, owner string) error {
	path = strings.TrimSpace(path)
	owner = strings.TrimSpace(owner)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(path, perm); err != nil {
		return err
	}
	if runtime.GOOS != "linux" || owner == "" {
		return nil
	}
	if !linuxCommandExists("chown") {
		return fmt.Errorf("chown not available while preparing %s", path)
	}
	return exec.Command("chown", "-R", owner+":"+owner, path).Run()
}

func prepareHostedManagedActor(desired config.ServiceEntry, crate CrateState) ([]Action, []string) {
	actions := []Action{}
	issues := []string{}
	if runtime.GOOS != "linux" || crate.ExecutionAdapter != "systemd" || crate.Module {
		return actions, issues
	}
	if issue := managedActorIdentityIssue(crate); issue != "" {
		issues = append(issues, issue)
		return actions, issues
	}
	if issue := managedActorAdoptionIssue(crate); issue != "" {
		issues = append(issues, issue)
		return actions, issues
	}
	if strings.TrimSpace(crate.ActorUser) == "" {
		issues = append(issues, fmt.Sprintf("managed actor provisioning requires actor identity for %s", desired.Name))
		return actions, issues
	}
	groupExists := linuxGroupExists(crate.ActorGroup)
	if err := ensureLinuxGroup(crate.ActorGroup); err != nil {
		issues = append(issues, fmt.Sprintf("provision managed group %s: %v", crate.ActorGroup, err))
	} else if crate.ActorGroup != "" && !groupExists {
		actions = append(actions, Action{
			Description: fmt.Sprintf("ensured managed actor group %s for %s", crate.ActorGroup, desired.Name),
			Component:   "service",
			Target:      crate.ActorGroup,
		})
	}
	accountExists := linuxAccountExists(crate.ActorUser)
	if err := ensureLinuxAccount(crate.ActorUser, crate.ActorGroup, crate.ActorHome); err != nil {
		issues = append(issues, fmt.Sprintf("provision managed actor %s: %v", crate.ActorUser, err))
	} else if !accountExists {
		actions = append(actions, Action{
			Description: fmt.Sprintf("ensured managed actor %s for %s", crate.ActorUser, desired.Name),
			Component:   "service",
			Target:      crate.ActorUser,
		})
	}
	for _, dir := range []struct {
		path string
		name string
	}{
		{path: crate.ActorHome, name: "actor home"},
		{path: crate.ActorRuntimeDir, name: "actor runtime"},
		{path: crate.ActorStateDir, name: "actor state"},
	} {
		if strings.TrimSpace(dir.path) == "" {
			continue
		}
		_, existed := os.Stat(dir.path)
		if err := ensureOwnedDirectory(dir.path, 0755, crate.ActorUser); err != nil {
			issues = append(issues, fmt.Sprintf("prepare %s for %s: %v", dir.name, desired.Name, err))
			continue
		}
		if os.IsNotExist(existed) {
			actions = append(actions, Action{
				Description: fmt.Sprintf("ensured %s dir %s for %s", dir.name, dir.path, desired.Name),
				Component:   "directory",
				Target:      dir.path,
			})
		}
	}
	return actions, issues
}
