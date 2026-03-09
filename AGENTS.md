# AGENTS.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Existing guidance reviewed
- There was no pre-existing `AGENTS.md`, `WARP.md`, `CLAUDE.md`, Cursor rules, or Copilot instructions in this repository.
- `README.md`, `CONTRIBUTING.md`, packaging manifests, and the GitHub Actions workflow are the main sources of project guidance.

## Common commands
### Windows / PowerShell
- Build all binaries: `.\build.ps1 build`
- Clean build artifacts: `.\build.ps1 clean`
- Show build script help: `.\build.ps1 help`

### Linux / WSL
- Build all binaries: `make build`
- Build Debian packages: `make deb`
- Build autoinstall ISO: `make iso`
- Build qcow2 image: `make qcow2`
- Clean build artifacts: `make clean`

### Go commands used during development
- Format all Go code: `go fmt ./...`
- Run all tests: `go test ./...`
- Run a single package: `go test ./internal/state`
- Run a single test by name: `go test ./internal/state -run TestName`
- Build one binary directly: `go build ./cmd/crateos`

## Toolchain and build notes
- The repository is a Go module (`github.com/crateos/crateos`) targeting Go `1.24` with toolchain `go1.24.4`.
- The main build outputs are three binaries under `dist/bin`: `crateos`, `crateos-agent`, and `crateos-policy`.
- Packaging/image builds are Linux-oriented even though there is a Windows-friendly `build.ps1`; `deb`, `iso`, and `qcow2` are expected to run in WSL/Linux.
- The release workflow in `.github/workflows/release.yml` still sets `GO_VERSION: '1.22'`, which does not match `go.mod`. Treat CI/toolchain compatibility as something to verify before changing release automation.
- There are currently no `*_test.go` files in the repository. Use the `go test` commands above as the expected workflow when tests are added.

## High-level architecture
### Product model
- CrateOS is an appliance-style control plane layered on Ubuntu Server, centered around a single canonical root at `/srv/crateos`.
- The codebase is built around three executables:
  - `crateos`: operator-facing CLI and Bubble Tea TUI
  - `crateos-agent`: long-running reconciler plus local API server
  - `crateos-policy`: periodic drift-check binary intended to run from a systemd timer

### Codebase shape
- `cmd/` contains the three executable entrypoints only.
- `internal/` contains almost all runtime behavior:
  - `platform`: shared constants such as `/srv/crateos`, required directory names, and the agent socket path
  - `config`: loads desired state from YAML in `/srv/crateos/config`
  - `state`: probes actual machine state, reconciles desired vs actual state, applies remediations, and writes state snapshots
  - `api`: exposes the agent’s local Unix-socket HTTP API and client used by both the CLI and TUI
  - `tui`: Bubble Tea application for status, services, users, logs, and network views
  - `auth`: resolves role/user permissions from `users.yaml`
  - `modules`: loads module metadata from `packaging/modules/*.module.yaml`
  - `logs`: exports curated journald logs into `/srv/crateos/logs`
  - `sysinfo`: cross-platform machine and network inspection helpers
- `packaging/` holds the shipped defaults and Debian package assets.
- `images/` holds Linux image-build scripts for ISO and qcow2 outputs.

### Runtime control flow
1. `crateos-agent` boots, ensures `/srv/crateos` exists, seeds default config files from `/usr/share/crateos/defaults/config`, writes `state/installed.json`, and starts the local API server.
2. On startup and every 30 seconds, the agent runs `state.Reconcile()`.
3. Reconciliation loads desired config from `/srv/crateos/config/*.yaml`, probes live system state, compares the two, applies mutations, then writes JSON snapshots such as `state/desired.json`, `state/actual.json`, per-service `state.json`, and `crate-state.json`.
4. The agent also exports curated journald logs into `/srv/crateos/logs`.
5. `crateos-policy` is separate from the agent; it runs point-in-time checks for required directories and the installed marker, and is scheduled by `crateos-policy.timer`.
6. Platform-level configs are also rendered into canonical managed artifacts under `state/rendered` and previous versions are copied into `state/last-good` before overwrite.

