.DEFAULT_GOAL := help

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*## "; printf "\nUsage:\n  make <target>\n\nTargets:\n"} \
		/^([a-zA-Z_-]+):.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run unit tests with race detector
	go test --race ./...

lint: ## Run golangci-lint
	golangci-lint run

mocks: ## Generate mocks using mockery
	mockery

tidy: ## Run go mod tidy
	go mod tidy

fmt: ## Format code with gofmt
	gofmt -w .

fields: ## Fix field alignment
	fieldalignment -fix ./...


# Temporary helpers to initialize the project structure
# Commands below can be removed once project is initialized
init: tidy fmt 

