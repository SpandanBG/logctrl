# Minimum total coverage (%) enforced in CI. Raise it as coverage grows.
COVER_MIN := 97
# Everything but `main` (needs a real terminal) and the test helpers.
COVER_PKGS = $(shell go list ./... | grep -v -e '^github.com/SpandanBG/logctrl$$' -e /internal/)
# Tests count for every package they exercise, not just their own, so code
# tested through a seam in another package (e.g. components via ui) is covered.
comma := ,
COVER_PKG_LIST = $(subst $(eval) ,$(comma),$(strip $(COVER_PKGS)))

.PHONY: mock dev build test cover hooks

mock:
	node ./tests/tools/mockLogger.js

dev:
	go run main.go

build:
	go build -o bin/out main.go

test:
	@test -z "$$(gofmt -l .)" || (echo "not gofmt'ed (run gofmt -w):"; gofmt -l .; exit 1)
	go vet ./...
	scripts/check-dividers.sh
	go test -race ./...

# -count=1: a cached test result keeps the coverage of files since changed or
# deleted, so the total would be wrong.
cover:
	go test -race -count=1 -coverprofile=cover.out -coverpkg=$(COVER_PKG_LIST) $(COVER_PKGS)
	@go tool cover -func=cover.out | awk -v min=$(COVER_MIN) '/^total:/ { \
		sub("%", "", $$3); \
		if ($$3 + 0 < min) { printf "coverage %s%% is below the %s%% minimum\n", $$3, min; exit 1 } \
		printf "coverage %s%% (minimum %s%%)\n", $$3, min }'

# Use the committed git hooks in .githooks/
hooks:
	git config core.hooksPath .githooks
