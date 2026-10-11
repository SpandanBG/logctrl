# Minimum total coverage (%) enforced in CI. Raise it as coverage grows.
COVER_MIN := 94
# Everything but `main` (needs a real terminal) and the test helpers.
COVER_PKGS = $(shell go list ./... | grep -v -e '^github.com/SpandanBG/logctrl$$' -e /internal/)

.PHONY: mock dev build test cover hooks

mock:
	node ./tests/tools/mockLogger.js

dev:
	go run main.go

build:
	go build -o bin/out main.go

test:
	go vet ./...
	go test -race ./...

cover:
	go test -race -coverprofile=cover.out $(COVER_PKGS)
	@go tool cover -func=cover.out | awk -v min=$(COVER_MIN) '/^total:/ { \
		sub("%", "", $$3); \
		if ($$3 + 0 < min) { printf "coverage %s%% is below the %s%% minimum\n", $$3, min; exit 1 } \
		printf "coverage %s%% (minimum %s%%)\n", $$3, min }'

# Use the committed git hooks in .githooks/
hooks:
	git config core.hooksPath .githooks
