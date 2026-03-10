package state

import (
	"fmt"
	"runtime"

	"github.com/crateos/crateos/internal/platform"
	"github.com/crateos/crateos/internal/sysinfo"
)

func reconcileStorage() ([]Action, PlatformAdapterState) {
	adapter := platformAdapterState("storage", "Storage", true)
	adapter.RenderedPaths = append(adapter.RenderedPaths, platform.CratePath("state", "storage-state.json"))
	devices := sysinfo.StorageDevices()
	storageState := normalizeStorageDevices(devices)
	writeStorageState(storageState)
	adapter.Validation = "ok"
	adapter.Apply = "ok"
	switch {
	case runtime.GOOS != "linux":
		adapter.Status = "unknown"
		adapter.Health = "unknown"
		adapter.Summary = "storage posture only probes linux hosts"
	case len(storageState.Devices) == 0:
		adapter.Status = "failed"
		adapter.Health = "degraded"
		adapter.Validation = "failed"
		adapter.Apply = "skipped"
		adapter.LastError = "no storage devices detected from lsblk"
		adapter.ValidationErr = adapter.LastError
		adapter.Summary = adapter.LastError
	case len(storageState.SafeTargets) == 0:
		adapter.Status = "ready"
		adapter.Health = "degraded"
		adapter.Summary = fmt.Sprintf("detected %d storage devices; no dedicated safe data target mounted yet", len(storageState.Devices))
		adapter.LastError = "stateful crates still share the system disk unless a target is mounted under /srv, /mnt, or /media"
	default:
		adapter.Status = "ready"
		adapter.Health = "ok"
		adapter.Summary = fmt.Sprintf("detected %d storage devices; %d safe data target(s) available", len(storageState.Devices), len(storageState.SafeTargets))
	}
	return nil, adapter
}
