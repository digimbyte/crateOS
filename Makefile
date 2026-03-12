VERSION  ?= 0.1.0+noble1
GOFLAGS  ?= -trimpath
DIST     := dist
BIN      := $(DIST)/bin
IMAGE_FORMAT ?=

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
DEB_ARCH ?= $(GOARCH)

ifeq ($(PLATFORM),x86)
  DEFAULT_IMAGE_FORMAT := iso
  IMAGE_BUILDER_iso    := images/iso/build.sh
  IMAGE_BUILDER_qcow2  := images/qcow2/build.sh
else ifeq ($(PLATFORM),rpi)
  DEFAULT_IMAGE_FORMAT := img
  IMAGE_BUILDER_img    := images/rpi/build.sh
else ifeq ($(PLATFORM),rpi0)
  DEFAULT_IMAGE_FORMAT := img
  IMAGE_BUILDER_img    := images/rpi0/build.sh
endif

ifeq ($(strip $(IMAGE_FORMAT)),)
  IMAGE_FORMAT := $(DEFAULT_IMAGE_FORMAT)
endif

IMAGE_BUILDER := $(IMAGE_BUILDER_$(IMAGE_FORMAT))

.PHONY: all build build-x86 build-rpi build-rpi0 deb deb-x86 deb-rpi deb-rpi0 image image-x86 image-rpi image-rpi0 iso qcow2 rpi clean

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
	@echo "  image              Build final image for PLATFORM/IMAGE_FORMAT"
	@echo "  image-x86          Build x86 ISO image"
	@echo "  image-rpi          Build Raspberry Pi OS image"
	@echo "  image-rpi0         Build Raspberry Pi Zero 2 OS image"
	@echo "  qcow2              Build QCOW2 VM image (x86 only)"
	@echo "  clean              Remove dist/ directory"
	@echo ""
	@echo "Examples:"
	@echo "  make PLATFORM=x86 image-x86"
	@echo "  make PLATFORM=x86 IMAGE_FORMAT=qcow2 image"
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
	@echo "==> normalizing Linux payload line endings"
	@python3 images/common/normalize-linux-payloads.py
	@for pkg in $(DEB_PKGS); do \
		echo "  • packaging $$pkg"; \
		tmpdir=$$(mktemp -d); \
		staging=$$tmpdir/$$pkg; \
		mkdir -p $$staging/DEBIAN; \
		mkdir -p $$staging/usr/local/bin; \
		chmod 755 $$staging; \
		chmod 755 $$staging/DEBIAN; \
		chmod 755 $$staging/usr; \
		chmod 755 $$staging/usr/local; \
		chmod 755 $$staging/usr/local/bin; \
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
		if [ -f $$staging/DEBIAN/control ]; then \
			sed -i "s/^Version: .*/Version: $(VERSION)/" $$staging/DEBIAN/control; \
		fi; \
		if [ -f $$staging/DEBIAN/postinst ]; then \
			sed -i "s/CRATEOS_VERSION:-[^}]*/CRATEOS_VERSION:-$(VERSION)/" $$staging/DEBIAN/postinst; \
		fi; \
		dpkg-deb --build $$staging $$tmpdir/$${pkg}_$(VERSION)_$(DEB_ARCH).deb; \
		cp $$tmpdir/$${pkg}_$(VERSION)_$(DEB_ARCH).deb $(DIST)/$${pkg}_$(VERSION)_$(DEB_ARCH).deb; \
		rm -rf $$tmpdir; \
	done
	@echo ""
	@echo "✓ .deb packages written to $(DIST)/"

# ── Image builders (platform-specific) ────────────────────────────────────────────────────
image:
	@if [ -z "$(IMAGE_BUILDER)" ]; then \
		echo "ERROR: unsupported image format '$(IMAGE_FORMAT)' for platform '$(PLATFORM)'"; \
		exit 1; \
	fi
	@echo "==> building $(PLATFORM) $(IMAGE_FORMAT) image via $(IMAGE_BUILDER)"
	bash $(IMAGE_BUILDER)
	@echo "==> image written to $(DIST)/"

image-x86: PLATFORM := x86
image-x86: IMAGE_FORMAT := iso
image-x86: deb-x86 image

iso: image-x86

image-rpi: PLATFORM := rpi
image-rpi: IMAGE_FORMAT := img
image-rpi: deb-rpi image

image-rpi0: PLATFORM := rpi0
image-rpi0: IMAGE_FORMAT := img
image-rpi0: deb-rpi0 image

qcow2: PLATFORM := x86
qcow2: IMAGE_FORMAT := qcow2
qcow2: deb-x86 image

rpi: image-rpi

# ── Cleanup ──────────────────────────────────────────────────────────
clean:
	rm -rf $(DIST)
