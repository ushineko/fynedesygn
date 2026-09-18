default: help

.PHONY: help
help: ## Show this help
	@echo
	@echo "Available commands:"
	@echo
	@awk -F ':|##' '/^[^\t].+?:.*?##/ {printf "\033[36m%-30s\033[0m %s\n", $$1, $$NF}' $(MAKEFILE_LIST)

BINDIR=$(shell go env GOPATH)
MODULE=github.com/ushineko/fynedesygn

LINT_NAME?=golangci-lint
LINT_VERSION?=v2.12.2
LINT_PROGRAM=$(LINT_NAME)-$(LINT_VERSION)

# Release asset coordinates for the pinned linter version.
# (The upstream install.sh is not used: its checksum extraction matches the
# .sbom.json asset line and fails verification on recent releases.)
LINT_VERSION_NUM=$(LINT_VERSION:v%=%)
LINT_BASE_URL=https://github.com/golangci/golangci-lint/releases/download/$(LINT_VERSION)

.PHONY: install-lint
install-lint: $(BINDIR)/bin/$(LINT_PROGRAM) ## Install the pinned linter

$(BINDIR)/bin/$(LINT_PROGRAM):
	@echo "Setting up $(LINT_PROGRAM) ..."
	@set -e; \
	os=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	arch=$$(uname -m); \
	case "$$arch" in x86_64) arch=amd64;; aarch64|arm64) arch=arm64;; esac; \
	dist="$(LINT_NAME)-$(LINT_VERSION_NUM)-$$os-$$arch"; \
	tmp=$$(mktemp -d); trap 'rm -rf "$$tmp"' EXIT; \
	curl -fsSL "$(LINT_BASE_URL)/$$dist.tar.gz" -o "$$tmp/$$dist.tar.gz"; \
	curl -fsSL "$(LINT_BASE_URL)/$(LINT_NAME)-$(LINT_VERSION_NUM)-checksums.txt" -o "$$tmp/checksums.txt"; \
	want=$$(awk -v f="$$dist.tar.gz" '$$2 == f {print $$1}' "$$tmp/checksums.txt"); \
	got=$$( (sha256sum "$$tmp/$$dist.tar.gz" 2>/dev/null || shasum -a 256 "$$tmp/$$dist.tar.gz") | awk '{print $$1}'); \
	if [ -z "$$want" ] || [ "$$want" != "$$got" ]; then echo "checksum mismatch for $$dist.tar.gz: want '$$want' got '$$got'"; exit 1; fi; \
	tar -C "$$tmp" -xzf "$$tmp/$$dist.tar.gz"; \
	mkdir -p "$(BINDIR)/bin"; \
	mv -v "$$tmp/$$dist/$(LINT_NAME)" "$(BINDIR)/bin/$(LINT_PROGRAM)"

.PHONY: setup
setup: install-lint ## Set up the machine for local development
	@echo "Make sure your PATH includes $$(go env GOPATH)/bin."

# golangci-lint type-checks against the standard library sources of whichever Go
# it finds, using a go/types built into the linter binary. A linter built with
# Go 1.26 panics outright ("file requires newer Go version go1.27") on a machine
# whose GOROOT is 1.27. go.mod deliberately carries no `toolchain` line, so the
# pin lives here instead, matching the Go that this linter release was built
# with. Bump it together with LINT_VERSION.
LINT_GO_TOOLCHAIN?=go1.26.0

.PHONY: lint
lint: export GOTOOLCHAIN = $(LINT_GO_TOOLCHAIN)
lint: install-lint ## Lint the module
	@go version
	$(BINDIR)/bin/$(LINT_PROGRAM) run --timeout 5m0s --config config/.golangci-$(LINT_VERSION).yml ./...

# Every test in this module runs headless: the Fyne test driver draws into
# memory and needs no DISPLAY or WAYLAND_DISPLAY.
.PHONY: test
test: ## Run the tests with the race detector
	@go test -race ./...

.PHONY: coverage
coverage: ## Run the tests and open a coverage report
	@go test -coverprofile coverage.out ./...
	@go tool cover -html=coverage.out
	@rm -f coverage.out

.PHONY: vuln
vuln: ## Scan the module for known vulnerabilities
	@govulncheck ./...

# Mermaid sources (*.mmd) are rendered to PNG at development time and the PNGs
# are committed and embedded; consumers of the library never need node or a
# browser. See docs/mermaid.md.
.PHONY: generate
generate: ## Render missing mermaid diagrams and other generated files (needs mmdc)
	@go generate ./...

.PHONY: check-diagrams
check-diagrams: ## Fail if a mermaid source has no image or an image has no source
	@go run ./cmd/fynedesygn-mermaid -check -root docs
	@go run ./cmd/fynedesygn-mermaid -check -root examples/document-viewer

.PHONY: gallery
gallery: ## Build the reference gallery for the host platform (needs CGO)
	CGO_ENABLED=1 go build -trimpath -o fynedesygn-gallery ./cmd/fynedesygn-gallery

.PHONY: build-examples
build-examples: ## Build every example program into bin/
	@mkdir -p bin
	@for d in examples/*/; do n=$$(basename $$d); CGO_ENABLED=1 go build -trimpath -o bin/$$n ./$$d || exit 1; echo "  bin/$$n"; done

.PHONY: screenshots
screenshots: gallery ## Refresh docs/img from the gallery (KDE/Wayland; needs kdotool, spectacle, Pillow)
	./tools/screenshot.sh --all

.PHONY: clean
clean: ## Remove build output
	@rm -rf fynedesygn-gallery fynedesygn-gallery.exe coverage.out bin/
