package main

import (
	"fmt"
	"github.com/crateos/crateos/internal/api"
	"github.com/crateos/crateos/internal/config"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/sysinfo"
	"github.com/crateos/crateos/internal/takeover"
	"github.com/crateos/crateos/internal/tui"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "console":
			if err := tui.Run(); err != nil {
				log.Fatalf("console error: %v", err)
			}
		case "version":
			printVersion()
		case "status":
			printStatus()
		case "bootstrap":
			handleBootstrapCmd(os.Args[2:])
		case "repair-install-contract":
			handleRepairInstallContractCmd(os.Args[2:])
		case "service":
			handleServiceCmd(os.Args[2:])
		case "user":
			handleUserCmd(os.Args[2:])
		case "backup":
			handleBackupCmd(os.Args[2:])
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
		return
	}
	printUsage()
}

func printUsage() {
	fmt.Println("CrateOS — appliance-style server platform")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  crateos console   Enter the CrateOS interactive console")
	fmt.Println("  crateos status    Show system status summary")
	fmt.Println("  crateos version   Print version information")
	fmt.Println("  crateos bootstrap <name>                Create the first admin for primer recovery when users are missing")
	fmt.Println("  crateos repair-install-contract [name]  Repair the local console/install contract for the named operator")
	fmt.Println("  crateos service enable|disable|start|stop <name>   Manage services")
	fmt.Println("  crateos user add <name> <role>          Add a user with role")
	fmt.Println("  crateos user del <name>                 Delete a user")
	fmt.Println("  crateos user rename <old> <new>         Rename a user")
	fmt.Println("  crateos user set-role <name> <role>     Set a user's role")
	fmt.Println("  crateos user set-perms <name> <p1,p2>   Replace a user's permissions list")
	fmt.Println("  crateos backup [out.tar.gz]             Backup config + services state/data")
}

func printVersion() {
	fmt.Printf("CrateOS %s (%s/%s)\n", platform.Version, runtime.GOOS, runtime.GOARCH)
}

func printStatus() {
	info := sysinfo.Gather()
	fmt.Println("=== CrateOS Status ===")
	fmt.Printf("  Version:   %s\n", platform.Version)
	fmt.Printf("  Hostname:  %s\n", info.Hostname)
	fmt.Printf("  Platform:  %s/%s\n", info.OS, info.Arch)
	fmt.Printf("  Time:      %s\n", time.Now().Format(time.RFC3339))
	fmt.Printf("  CPUs:      %d\n", info.CPUs)
	fmt.Println()
	fmt.Printf("  Crate Root: %s\n", platform.CrateRoot)

	marker := platform.CratePath("state", "installed.json")
	if _, err := os.Stat(marker); err == nil {
		fmt.Println("  Installed:  yes")
	} else {
		fmt.Println("  Installed:  no")
	}
}

func resolveCLIUser() string {
	cfg, err := config.Load()
	if err != nil {
		return ""
	}
	if len(cfg.Users.Users) == 0 {
		return ""
	}
	return strings.TrimSpace(cfg.Users.Users[0].Name)
}

func requireCLIUser() string {
	user := resolveCLIUser()
	if user == "" {
		fmt.Fprintln(os.Stderr, "no configured operator found; complete the CrateOS primer or run: crateos bootstrap <name>")
		os.Exit(1)
	}
	return user
}

func handleBootstrapCmd(args []string) {
	if len(args) < 1 || strings.TrimSpace(args[0]) == "" {
		fmt.Println("usage: crateos bootstrap <name>")
		os.Exit(1)
	}
	name := strings.TrimSpace(args[0])
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap failed: %v\n", err)
		os.Exit(1)
	}
	if len(cfg.Users.Users) > 0 {
		fmt.Fprintln(os.Stderr, "bootstrap failed: users already configured")
		os.Exit(1)
	}
	if cfg.Users.Roles == nil {
		cfg.Users.Roles = map[string]config.Role{}
	}
	if _, ok := cfg.Users.Roles["admin"]; !ok {
		cfg.Users.Roles["admin"] = config.Role{
			Description: "Full platform access including break-glass shell",
			Permissions: []string{"*"},
		}
	}
	cfg.Users.Users = append(cfg.Users.Users, config.UserEntry{
		Name: name,
		Role: "admin",
	})
	if err := config.SaveUsers(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("bootstrap complete")
}

