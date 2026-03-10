VERSION  ?= 0.1.0+noble1
GOFLAGS  ?= -trimpath
DIST     := dist
BIN      := $(DIST)/bin

# Platform selection (x86, rpi, rpi0)
PLATFORM ?= x86

# Platform-specific GOOS/GOARCH
ifeq ($(PLATFORM),x86)
  GOOS   ?= linux
  GOARCH ?= amd64
  VERSION ?= 0.1.0+noble1
else ifeq ($(PLATFORM),rpi)
  GOOS   ?= linux
  GOARCH ?= arm64
  VERSION ?= 0.1.0+rpi1
else ifeq ($(PLATFORM),rpi0)
  GOOS   ?= linux
  GOARCH ?= arm64
  VERSION ?= 0.1.0+rpi0-1
endif

CMDS     := crateos crateos-agent crateos-policy
DEB_PKGS := crateos crateos-agent crateos-policy

.PHONY: all build build-x86 build-rpi build-rpi0 deb deb-x86 deb-rpi deb-rpi0 iso image-x86 image-rpi image-rpi0 qcow2 rpi clean

all: build

help:
	@echo "CrateOS Build System"
	@echo "Usage: make [PLATFORM=x86|rpi|rpi0] [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build              Build binaries for current PLATFORM (default: x86)"
	@echo "  build-x86          Build x86 binaries"
	@echo "  build-rpi          Build Raspberry Pi binaries"
	@echo "  build-rpi0         Build Raspberry Pi Zero 2 binaries"
	@echo "  deb                Build .deb packages for current PLATFORM"
	@echo "  deb-x86            Build x86 .deb packages"
	@echo "  deb-rpi            Build Raspberry Pi .deb packages"
	@echo "  deb-rpi0           Build Raspberry Pi Zero 2 .deb packages"
	@echo "  image-x86          Build x86 ISO image"
	@echo "  image-rpi          Build Raspberry Pi OS image"
	@echo "  image-rpi0         Build Raspberry Pi Zero 2 OS image"
	@echo "  qcow2              Build QCOW2 VM image (x86 only)"
	@echo "  clean              Remove dist/ directory"
	@echo ""
	@echo "Examples:"
	@echo "  make PLATFORM=x86 image-x86"
	@echo "  make PLATFORM=rpi deb-rpi"
	@echo "  make PLATFORM=rpi0 build-rpi0"

# ── Build ────────────────────────────────────────────────────────────
build: $(addprefix $(BIN)/,$(CMDS))

build-x86: PLATFORM := x86
build-x86: build

build-rpi: PLATFORM := rpi
build-rpi: build

build-rpi0: PLATFORM := rpi0
build-rpi0: build

LDFLAGS  := -X github.com/crateos/crateos/internal/platform.Version=$(VERSION) -X github.com/crateos/crateos/internal/platform.BuildTarget=$(PLATFORM)

$(BIN)/%: cmd/%/main.go
	@mkdir -p $(BIN)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $@ ./cmd/$*

# ── Debian packages ──────────────────────────────────────────
deb: build package-deb

deb-x86: PLATFORM := x86
deb-x86: deb

deb-rpi: PLATFORM := rpi
deb-rpi: deb

deb-rpi0: PLATFORM := rpi0
deb-rpi0: deb

package-deb: build
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

# ── Image builders (platform-specific) ────────────────────────────────────────────────────
image-x86: deb-x86
	@echo "==> building x86 autoinstall ISO"
	bash images/x86/build.sh
	@echo "==> ISO written to $(DIST)/"

iso: image-x86

image-rpi: deb-rpi
	@echo "==> building Raspberry Pi image"
	bash images/rpi/build.sh
	@echo "==> RPi image written to $(DIST)/"

image-rpi0: deb-rpi0
	@echo "==> building Raspberry Pi Zero 2 image"
	bash images/rpi0/build.sh
	@echo "==> RPi0 image written to $(DIST)/"

qcow2: deb-x86
	@echo "==> building qcow2 image"
	bash images/qcow2/build.sh
	@echo "==> qcow2 written to $(DIST)/"

rpi: image-rpi

# ── Cleanup ──────────────────────────────────────────────────────────
clean:
	rm -rf $(DIST)
