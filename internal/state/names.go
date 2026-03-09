package state

import (
	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/modules"
)

// CollectServiceNames returns all service names to probe/ensure.
func CollectServiceNames(cfg *config.Config) []string {
	var names []string
	mods := modules.LoadAll(".")
	for _, s := range cfg.Services.Services {
		names = append(names, s.Name)
		if mod, ok := mods[s.Name]; ok {
			names = appendUnique(names, modules.ResolveUnits(s.Name, mod, true)...)
		}
	}
	// platform services
	names = appendUnique(names, "crateos-agent", "crateos-policy")
	return names
}
