package state

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/crateos/crateos/internal/config"
	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/users"
)

func reconcileUsers(cfg *config.Config) ([]Action, PlatformAdapterState) {
	var actions []Action
	var issues []string
	adapter := platformAdapterState("users", "User Provisioning", true)

	if runtime.GOOS != "linux" {
		adapter.Summary = "user provisioning not supported on non-Linux platforms"
		return actions, finalizePlatformAdapterState(adapter, issues)
	}

	reconciled, provState, err := users.ProvisionUsers(cfg)
	if err != nil {
		issues = append(issues, fmt.Sprintf("user provisioning failed: %v", err))
		adapter.Validation = "failed"
		adapter.ValidationErr = err.Error()
	} else {
		adapter.Validation = "ok"
	}

	for _, issue := range provState.Issues {
		issues = append(issues, issue)
	}

	renderedPath := platform.CratePath("state", "rendered", "user-provisioning.json")
	adapter.RenderedPaths = append(adapter.RenderedPaths, renderedPath)
	if data, marshalErr := json.MarshalIndent(provState, "", "  "); marshalErr == nil {
		if action, writeErr := writeManagedArtifact(
			"users/provisioning.json",
			renderedPath,
			string(data)+"\n",
			"users",
			"rendered user provisioning state",
		); writeErr != nil {
			issues = append(issues, writeErr.Error())
		} else if action != nil {
			actions = append(actions, *action)
		}
	} else {
		issues = append(issues, fmt.Sprintf("failed to marshal provisioning state: %v", marshalErr))
	}

	adapter.Summary = provState.Summary
	if len(reconciled) > 0 {
		adapter.Apply = "ok"
	}

	return actions, finalizePlatformAdapterState(adapter, issues)
}
