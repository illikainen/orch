# On Linux, every command that's executed with $(SANDBOX) is executed in a
# bubblewrap container without network access and with limited access to the
# filesystem.

OUTPUT ?= $(shell pwd)/build/
OUTPUT_RELEASE ?= $(OUTPUT)release/
OUTPUT_TOOLS ?= $(OUTPUT)tools/

MODULE := $(shell grep '^module' go.mod|cut -d' ' -f2)
NAME := $(shell basename $(MODULE))
VERSION := $(shell jq .Version src/metadata/metadata.json 2>/dev/null || echo "0.0.0")

SRC := main.go ./src ./tools
GOFER := go run $(shell pwd)/tools/gofer.go
SANDBOX := $(GOFER) sandbox

TOOL_NILERR := $(shell $(GOFER) mod-path -mod github.com/gostaticanalysis/nilerr)/cmd/nilerr
TOOL_ERRCHECK := $(shell $(GOFER) mod-path -mod github.com/kisielk/errcheck)
TOOL_REVIVE := $(shell $(GOFER) mod-path -mod github.com/mgechev/revive)
TOOL_GOIMPORTS := $(shell $(GOFER) mod-path -mod golang.org/x/tools)/cmd/goimports
TOOL_STATICCHECK := $(shell $(GOFER) mod-path -mod honnef.co/go/tools)/cmd/staticcheck
TOOL_GOSEC := ./tools/gosec.go

export CGO_ENABLED := 0
export GO111MODULE := on
export GOFLAGS := -mod=readonly
export GOSUMDB := sum.golang.org
export REAL_GOPROXY := $(shell go env GOPROXY)
export GOPROXY := off

# Unfortunately there is no Go-specific way of pinning the CA for GOPROXY.
# The go.pem file is created by the `pin` target in this Makefile.
export SSL_CERT_FILE := ./go.pem
export SSL_CERT_DIR := /path/does/not/exist/to/pin/ca

export PATH := $(OUTPUT_TOOLS):$(PATH)

define PIN_EXPLANATION
# The checksums for go.sum and go.mod are pinned because `go mod` with
# `-mod=readonly` isn't read-only.  The `go mod` commands will still modify the
# dependency tree if they find it necessary (e.g., to add a missing module or
# module checksum).
#
# Run `make pin` to update this file.
endef
export PIN_EXPLANATION

all: build

tidy:
	@GOPROXY=$(REAL_GOPROXY) go mod tidy
	@$(SANDBOX) go mod verify

prepare-offline: tidy
	@GOPROXY=$(REAL_GOPROXY) go list -m -json all >/dev/null

build:
	@mkdir -p $(OUTPUT)
	@$(SANDBOX) env RELEASE=0 go generate
	@$(SANDBOX) go build -ldflags "-s -w" -o $(OUTPUT)
	@$(SANDBOX) echo "output stored in $(OUTPUT)"

debug:
	@mkdir -p $(OUTPUT)
	@$(SANDBOX) env RELEASE=0 go generate
	@$(SANDBOX) go build -gcflags "all=-N -l" -o $(OUTPUT)
	@$(SANDBOX) echo "output stored in $(OUTPUT)"

release: clean
	@mkdir -p $(OUTPUT_RELEASE)
	@$(SANDBOX) env RELEASE=1 go generate
	@$(SANDBOX) echo "building $(NAME)-$(VERSION) for linux-amd64"
	@$(SANDBOX) -os=linux -arch=amd64 go build -ldflags "-s -w" -o $(OUTPUT_RELEASE)$(NAME)-$(VERSION)-linux-amd64
	@$(SANDBOX) echo "building $(NAME)-$(VERSION) for linux-arm64"
	@$(SANDBOX) -os=linux -arch=arm64 go build -ldflags "-s -w" -o $(OUTPUT_RELEASE)$(NAME)-$(VERSION)-linux-arm64
	@$(SANDBOX) echo "building $(NAME)-$(VERSION) for darwin-arm64"
	@$(SANDBOX) -os=darwin -arch=arm64 go build -ldflags "-s -w" -o $(OUTPUT_RELEASE)$(NAME)-$(VERSION)-darwin-arm64
	@$(SANDBOX) echo "building $(NAME)-$(VERSION) for windows-amd64"
	@$(SANDBOX) -os=windows -arch=amd64 go build -ldflags "-s -w" -o $(OUTPUT_RELEASE)$(NAME)-$(VERSION)-windows-amd64.exe
	@$(SANDBOX) echo "output stored in $(OUTPUT_RELEASE)"

