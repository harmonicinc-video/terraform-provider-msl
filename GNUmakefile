TEST ?= $$(go list ./...)
PKG_NAME = msl

GOLANGCI_LINT_VERSION = v2.6.1
TFPLUGINDOCS_VERSION = v0.25.0

# Local provider install parameters
# Can be overridden: make version=5.1.0 install
# Defaults to the latest git tag (v-prefix stripped), falls back to 0.0.0-dev
version ?= $(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0-dev")
registry_name = registry.terraform.io
namespace = harmonicinc-video
bin_name = terraform-provider-$(PKG_NAME)
build_dir = .build
TF_PLUGIN_DIR ?= ~/.terraform.d/plugins
install_path = $(TF_PLUGIN_DIR)/$(registry_name)/$(namespace)/$(PKG_NAME)/$(version)/$$(go env GOOS)_$$(go env GOARCH)

BIN      = $(CURDIR)/bin
GOCMD = go
GOTEST = $(GOCMD) test
GOBUILD = $(GOCMD) build
GOMODTIDY = $(GOCMD) mod tidy
M = $(shell echo ">")

GOLANGCILINT = $(BIN)/golangci-lint
# Requires: go (for go env GOOS/GOARCH), curl, tar, and sha256sum or shasum.
$(BIN)/golangci-lint: | $(BIN) ; $(info $(M) Installing golangci-lint $(GOLANGCI_LINT_VERSION)...) @
	$Q set -e; \
	_OS=$$(go env GOOS); \
	_ARCH=$$(go env GOARCH); \
	_VERSION="$(GOLANGCI_LINT_VERSION)"; _VERSION="$${_VERSION#v}"; \
	_TARBALL="golangci-lint-$${_VERSION}-$${_OS}-$${_ARCH}.tar.gz"; \
	_BASE="https://github.com/golangci/golangci-lint/releases/download/$(GOLANGCI_LINT_VERSION)"; \
	_TMP=$$(mktemp -d 2>/dev/null || { _D="$${TMPDIR:-/tmp}/golangci-lint-install.$$"; mkdir -p "$${_D}" && echo "$${_D}"; }); \
	curl -sSfL "$${_BASE}/$${_TARBALL}" -o "$${_TMP}/$${_TARBALL}"; \
	curl -sSfL "$${_BASE}/golangci-lint-$${_VERSION}-checksums.txt" -o "$${_TMP}/checksums.txt"; \
	cd "$${_TMP}"; \
	_CHECKSUM_LINES=$$(grep -cF "  $${_TARBALL}" checksums.txt 2>/dev/null || true); \
	[ "$${_CHECKSUM_LINES}" -eq 1 ] || { printf 'Error: expected exactly 1 checksum line for %s, got %s\n' "$${_TARBALL}" "$${_CHECKSUM_LINES}" >&2; exit 1; }; \
	grep -F "  $${_TARBALL}" checksums.txt | \
		if command -v sha256sum >/dev/null 2>&1; then sha256sum -c -; else shasum -a 256 -c -; fi; \
	tar -xzf "$${_TMP}/$${_TARBALL}" -C "$(BIN)" --strip-components=1 \
		"golangci-lint-$${_VERSION}-$${_OS}-$${_ARCH}/golangci-lint"; \
	rm -rf "$${_TMP}"

TFPLUGINDOCS = $(BIN)/tfplugindocs
$(BIN)/tfplugindocs: | $(BIN) ; $(info $(M) Installing tfplugindocs $(TFPLUGINDOCS_VERSION)...) @
	$Q GOBIN=$(BIN) go install github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@$(TFPLUGINDOCS_VERSION)

$(BIN):
	@mkdir -p $@

# Targets
default: build

.PHONY: install
install: build
	mkdir -p $(install_path)
	cp $(build_dir)/$(bin_name) $(install_path)/$(bin_name)_v$(version)

.PHONY: build
build:
	mkdir -p $(build_dir)
	$(GOBUILD) -o $(build_dir)/$(bin_name)

.PHONY: tidy
tidy: ; $(info $(M) Running go mod tidy...) @
	@$(GOMODTIDY)

.PHONY: test
test:
	$(GOTEST) $(TEST) -v $(TESTARGS) -timeout 70m 2>&1

# not used yet, but can be used for running acceptance tests with TF_ACC=1
.PHONY: testacc
testacc:
	TF_ACC=1 $(GOTEST) $(TEST) -v $(TESTARGS) -timeout 300m

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: fmt-check
fmt-check:
	$(eval OUTPUT = $(shell gofmt -l .))
	@if [ "$(OUTPUT)" != "" ]; then\
		echo "Found following files with incorrect format:";\
		echo "$(OUTPUT)";\
		false;\
	fi

.PHONY: terraform-fmtcheck
terraform-fmtcheck:
	terraform fmt -recursive -check -diff

.PHONY: terraform-fmt
terraform-fmt:
	terraform fmt -recursive

.PHONY: lint
lint: | $(GOLANGCILINT) ; $(info $(M) Running golangci-lint...) @
	$Q $(BIN)/golangci-lint run --timeout 5m

.PHONY: docs
docs: | $(TFPLUGINDOCS) ; $(info $(M) Generating docs...) @
	$Q $(BIN)/tfplugindocs generate --provider-name $(PKG_NAME) --website-source-dir templates

.PHONY: clean
clean:
	@rm -rf $(build_dir) $(BIN)
