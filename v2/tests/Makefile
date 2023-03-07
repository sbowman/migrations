GO_FILES = $(shell find . -path ./.idea -prune -o -type f -name '*.go' -print)
GO_TEST_FLAGS ?= -p 8 -count=1 -cover

default: all

setup_db:
	@dropdb migrations_test || true
	@createdb migrations_test

migrations: $(GO_FILES)
	@go test

all: setup_db migrations
