# CRATE_CONFIG_SCHEMA

Defines the typed configuration CrateOS consumes under `/srv/crateos/config`.

## Files and top-level objects
- `crateos.yaml` → `crateos` (platform)
- `network.yaml` → `network`
- `firewall.yaml` → `firewall`
- `services.yaml` → `services`
- `users.yaml` → `users`
- `reverse-proxy.yaml` → `reverse_proxy`

## Field types
- `string`, `number`, `bool`
- `enum` (constrained string)
- `list` of strings
- `duration` (e.g., `30s`, `5m`)
- `size` (future)

## crateos.yaml
```yaml
version: "0.1.0"
platform:
  hostname: "crateos"
  timezone: "UTC"
  locale: "en_US.UTF-8"
access:
  ssh:
    enabled: true
    landing: "console"
  local_gui:
    enabled: false
    provider: "lightdm"
    landing: "workspace"
    default_shell: "crateos-session"
  virtual_desktop:
    enabled: false
    provider: ""
    landing: "workspace"
  break_glass:
    enabled: true
    require_permission: "shell.breakglass"
    allowed_surfaces: ["ssh"]
crate_root: "/srv/crateos"
log_level: "info"
updates:
  enabled: true
  channel: "stable"
  auto_apply: false
  check_interval: "6h"
maintenance:
  cleanup_enabled: true
  journal_vacuum_size: "500M"
  journal_vacuum_time: "7d"
  docker_prune: true
  cache_cleanup: true
```
Access/session rules:
- `ssh.enabled` should remain `true` for appliance access; `ssh.landing` must be `console`
- `local_gui.provider` is currently `lightdm` when `local_gui.enabled: true`
- `local_gui.landing` and `virtual_desktop.landing` must remain CrateOS-owned surfaces such as `console`, `panel`, `workspace`, or `recovery`
- `default_shell` for graphical sessions must remain `crateos-session`; CrateOS does not treat a normal distro desktop/session as the operator landing surface
- `break_glass.require_permission` should normally be `shell.breakglass`
- `break_glass.allowed_surfaces` lists which controlled entry surfaces may expose an explicit break-glass path

## network.yaml
```yaml
manager: "networkmanager"
profiles:
  - name: "LAN"
    type: "ethernet"
    mac: "00:11:22:33:44:55"
    method: "auto"      # auto | manual
    metric: 100
    autoconnect: true
  - name: "STATIC-LAN"
    type: "ethernet"
    mac: "00:11:22:33:44:55"
    method: "manual"
    metric: 150
    autoconnect: true
    static:
      address: "192.168.1.10/24"
      gateway: "192.168.1.1"
      dns: ["1.1.1.1", "8.8.8.8"]
  - name: "WIFI"
    type: "wifi"
    mac: "66:77:88:99:AA:BB"
    ssid: "CrateWiFi"
    password: "replace-me"
    method: "auto"
    metric: 600
    autoconnect: true
dns:
  fallback: ["1.1.1.1", "8.8.8.8"]
self_heal: true
```
Network profile rules:
- `manager` must be `networkmanager`
- `type` must be `ethernet` or `wifi`
- `method` must be `auto` or `manual`
- Wi-Fi profiles require both `ssid` and `password`
- Manual profiles require `static.address` in CIDR form and `static.gateway` as an IP
- `self_heal: true` allows CrateOS to remove stale CrateOS-managed native profiles that are no longer in desired state

## firewall.yaml
```yaml
enabled: true
backend: "nftables"
default_input: "drop"
default_forward: "drop"
default_output: "accept"
allow:
  - { name: "SSH", port: 22, protocol: "tcp", source: "any" }
  - { name: "Cockpit", port: 9090, protocol: "tcp", source: "lan" }
  - { name: "App", port: 3000, protocol: "tcp", source: "192.168.1.0/24" }
rate_limit:
  ssh:
    enabled: true
    max_attempts: 5
    window: "60s"
icmp:
  allow_ping: true
```
Supported firewall `source` values:
- `any`
- `lan`
- `vpn`
- `local`
- a single IP
- a CIDR range

