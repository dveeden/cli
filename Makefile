ALL_SRC               := $(shell find . -name "*.go" | grep -v -e vendor)
GOLANGCI_LINT_VERSION := 1.12.2
SERVICE_NAME := cli
MAIN_GO := main.go

include ./mk-include/cc-begin.mk
include ./mk-include/cc-semver.mk
include ./mk-include/cc-protoc.mk
include ./mk-include/cc-go.mk
include ./mk-include/cc-end.mk

CCLOUD_APIS := $(shell go list -f '{{ .Dir }}' -m github.com/confluentinc/ccloudapis)

.PHONY: generate-protoc
generate-protoc:
	echo $(CCLOUD_APIS)
	@which modvendor >/dev/null || go get github.com/goware/modvendor@latest >/dev/null
	@go mod vendor  >/dev/null
	@modvendor -copy="**/*.c **/*.h **/*.proto **/*.sh" -v  >/dev/null
	protoc shared/kafka/*.proto -Ishared/kafka -I$(CCLOUD_APIS) --gogo_out=plugins=grpc:shared/kafka
	protoc shared/connect/*.proto -Ishared/connect -I$(CCLOUD_APIS) --gogo_out=plugins=grpc:shared/connect
	protoc shared/ksql/*.proto -Ishared/ksql -I$(CCLOUD_APIS) --gogo_out=plugins=grpc:shared/ksql

.PHONY: install-plugins
install-plugins:
	@GO111MODULE=on go install ./dist/...

ifeq ($(shell uname),Darwin)
GORELEASER_CONFIG ?= .goreleaser-mac.yml
else
GORELEASER_CONFIG ?= .goreleaser-linux.yml
endif

.PHONY: binary
binary:
	@which goreleaser >/dev/null 2>&1 || go get github.com/goreleaser/goreleaser >/dev/null 2>&1
	@GO111MODULE=on goreleaser release --snapshot --rm-dist -f $(GORELEASER_CONFIG)