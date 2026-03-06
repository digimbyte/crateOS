package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/sysinfo"
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
