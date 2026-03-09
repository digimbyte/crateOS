# PLATFORM_FILESYSTEM

Canonical CrateOS layout under `/srv/crateos` (created by agent postinst and enforced by agent):

```
/srv/crateos/
  config/          # human-edited YAML configs
  modules/         # module definitions / registry cache (future)
  services/        # per-service roots
  state/           # desired/actual state snapshots
    last-good/
    backups/
  logs/            # curated/exported logs
  export/          # symlink farm to OS internals
  registry/        # module registry cache
  runtime/         # sockets, pidfiles, temp (agent.sock lives here)
  cache/           # build/download cache
  backups/         # operator backups (tarballs)
  bin/             # platform binaries if needed
```

Per-service canonical layout (created on enable/install):
```
/srv/crateos/services/<service>/
  config/
  data/
  logs/
  runtime/
  backups/
  state.json
```

Permissions:
- Owned by root; readable by operators as needed.
- Agent socket: `/srv/crateos/runtime/agent.sock` (Unix socket, local-only).

Symlink/export examples (created by reconcile on Linux):
- `/srv/crateos/export/etc/NetworkManager` → `/etc/NetworkManager`
- `/srv/crateos/export/etc/nginx` → `/etc/nginx`
- `/srv/crateos/export/etc/nftables.conf` → `/etc/nftables.conf`
- `/srv/crateos/export/etc/ssh` → `/etc/ssh`
- `/srv/crateos/export/var/log/journal` → `/var/log/journal`

Markers:
- `/srv/crateos/state/installed.json` (written on install).
