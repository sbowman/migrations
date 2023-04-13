PROJECT = migrate
VERSION ?= 2.0.0

REPO = github.com/sbowman/migrations/v2/cli
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
UNAME := $(shell uname -s)

GO_BUILD_FLAGS ?= "-w -s -X '$(REPO)/config.Version=$(VERSION)' -X '$(REPO)/config.Release=$(RELEASE)' -X '$(REPO)/config.Built=$(BUILD_TIME)'"

GO_FILES = $(shell find . -path ./.idea -prune -o -type f -name '*.go' -print)
GO_TEST_FLAGS ?= -p 8 -count=1 -cover

VERBOSITY ?= 2

default: $(PROJECT)

# domain/types/private_key_cipher.go security/key_manager_cipher.go
$(PROJECT): $(GO_FILES)
	@go build -ldflags=$(GO_BUILD_FLAGS)

install:
	@go install -ldflags=$(GO_BUILD_FLAGS)

.PHONY: clean
clean:
	@rm $(PROJECT)
