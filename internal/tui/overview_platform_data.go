package tui

import "github.com/crateos/crateos/internal/state"

func readFallbackPlatformState() PlatformInfo {
	snapshot := state.LoadPlatformState()
	info := PlatformInfo{
		GeneratedAt: snapshot.GeneratedAt,
		Adapters:    make([]PlatformAdapter, 0, len(snapshot.Adapters)),
	}
	for _, adapter := range snapshot.Adapters {
		info.Adapters = append(info.Adapters, PlatformAdapter{
			Name:          adapter.Name,
			DisplayName:   adapter.DisplayName,
			Enabled:       adapter.Enabled,
			Status:        adapter.Status,
			Health:        adapter.Health,
			Summary:       adapter.Summary,
			LastError:     adapter.LastError,
			Validation:    adapter.Validation,
			ValidationErr: adapter.ValidationErr,
			Apply:         adapter.Apply,
			ApplyErr:      adapter.ApplyErr,
			RenderedPaths: append([]string(nil), adapter.RenderedPaths...),
			NativeTargets: append([]string(nil), adapter.NativeTargets...),
		})
	}
	return info
}
