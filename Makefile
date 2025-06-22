.PHONY: build install test clean lint deps doctor setup release

BINARY_NAME=env-sync
BIN_DIR=bin

build:
	@echo "Building the binary..."
	@go build -o $(BIN_DIR)/$(BINARY_NAME) ./cmd/main.go

install:
	@echo "Installing the binary..."
	@go install ./cmd/main.go

deps:
	@echo "Installing system dependencies..."
	@if [ "$(shell uname)" = "Darwin" ] || [ "$(shell uname)" = "Linux" ]; then \
		chmod +x scripts/install-deps.sh && ./scripts/install-deps.sh; \
	else \
		powershell -ExecutionPolicy Bypass -File scripts/install-deps.ps1; \
	fi

doctor: build
	@echo "Running system health check..."
	@./$(BIN_DIR)/$(BINARY_NAME) doctor

setup: deps build doctor
	@echo "🎉 env-sync setup complete!"

test:
	@echo "Running tests..."
	@go test ./...

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BIN_DIR)

lint:
	@echo "Linting code..."
	@golangci-lint run

release:
	@echo "Creating release..."
	@goreleaser release --snapshot --clean 