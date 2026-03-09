VERSION  ?= 0.1.0+noble1
GOFLAGS  ?= -trimpath
DIST     := dist
BIN      := $(DIST)/bin

CMDS     := crateos crateos-agent crateos-policy
DEB_PKGS := crateos crateos-agent crateos-policy

.PHONY: all build deb iso qcow2 rpi clean

all: build

# ── Build ────────────────────────────────────────────────────────────
build: $(addprefix $(BIN)/,$(CMDS))

LDFLAGS  := -X github.com/crateos/crateos/internal/platform.Version=$(VERSION)

$(BIN)/%: cmd/%/main.go
	@mkdir -p $(BIN)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@ ./cmd/$*

# ── Debian packages ──────────────────────────────────────────────────
deb: build
	@for pkg in $(DEB_PKGS); do \
		echo "==> packaging $$pkg"; \
		staging=$(DIST)/deb-staging/$$pkg; \
		rm -rf $$staging; \
		mkdir -p $$staging/DEBIAN; \
		mkdir -p $$staging/usr/local/bin; \
		cp packaging/deb/$$pkg/DEBIAN/* $$staging/DEBIAN/; \
		chmod 755 $$staging/DEBIAN/* 2>/dev/null || true; \
		cp $(BIN)/$$pkg $$staging/usr/local/bin/; \
		if [ -d packaging/deb/$$pkg/etc ]; then \
			cp -r packaging/deb/$$pkg/etc $$staging/; \
		fi; \
		if [ -d packaging/deb/$$pkg/lib ]; then \
			cp -r packaging/deb/$$pkg/lib $$staging/; \
		fi; \
		if [ -d packaging/deb/$$pkg/usr ]; then \
			cp -r packaging/deb/$$pkg/usr $$staging/; \
			chmod 755 $$staging/usr/local/bin/* 2>/dev/null || true; \
		fi; \
		if [ "$$pkg" = "crateos-agent" ]; then \
			mkdir -p $$staging/usr/share/crateos/defaults; \
			cp -r packaging/config $$staging/usr/share/crateos/defaults/; \
		fi; \
		# Inject version into postinst (CRATEOS_VERSION env placeholder).
		if [ -f $$staging/DEBIAN/control ]; then \
			sed -i "s/^Version: .*/Version: $(VERSION)/" $$staging/DEBIAN/control; \
		fi; \
		if [ -f $$staging/DEBIAN/postinst ]; then \
			sed -i "s/CRATEOS_VERSION:-[^}]*/CRATEOS_VERSION:-$(VERSION)/" $$staging/DEBIAN/postinst; \
		fi; \
		dpkg-deb --build $$staging $(DIST)/$${pkg}_$(VERSION)_amd64.deb; \
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
