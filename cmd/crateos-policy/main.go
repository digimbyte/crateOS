package main

import (
	"fmt"
	"log"
	"os"

	"github.com/crateos/crateos/internal/platform"
)

// checks is the list of policy assertions to verify.
var checks = []struct {
	name string
	fn   func() error
}{
	{"crate root exists", checkCrateRoot},
	{"installed marker present", checkInstalledMarker},
	{"required dirs present", checkRequiredDirs},
}

func main() {
	log.SetPrefix("crateos-policy: ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)

	log.Printf("policy check v%s", platform.Version)

	failed := 0
	for _, c := range checks {
		if err := c.fn(); err != nil {
			log.Printf("FAIL  %s: %v", c.name, err)
			failed++
		} else {
			log.Printf("OK    %s", c.name)
		}
	}

	fmt.Println()
	if failed > 0 {
		fmt.Printf("policy check: %d/%d checks failed\n", failed, len(checks))
		os.Exit(1)
	}
	fmt.Printf("policy check: all %d checks passed\n", len(checks))
}

func checkCrateRoot() error {
	info, err := os.Stat(platform.CrateRoot)
	if err != nil {
		return fmt.Errorf("not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", platform.CrateRoot)
	}
	return nil
}

func checkInstalledMarker() error {
	p := platform.CratePath("state", "installed.json")
	if _, err := os.Stat(p); err != nil {
		return fmt.Errorf("missing: %w", err)
	}
	return nil
}

func checkRequiredDirs() error {
	for _, d := range platform.RequiredDirs {
		p := platform.CratePath(d)
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("%s missing: %w", d, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", d)
		}
	}
	return nil
}