func handleRepairInstallContractCmd(args []string) {
	username := ""
	if len(args) > 0 {
		username = strings.TrimSpace(args[0])
	}
	if username == "" {
		username = resolveCLIUser()
	}
	if err := takeover.RepairLocalInstallContract(username); err != nil {
		fmt.Fprintf(os.Stderr, "repair-install-contract failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("repair-install-contract complete")
}
func handleBackupCmd(args []string) {
	if runtime.GOOS != "linux" {
		fmt.Println("backup only supported on Linux host")
		os.Exit(1)
	}
	out := "crateos-backup.tar.gz"
	if len(args) >= 1 {
		out = args[0]
	}
	cmd := exec.Command("tar", "-czf", out,
		platform.CratePath("config"),
		platform.CratePath("services"),
		platform.CratePath("state"),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "backup failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("backup written to %s\n", out)
}

func handleServiceCmd(args []string) {
	if len(args) < 2 {
		fmt.Println("usage: crateos service enable|disable|start|stop <name>")
		os.Exit(1)
	}
	action, name := args[0], args[1]
	client := api.NewClient(requireCLIUser())
	var err error
	switch action {
	case "enable":
		err = client.EnableService(name)
	case "disable":
		err = client.DisableService(name)
	case "start":
		err = client.StartService(name)
	case "stop":
		err = client.StopService(name)
	default:
		fmt.Println("usage: crateos service enable|disable|start|stop <name>")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "service %s failed: %v\n", action, err)
		os.Exit(1)
	}
	fmt.Printf("service %s %s\n", name, action)
}

func handleUserCmd(args []string) {
	if len(args) < 2 {
		fmt.Println("usage: crateos user add <name> <role> | crateos user del <name> | crateos user rename <old> <new> | crateos user set-role <name> <role> | crateos user set-perms <name> <p1,p2>")
		os.Exit(1)
	}
	client := api.NewClient(requireCLIUser())
	switch args[0] {
	case "add":
		if len(args) < 3 {
			fmt.Println("usage: crateos user add <name> <role>")
			os.Exit(1)
		}
		var perms []string
		if len(args) >= 4 && strings.TrimSpace(args[3]) != "" {
			perms = strings.Split(args[3], ",")
		}
		err := client.AddUser(args[1], args[2], perms)
		if err != nil {
			fmt.Fprintf(os.Stderr, "user add failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("user added")
	case "del":
		err := client.DeleteUser(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "user delete failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("user deleted")
	case "rename":
		if len(args) < 3 {
			fmt.Println("usage: crateos user rename <old> <new>")
			os.Exit(1)
		}
		err := client.UpdateUser(args[1], args[2], "", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "rename failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("user renamed")
	case "set-role":
		if len(args) < 3 {
			fmt.Println("usage: crateos user set-role <name> <role>")
			os.Exit(1)
		}
		err := client.UpdateUser(args[1], "", args[2], nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "set-role failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("user role updated")
	case "set-perms":
		if len(args) < 3 {
			fmt.Println("usage: crateos user set-perms <name> <p1,p2>")
			os.Exit(1)
		}
		perms := strings.Split(args[2], ",")
		err := client.UpdateUser(args[1], "", "", perms)
		if err != nil {
			fmt.Fprintf(os.Stderr, "set-perms failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("user perms updated")
	default:
		fmt.Println("usage: crateos user add <name> <role> | crateos user del <name> | crateos user rename <old> <new> | crateos user set-role <name> <role> | crateos user set-perms <name> <p1,p2>")
		os.Exit(1)
	}
}
