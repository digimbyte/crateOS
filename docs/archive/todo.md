### 1. Add `LICENSE` — **DONE**

Apache-2.0 added at repo root; README updated.

### 2. Clean repo metadata — **PARTIAL**

Done:
* `.gitignore`
* `CONTRIBUTING.md`
* `CODEOWNERS`
* issue templates + PR template

Remaining:
* branch rename to **main** (optional)

### 3. Freeze the MVP contract — PENDING

Create one tiny milestone and do not drift:

* boot image
* `/srv/crateos` exists
* `crateos-agent` runs
* SSH lands in Crate console
* one crate can be installed/configured/enabled

That is the first real product checkpoint.

### 4. Build the minimum binaries — DONE

Get these to compile cleanly:

* `crateos`
* `crateos-agent`
* `crateos-policy`

Even if they do almost nothing yet, they must:

* build reliably
* install cleanly
* start cleanly

### 5. Create the canonical filesystem — DONE

Make the installer or package create:

```text
/srv/crateos/
  config/
  services/
  state/
  logs/
  registry/
  export/
  runtime/
  cache/
  backups/
```

That should happen before complex runtime logic.

### 6. Create systemd units — DONE

Ship:

* `crateos-agent.service`
* `crateos-policy.service`
* `crateos-policy.timer`

And prove they:

* install
* enable
* start
* log somewhere predictable

### 7. Lock the SSH entry model — DONE

Implement the first real UX rule:

* SSH goes to Crate console
* shell is not default
* break-glass later

This is one of the core differentiators, so do it early.

### 8. Make `make build` and `build.ps1` solid — PARTIAL

Before ISO work gets fancy, make sure:

* Windows can build binaries
* WSL/Linux can build binaries
* outputs go somewhere predictable
* repo has one obvious developer entrypoint

### 9. Build the first `.deb` — READY (needs run/verify)

Not a full image yet — first prove packaging.

The deb should:

* install binaries
* create `/srv/crateos`
* install systemd units
* enable/start agent
* optionally drop SSH config

### 10. Build the first qcow2 image — READY (script implemented; needs run/verify)

This should be the first artifact you treat as “real.”

Why qcow2 first:

* fastest iteration
* easiest to boot/test repeatedly
* no USB friction
* proves your image pipeline

### 11. Only then do ISO — READY (xorriso repack implemented; needs run/verify)

Once qcow2 works:

* autoinstall ISO
* install crate packages
* boot into same experience

### 12. First real crate trio — PENDING (start nginx, then postgres)

Once base platform boots:

* `nginx`
* `postgres`
* `redis`

Not all at once. Start with **nginx**, then **postgres**.

---

# Best practical order

If I were sequencing your next commits, I’d do this:

1. **Add Apache-2.0 license**
2. **Clean README/license references**
3. **Create `/srv/crateos` bootstrap logic**
4. **Get `crateos`, `crateos-agent`, `crateos-policy` building**
5. **Add systemd units**
6. **Make SSH drop into Crate console**
7. **Build first `.deb` package**
8. **Build first qcow2 image**
9. **Prove boot + agent + console works**
10. **Implement first crate: nginx**

---

# What not to do next

Do **not** do these yet:

* fancy web UI
* Pi image
* signatures
* enterprise registry
* advanced role model
* full hardware dashboard
* premium licensing flow

Those are all second-wave problems.

---

# The actual milestone you want

Your next real milestone is:

> **“Fresh boot lands in CrateOS, agent is alive, and one crate can be managed.”**
