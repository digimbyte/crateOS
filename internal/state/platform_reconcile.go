package state

import (
	"fmt"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/virtualization"
)

func reconcilePlatform(cfg *config.Config) []Action {
	var actions []Action
	accessActions, accessState := reconcileAccess(cfg.CrateOS)
	userActions, userState := reconcileUsers(cfg)
	virtDesktopState := virtualization.ReconcileVirtualDesktop(cfg)
	reverseProxyActions, reverseProxyState := reconcileReverseProxy(cfg.ReverseProxy)
	firewallActions, firewallState := reconcileFirewall(cfg.Firewall)
	networkActions, networkState := reconcileNetwork(cfg.Network)
	storageActions, storageState := reconcileStorage()
	actions = append(actions, accessActions...)
	actions = append(actions, userActions...)
	actions = append(actions, reverseProxyActions...)
	actions = append(actions, firewallActions...)
	actions = append(actions, networkActions...)
	actions = append(actions, storageActions...)
	if len(virtDesktopState.Issues) > 0 {
		for _, issue := range virtDesktopState.Issues {
			actions = append(actions, Action{
				Description: fmt.Sprintf("virtual desktop: %s", issue),
				Component:   "virtualization",
				Target:      "sessions",
			})
		}
	}
	writePlatformState(PlatformState{
		Adapters: []PlatformAdapterState{accessState, userState, reverseProxyState, firewallState, networkState, storageState},
	})
	return actions
}
