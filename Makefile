commit = $(shell git rev-parse --short HEAD)

.PHONY: test
test: ## Run go test
	go test ./...

.PHONY: clean
clean: ## Clean the workspace
	rm -rf conxec
	rm -rf bin/
	rm -rf dist/

.PHONY: build
build: ## Build the binary
	go build -ldflags "-X main.version="dev" -X main.commit=$(commit)" -o conxec