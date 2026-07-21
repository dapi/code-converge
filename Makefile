MEMORY_BANK_LINT_VERSION := 2e7324181af3034d9e22a411eb977b6729fae2b8
VERSION ?= $(shell tr -d '\n' < VERSION)

.PHONY: build dist docs-lint markdown-links memory-bank-lint print-version release-scripts-check test verify bump-patch bump-minor bump-major release-patch release-minor release-major

build:
	go build -trimpath -buildvcs=false -ldflags="-X github.com/dapi/reviewer/internal/version.Version=$(VERSION)" -o reviewer ./cmd/reviewer

test:
	go test ./...

verify: test release-scripts-check
	go vet ./...
	$(MAKE) docs-lint

dist:
	go run ./tools/build-dist -version="$${VERSION:-dev}"

release-scripts-check:
	bash -n scripts/bump-version scripts/prepare-release
	sh -n scripts/install.sh

print-version:
	@tr -d '\n' < VERSION
	@printf '\n'

bump-patch:
	@bash scripts/bump-version patch

bump-minor:
	@bash scripts/bump-version minor

bump-major:
	@bash scripts/bump-version major

release-patch:
	@bash scripts/prepare-release patch

release-minor:
	@bash scripts/prepare-release minor

release-major:
	@bash scripts/prepare-release major

docs-lint: memory-bank-lint markdown-links

markdown-links:
	go run ./scripts/check-markdown-links.go README.md AGENTS.md .protocols/memory-bank-integration.md

memory-bank-lint:
	go run github.com/dapi/memory-bank/cmd/memory-bank-lint@$(MEMORY_BANK_LINT_VERSION) --repo-root .
