# CrateOS Module Registry

This document defines the **CrateOS module registry model**.

The module registry is how CrateOS discovers, versions, installs, updates, verifies, and manages available crates.

The registry must support both:

* a clean **panel-first / TUI-first user experience**
* an open and extensible ecosystem for module authors

CrateOS modules are not random scripts. They are **versioned managed definitions** that must conform to the CrateOS module specification and state machine.

---

## Core principle

A module registry is not just a download location.
It is the **source of truth** for available crates, their metadata, versions, compatibility, trust level, and installation pipeline.

Users should be able to think in terms of:

* browse modules
* install modules
* update modules
* remove modules

Users should not need to care whether the module came from:

* a local file
* a Git repository
* an HTTP registry
* a bundled offline catalog

The source is an implementation detail. The registry model stays the same.

---

## Design goals

* **Single user-facing model**: all install sources behave the same in the panel/TUI.
* **Versioned modules**: modules are identified and managed by semantic versions.
* **Compatibility-aware**: modules must declare which CrateOS versions they support.
* **Trust-aware**: the registry should distinguish official, community, local, and untrusted crates.
* **Offline-capable**: CrateOS must support bundled/local module catalogs.
* **Extensible**: future support for signatures, enterprise registries, and curated channels.

---

## Registry responsibilities

A CrateOS module registry should answer these questions:

* What crates are available?
* What versions of each crate exist?
* Which versions are compatible with this CrateOS build?
* Which crates are installed locally?
* Which installed crates have updates available?
* Which crates are trusted, verified, local-only, or unsupported?
* How should a selected crate be installed?

---

## Registry layers

CrateOS should treat the registry as having three layers:

### 1. Catalog layer

The list of available crates and their metadata.

### 2. Artifact layer

The actual module payloads/manifests/templates/hooks.

### 3. Local registry state

The locally known view of installed and cached crates.

---

## Registry source types

CrateOS should support multiple registry source types behind one model.

### Official registry

First-party CrateOS module catalog.

Examples:

* official nginx module
* official postgres module
* official redis module

### Community registry

Third-party public catalog of community crates.

### Private/enterprise registry

Self-hosted or organization-managed catalog.

### Local registry

Modules stored on disk, useful for:

* offline installs
* local development
* bundled images

### Git source (future/optional)

A module can be sourced from a Git repository, but this should still resolve into a normal local module artifact and metadata record.

---

## Registry trust levels

CrateOS should expose trust levels clearly in the UI.

Suggested trust levels:

* **official** — maintained by the CrateOS project
* **verified** — tested or curated by a trusted party
* **community** — public, not verified
* **local** — locally installed/offline module
* **untrusted** — imported without verification

These trust levels are descriptive. They should influence warnings, not necessarily block installation by default.

---

## Module identity

A module should be uniquely identified by:

* `id`
* `version`
* `source`

Example identity:

```text
postgres@16.0.0 (official)
```

### Required identity fields

* `id`: stable module identifier, e.g. `postgres`
* `version`: semantic version of the module definition itself
* `source`: registry source or local path
* `channel`: optional release channel such as `stable`, `beta`, `edge`

Important:

* Module version is not always the same as the software version.
* A `postgres` module may manage PostgreSQL 16, while the module definition itself may be version `1.2.0`.

---

## Module record shape

Example catalog entry:

```yaml
id: postgres
name: PostgreSQL
category: database
summary: Managed PostgreSQL database service
moduleVersion: 1.2.0
softwareVersion: "16"
source: official
channel: stable
trust: official
compatibility:
  crateos:
    min: 0.1.0
    max: 0.x
capabilities:
  - database
  - stateful
  - networked
artifacts:
  manifest: postgres/module.yaml
  templates: postgres/templates/
  hooks: postgres/hooks/
checksum: sha256:abcd1234
```

---

## Compatibility model

Every module must declare CrateOS compatibility.

Example:

```yaml
compatibility:
  crateos:
    min: 0.1.0
    max: 0.x
```

Optional future fields:

* supported architectures (`amd64`, `arm64`)
* supported runtimes (`systemd`, `container`)
* required platform capabilities (`docker`, `nginx`, `wireguard`, etc.)

The registry must prevent or warn on incompatible installs.

---

## Channels

Modules may be published to channels.

Suggested channels:

* `stable`
* `beta`
* `edge`
* `dev`

UI behavior:

* default users see `stable`
* advanced users may opt into other channels
* installed crates should record the channel they came from

---

## Local registry structure

CrateOS should maintain local registry data under a predictable root.

Suggested layout:

```text
/srv/crateos/registry/
  sources.yaml           # configured registry sources
  catalog/               # cached catalog data
  artifacts/             # downloaded module artifacts
  installed/             # installed module records
  trust/                 # trust and signature metadata
  cache/                 # temporary cache
```

Suggested substructure:

```text
/srv/crateos/registry/catalog/official.json
/srv/crateos/registry/catalog/community.json
/srv/crateos/registry/artifacts/postgres/1.2.0/
/srv/crateos/registry/installed/postgres.json
```