## services.yaml
```yaml
services:
  - name: "nginx"
    enabled: true
    runtime: "systemd"
    autostart: true
    actor:
      name: "svc-nginx"
      type: "service"
    execution:
      mode: "service"
      timeout: "0"
      stop_timeout: "30s"
      on_timeout: "kill"
      kill_signal: "SIGTERM"
      concurrency: "replace"
    options: {}
  - name: "docs-api"
    enabled: true
    runtime: "systemd"
    autostart: true
    actor:
      name: "svc-docs-api"
      type: "bot"
    deploy:
      source: "upload"
      upload_path: "/srv/crateos/uploads/docs-api"
      working_dir: "/srv/crateos/services/docs-api/runtime/app"
      entry: "server.js"
      install_cmd: "npm ci --omit=dev"
      env_file: "/srv/crateos/services/docs-api/config/.env"
    execution:
      mode: "service"
      start_cmd: "node server.js"
      timeout: "0"
      stop_timeout: "20s"
      on_timeout: "kill"
      kill_signal: "SIGTERM"
      concurrency: "replace"
  - name: "nightly-sync"
    enabled: true
    runtime: "task"
    autostart: false
    actor:
      name: "job-nightly-sync"
      type: "bot"
    deploy:
      source: "upload"
      upload_path: "/srv/crateos/uploads/nightly-sync"
      working_dir: "/srv/crateos/services/nightly-sync/runtime/job"
      entry: "sync.js"
    execution:
      mode: "job"
      start_cmd: "node sync.js"
      schedule: "0 15 2 * * *"
      timeout: "10m"
      stop_timeout: "15s"
      on_timeout: "kill"
      kill_signal: "SIGTERM"
      concurrency: "forbid"
```
Service rules:
- `name` must match the crate/module ID
- `enabled` and `autostart` are operator intent
- `runtime` is only authoritative for non-module/manual crates
- for module-backed crates, runtime/install mode/units/packages come from `packaging/modules/*.module.yaml`
- `actor.name` is the CrateOS-managed execution identity for that service/job; it should not reuse a human operator account
- `actor.type` should distinguish long-lived service actors from bot/job actors
- `deploy.source` expresses how files arrive (`registry`, `upload`, `git`, future providers)
- `deploy.upload_path` is the intake location for operator uploads before CrateOS renders the runnable work tree
- `execution.mode` should be `service` for long-lived workloads or `job` for scheduled/one-shot workloads
- `execution.schedule` is required for recurring jobs and should map to a CrateOS-managed timer backend
- `execution.timeout` defines max runtime before timeout policy applies; `0` means no runtime deadline
- `execution.on_timeout` defines timeout behavior such as `kill` or future retry/fail modes
- `execution.concurrency` defines overlap policy such as `replace`, `forbid`, or future queue semantics

## users.yaml (roles + per-user overrides)
```yaml
roles:
  admin:
    description: "Full platform access"
    permissions: ["*"]
  staff:
    description: "Broad operators"
    permissions: ["sys.view", "sys.manage", "svc.*", "net.*", "proxy.*", "logs.view", "users.view", "shell.breakglass"]
users:
  - name: "crate"
    role: "admin"
    permissions: []      # optional per-user allow/deny; prefix with "-" to deny
    priority: 0          # optional (future tie-break)
```

Permission matching:
- Deny (`-perm`) overrides allow.
- Wildcards supported (`svc.*`).

## reverse-proxy.yaml
```yaml
enabled: true
defaults:
  listen_http: 80
  listen_https: 443
  ssl: false
  ssl_cert: "/etc/ssl/certs/crateos.crt"
  ssl_key: "/etc/ssl/private/crateos.key"
mappings:
  - name: "app"
    hostname: "app.local"
    target: "http://127.0.0.1:3000"
    path: "/"
    ssl: false
health_check:
  enabled: true
  interval: "30s"
  timeout: "5s"
validate_before_apply: true
```
