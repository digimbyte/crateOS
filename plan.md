# Problem statement
Define the remaining work to deliver a complete cPanel-style, multi-tenant OS experience for CrateOS, from login to backups and releases.
# Current state (high level)
CrateOS has a defined canonical root under `/srv/crateos`, an autoinstall seed with SSH enabled and a forced TUI entry, baseline configs under `packaging/config/`, a multi-tenant permissions schema documented in `GUIDE.md`, and package policy split into required vs optional in `packaging/config/packages.yaml`.
# Proposed changes
## 1) Identity, access, and login UX
* Implement user lifecycle (create/disable/delete, password reset) and session management.
* Enforce RBAC consistently across TUI, agent API, and web panel.
* Add MFA and break-glass audit logging with explicit permission checks.
## 2) Control plane core (TUI + API + agent)
* Define a stable local API contract (Unix socket or local HTTPS) that the TUI and web panel consume.
* Implement idempotent apply with drift detection and last-good snapshots.
* Provide a safe “apply preview” before changes are committed.
## 3) Services/modules system
* Formalize module lifecycle (install/enable/disable/uninstall) with per-service permissions.
* Add module metadata schema validation and compatibility checks.
* Implement plugin management hooks per service (add/remove/configure).
## 4) Filesystem/export model
* Complete symlink farm under `/srv/crateos/export/**` for known OS paths.
* Curate logs into `/srv/crateos/logs/**` with a stable log naming policy.
## 5) Networking, firewall, reverse proxy
* Enforce NetworkManager profile repair and route priority.
* Implement nftables generation from `firewall.yaml` with safe apply.
* Implement reverse proxy mappings from `reverse-proxy.yaml` and add ACME/TLS automation.
## 6) Backups & restore
* Define backup targets (system config + per-service data).
* Implement backup policies and restore workflows per service.
* Add export/download of backup archives.
## 7) Updates & releases
* Use GitHub Releases as source of truth with checksum/signing.
* Define update apply policy (manual/maintenance window).
* Add versioned config schema migrations for future changes.
## 8) Observability & audit
* Implement health dashboard (CPU/RAM/disk/network/service health).
* Centralize audit log for privileged actions.
* Add support bundle export (logs + configs, redacted).
## 9) Installer & image pipeline
* Finish ISO repack step and qcow2 cloud-init seed.
* Ensure packages list in autoinstall stays in sync with `packages.yaml`.
## MVP slice (recommended)
* Custom SSH → TUI login, user/role management, service enable/disable, logs view, and backups v1.
* Freeze a minimal module set: nginx + docker + postgres as opt-in.
* Deliver ISO and qcow2 images from GitHub Releases with checksums.