tools:
	@mkdir -p $(OUTPUT_TOOLS)
	@$(SANDBOX) go build -o $(OUTPUT_TOOLS) $(TOOL_NILERR)
	@$(SANDBOX) go build -o $(OUTPUT_TOOLS) $(TOOL_ERRCHECK)
	@$(SANDBOX) go build -o $(OUTPUT_TOOLS) $(TOOL_REVIVE)
	@$(SANDBOX) go build -o $(OUTPUT_TOOLS) $(TOOL_GOSEC)
	@$(SANDBOX) go build -o $(OUTPUT_TOOLS) $(TOOL_GOIMPORTS)
	@$(SANDBOX) go build -o $(OUTPUT_TOOLS) $(TOOL_STATICCHECK)
	@$(SANDBOX) echo "output stored in $(OUTPUT_TOOLS)"

clean:
	@$(SANDBOX) go clean
	@$(SANDBOX) go clean -cache
	@$(SANDBOX) rm -rfv $(OUTPUT_TOOLS) $(OUTPUT)

distclean:
	@$(SANDBOX) git clean -d -f -x

test:
	@$(SANDBOX) mkdir -p $(OUTPUT)
	@$(SANDBOX) go test -v -coverprofile=$(OUTPUT)/.coverage -coverpkg=./... ./...

coverage:
	@$(SANDBOX) go tool cover -func $(OUTPUT)/.coverage

check-nilerr:
	@$(SANDBOX) echo "Running nilerr"
	@$(SANDBOX) nilerr ./...

check-errcheck:
	@$(SANDBOX) echo "Running errcheck"
	@$(SANDBOX) errcheck ./...

check-revive:
	@$(SANDBOX) echo "Running revive"
	@$(SANDBOX) revive -config revive.toml -set_exit_status ./...

check-gosec:
	@$(SANDBOX) echo "Running gosec"
	@$(SANDBOX) gosec -quiet ./...

check-staticcheck:
	@$(SANDBOX) echo "Running staticcheck"
	@$(SANDBOX) staticcheck ./...

check-vet:
	@$(SANDBOX) echo "Running go vet"
	@$(SANDBOX) go vet ./...

check-fmt:
	@$(SANDBOX) echo "Running gofmt"
	@$(SANDBOX) gofmt -d -l $(SRC)

check-imports:
	@$(SANDBOX) echo "Running goimports"
	@$(SANDBOX) goimports -d -local $(MODULE) -l $(SRC)

check: verify check-nilerr check-errcheck check-revive check-gosec check-staticcheck check-vet check-fmt check-imports

fix-fmt:
	@$(SANDBOX) gofmt -w -l $(SRC)

fix-imports:
	@$(SANDBOX) goimports -w -l -local $(MODULE) $(SRC)

fix: verify fix-fmt fix-imports

pin:
	@$(SANDBOX) echo "$$PIN_EXPLANATION" > go.pin
	@$(SANDBOX) sha256sum go.sum go.mod >> go.pin
	@test -f /etc/ssl/certs/GTS_Root_R1.pem && test -f /etc/ssl/certs/GTS_Root_R4.pem && \
		cat /etc/ssl/certs/GTS_Root_R1.pem /etc/ssl/certs/GTS_Root_R4.pem > go.pem || true

verify:
	@$(SANDBOX) sha256sum --strict --check go.pin
	@$(SANDBOX) go mod verify

qa: build check test coverage

.PHONY: all tidy build release debug tools clean distclean
.PHONY: test coverage prepare-offline
.PHONY: check-nilerr check-errcheck check-revive check-gosec check-staticcheck check-vet check-fmt check-imports check
.PHONY: fix-imports fix-fmt fix pin verify qa
