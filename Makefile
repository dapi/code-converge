MEMORY_BANK_LINT_VERSION := 2e7324181af3034d9e22a411eb977b6729fae2b8

.PHONY: build dist docs-lint markdown-links memory-bank-lint test verify

build:
	go build -trimpath -buildvcs=false -o reviewer ./cmd/reviewer

test:
	go test ./...

verify: test
	go vet ./...
	$(MAKE) docs-lint

dist:
	go run ./tools/build-dist -version="$${VERSION:-dev}"

docs-lint: memory-bank-lint markdown-links

markdown-links:
	go run ./scripts/check-markdown-links.go README.md AGENTS.md .protocols/memory-bank-integration.md

memory-bank-lint:
	go run github.com/dapi/memory-bank/cmd/memory-bank-lint@$(MEMORY_BANK_LINT_VERSION) --repo-root .