### Desired state model
- Desired state is primarily YAML-driven and lives under `/srv/crateos/config`.
- `config.Load()` treats these files as a single config bundle:
  - `crateos.yaml`
  - `network.yaml`
  - `firewall.yaml`
  - `services.yaml`
  - `users.yaml`
  - `reverse-proxy.yaml`
- `services.yaml` is the registry of managed crates/services. User actions in the CLI/TUI/API usually mutate this file or `users.yaml`, then refresh derived crate state.
- Permission naming is canonicalized around short namespaces such as `sys.*`, `users.*`, `net.*`, `proxy.*`, `modules.*`, and `svc.*`. Keep new permission work inside that vocabulary.

### Agent API and UI relationship
- The TUI and CLI should be thought of as clients of the local agent, not as the primary place where system mutations happen.
- `internal/api/client.go` talks to the agent over the Unix socket at `/srv/crateos/runtime/agent.sock`.
- The TUI prefers API-backed data and falls back to local probing/config reads when the socket is unavailable.
- Service and user operations from the UI mutate YAML config through API handlers, then optionally run `systemctl` actions and refresh crate-state snapshots.

### Service/module model
- Reconciliation always manages `crateos-agent` and `crateos-policy`, then extends that with services declared in `services.yaml`.
- Managed services also get canonical per-service directories under `/srv/crateos/services/<name>/` (`config`, `data`, `logs`, `runtime`, `backups`).
- Module metadata in `packaging/modules` augments a service with package dependencies, unit names, canonical/native paths, and command-based health checks.
- `installMode: staged` matters: enabling a staged module can leave it intentionally enabled-but-not-started until an explicit start action. PostgreSQL is the main example of that pattern.

### Module manifest contract
- Module definitions in `packaging/modules/*.module.yaml` use a single canonical schema:
  - `apiVersion`
  - `kind`
  - `metadata`
  - `spec`
- `internal/modules/modules.go` only reads that schema. When adding or editing modules, keep them aligned with the existing `metadata/spec` shape rather than introducing alternate formats.

### Packaging and install behavior
- Debian package assets under `packaging/deb` are not just distribution artifacts; they are part of the runtime contract.
- `crateos-agent` package install creates `/srv/crateos/...`, writes the installed marker, and enables/starts `crateos-agent.service`.
- `crateos-policy` package install enables and starts `crateos-policy.timer`.
- The `crateos` package ships an SSH `ForceCommand` config so SSH sessions land in `crateos console` rather than a raw shell.
- Default YAML config shipped by the agent package is copied from `packaging/config` into `/usr/share/crateos/defaults/config`, then seeded into `/srv/crateos/config` on first run.

### State and export conventions
- `/srv/crateos` is the canonical source of truth for platform-owned config, state, logs, and exported OS views.
- Reconcile-managed platform artifacts currently render to `state/rendered`:
  - reverse proxy summary at `state/rendered/reverse-proxy.json`
  - firewall rules at `state/rendered/firewall.nft`
  - network profiles/state under `state/rendered/network*`
  - nginx reverse-proxy config at `services/nginx/config/crateos-generated.conf`
- On Linux, the agent also attempts native apply for those rendered artifacts:
  - nginx config synced to `/etc/nginx/conf.d/crateos-generated.conf` with `nginx -t`/reload when configured
  - firewall config synced to `/etc/nftables.conf` and applied with `nft -f`
  - network profiles synced to `/etc/NetworkManager/system-connections/*.nmconnection` with `nmcli connection reload`
- Before those managed artifacts are replaced, the previous versions are snapshotted under `state/last-good/<logical-path>`.
- Reconcile also writes operator-facing adapter status to `state/platform-state.json`; status surfaces should consume that stable snapshot instead of reconstructing platform state from logs.
- The agent creates an export/symlink farm under `/srv/crateos/export` for selected native OS paths such as NetworkManager, nginx, nftables, SSH, and journald logs.
- When changing behavior, preserve the idea that native OS locations are implementation details surfaced through canonical CrateOS paths.
