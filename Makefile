VERSION  ?= 0.1.0-dev
GOFLAGS  ?= -trimpath
DIST     := dist
BIN      := $(DIST)/bin

CMDS     := crateos crateos-agent crateos-policy
DEB_PKGS := crateos crateos-agent crateos-policy

.PHONY: all build deb iso qcow2 rpi clean

all: build

# ── Build ────────────────────────────────────────────────────────────
build: $(addprefix $(BIN)/,$(CMDS))

$(BIN)/%: cmd/%/main.go
	@mkdir -p $(BIN)
	go build $(GOFLAGS) -o $@ ./cmd/$*

# ── Debian packages ──────────────────────────────────────────────────
deb: build
	@for pkg in $(DEB_PKGS); do \
		echo "==> packaging $$pkg"; \
		staging=$(DIST)/deb-staging/$$pkg; \
		rm -rf $$staging; \
		mkdir -p $$staging/DEBIAN; \
		mkdir -p $$staging/usr/local/bin; \
		cp packaging/deb/$$pkg/DEBIAN/* $$staging/DEBIAN/; \
		chmod 755 $$staging/DEBIAN/postinst 2>/dev/null || true; \
		cp $(BIN)/$$pkg $$staging/usr/local/bin/; \
		if [ -d packaging/deb/$$pkg/etc ]; then \
			cp -r packaging/deb/$$pkg/etc $$staging/; \
		fi; \
		if [ -d packaging/deb/$$pkg/lib ]; then \
			cp -r packaging/deb/$$pkg/lib $$staging/; \
		fi; \
		dpkg-deb --build $$staging $(DIST)/$$pkg_$(VERSION)_amd64.deb; \
	done
	@echo "==> .deb packages written to $(DIST)/"

# ── ISO (autoinstall) ───────────────────────────────────────────────
iso: deb
	@echo "==> building autoinstall ISO"
	bash images/iso/build.sh
	@echo "==> ISO written to $(DIST)/"

# ── qcow2 VM image ──────────────────────────────────────────────────
qcow2: deb
	@echo "==> building qcow2 image"
	bash images/qcow2/build.sh
	@echo "==> qcow2 written to $(DIST)/"

# ── Raspberry Pi (stub) ─────────────────────────────────────────────
rpi:
	@echo "rpi target not yet implemented"

# ── Cleanup ──────────────────────────────────────────────────────────
clean:
	rm -rf $(DIST)