---

## Installed module record

Example:

```json
{
  "id": "postgres",
  "moduleVersion": "1.2.0",
  "softwareVersion": "16",
  "source": "official",
  "channel": "stable",
  "trust": "official",
  "installedAt": "2026-03-07T12:00:00Z",
  "state": "Running"
}
```

---

## Registry source configuration

CrateOS should support multiple configured sources.

Example `sources.yaml`:

```yaml
sources:
  - id: official
    type: http
    url: https://registry.crateos.example/official
    enabled: true
    trust: official

  - id: community
    type: http
    url: https://registry.crateos.example/community
    enabled: true
    trust: community

  - id: local-bundle
    type: file
    path: /srv/crateos/bundles/modules
    enabled: true
    trust: local
```

Supported source types (initial):

* `http`
* `file`
* `bundled`

Future:

* `git`
* `oci`
* `s3`

---

## Installation model

The panel/TUI should expose module installation as a simple action:

* browse
* install
* configure
* enable

Internally, the registry flow may be:

1. resolve selected module and version
2. check compatibility
3. check trust/warnings
4. fetch artifact if needed
5. cache artifact locally
6. register crate in installed state
7. hand off to crate install pipeline

This keeps the registry model separate from the runtime lifecycle.

---

## Update model

The registry should support update discovery.

Example logic:

* compare installed module version against source catalog
* filter by channel
* filter by compatibility
* show available update in panel/TUI

Update types may include:

* patch update
* minor update
* major update
* channel change

CrateOS should clearly distinguish:

* **module definition update**
* **underlying software version update**

Example:

* `postgres module 1.2.0 -> 1.3.0`
* PostgreSQL runtime remains `16`

Or:

* `postgres module 1.3.0` now supports PostgreSQL `17`

---

## Bundled/offline modules

CrateOS should support offline images and preloaded module bundles.

This is important for:

* ISO builds
* local appliances
* air-gapped environments
* Raspberry Pi installs

A bundled module should behave exactly like a registry-sourced module once imported.

The only difference is source metadata, e.g.:

```text
source: bundled
trust: local
```

---

## Module artifact model

A module artifact should contain everything needed to register and install the crate definition.

Suggested contents:

```text
postgres/
  module.yaml
  templates/
  hooks/
  assets/
  README.md
```

Optional packaging formats:

* unpacked directory
* tar.gz bundle
* signed archive (future)

The registry should normalize all of these into a cached local artifact directory.

---

## Signature and verification model (future-friendly)

CrateOS should be designed with future verification in mind, even if initial versions do not fully enforce it.

Possible verification fields:

* checksum
* signed manifest
* trusted registry key
* verified publisher

UI behavior should be able to say:

* verified
* unsigned
* checksum mismatch
* unknown publisher

Even if verification starts as warnings only, the model should exist early.

---

## Search and discovery

The registry should support filtering by:

* name
* category
* capability
* trust level
* source
* channel
* installed/not installed
* update available

This is important because the user experience is panel-first, not package-first.

A user should be able to browse:

* databases
* web apps
* proxies
* storage
* utilities

without caring about package names.

---

## Recommended user-facing install flow

Example:

1. User browses modules
2. User selects `PostgreSQL`
3. UI shows:

   * description
   * trust level
   * source
   * module version
   * software version
   * compatibility
   * required capabilities
4. User clicks **Install**
5. CrateOS resolves and caches artifact
6. Crate enters `Installing`
7. Crate enters `Installed`
8. User proceeds to **Configure**

The source mechanism stays hidden unless the user explicitly asks.

---

## Registry authoring guidance

Module authors should publish crates that:

* conform to `MODULE_SPEC.md`
* declare compatibility correctly
* version module definitions semantically
* include clear metadata
* avoid source-specific hacks

A good registry entry should feel like a product listing, not a package dump.

---

## Non-goals

The registry should **not** become:

* a generic package manager replacement
* a random script host
* a dependency-free-for-all
* a UX leak of apt/dpkg internals
* a trust-agnostic dump of arbitrary code

CrateOS may use native OS package systems underneath, but the registry is about **managed crate definitions**, not raw packages.

---

## Initial implementation recommendation

For the first implementation, keep it simple.

### Support initially

* local file registry
* bundled registry
* one official HTTP registry
* semantic module versions
* trust labels
* compatibility checks
* local cache under `/srv/crateos/registry/`

### Defer initially

* signatures enforcement
* Git-based sources
* enterprise auth
* OCI distribution
* complex dependency graphs

The first version should prove the model, not solve every future distribution problem.

---

## Summary

The CrateOS module registry is the discovery and delivery system for crates.

It provides:

* a consistent source-of-truth for modules
* versioning and compatibility awareness
* trust labels and future verification hooks
* a clean panel-first install/update/remove experience
* support for local, bundled, official, and community sources

This is what allows CrateOS to grow an ecosystem without turning into a random pile of scripts and packages.
