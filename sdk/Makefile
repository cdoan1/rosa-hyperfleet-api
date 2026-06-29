.PHONY: generate test build clean fmt lint deps install-tools

# Generate types from OpenAPI spec
generate:
	oapi-codegen -config .oapi-codegen.yaml ../openapi/openapi.yaml

# Run tests
test:
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Build (generate + test)
build: generate test

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run ./...

# Clean generated files
clean:
	rm -f pkg/types/generated.go
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	go mod download
	go mod tidy

# Install oapi-codegen
install-tools:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

.DEFAULT_GOAL := build
