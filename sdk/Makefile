.PHONY: generate test build clean fmt lint deps install-tools

generate:
	oapi-codegen -config .oapi-codegen.yaml ../openapi/openapi.yaml

test:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

build: generate test

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

clean:
	rm -f pkg/types/generated.go
	rm -f coverage.out coverage.html

deps:
	go mod download
	go mod tidy

install-tools:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

.DEFAULT_GOAL := build
