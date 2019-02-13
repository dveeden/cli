ALL_SRC               	:= $(shell find . -name "*.go" | grep -v -e vendor)
GOLANGCI_LINT_VERSION 	:= 1.12.2
SERVICE_NAME 			:= cli
MAIN_GO 				:= main.go

include ./mk-include/cc-begin.mk
include ./mk-include/cc-semver.mk
include ./mk-include/cc-protoc.mk
include ./mk-include/cc-go.mk
include ./mk-include/cc-end.mk

.PHONY: deps
deps:
	@GO111MODULE=on go mod vendor
	@modvendor -copy="**/*.c **/*.h **/*.proto **/*.sh"

.PHONY: generate-protoc
generate: deps
	$(foreach pkg, $(shell find ./shared -type d -maxdepth 1 -not -path "." -not -path "./shared"), $(shell protoc $(pkg)/*.proto --gogo_out=plugins=grpc:. -I vendor -I vendor/github.com/confluentinc/ccloudapis -I vendor/github.com/gogo/protobuf/protobuf -I .))

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