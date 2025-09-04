# Makefile for Cloud Whitelist Manager

# Variables
BINARY_NAME=cloud-whitelist-manager
MAIN_FILE=./cmd/cloud-whitelist-manager/main.go
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Build for current platform
build:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

# Build for Linux (for Docker)
build-linux:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux $(MAIN_FILE)

# Install dependencies
deps:
	$(GOGET) -v ./...

# Update dependencies
deps-update:
	$(GOMOD) tidy

# Format code
deps-fmt:
	$(GOFMT) ./...

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-cover:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

# Run tests with race detection
test-race:
	$(GOTEST) -race ./...

# Vet the code
code-vet:
	$(GOCMD) vet ./...

# Clean build files
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Run the application
run:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	./$(BUILD_DIR)/$(BINARY_NAME)

# Build Docker image
docker-build:
	docker build -t $(BINARY_NAME) .

# Run Docker container
docker-run:
	docker run --rm -v $(PWD)/config.yaml:/app/config.yaml $(BINARY_NAME)

# Run Docker container with docker compose
docker-compose-up:
	docker compose up -d

# Stop Docker containers with docker compose
docker-compose-down:
	docker compose down

# Help
help:
	@echo "Available commands:"
	@echo "  build             - Build for current platform"
	@echo "  build-linux       - Build for Linux"
	@echo "  deps              - Install dependencies"
	@echo "  deps-update       - Update dependencies"
	@echo "  deps-fmt          - Format code"
	@echo "  test              - Run tests"
	@echo "  test-cover        - Run tests with coverage"
	@echo "  test-race         - Run tests with race detection"
	@echo "  code-vet          - Vet the code"
	@echo "  clean             - Clean build files"
	@echo "  run               - Run the application"
	@echo "  docker-build      - Build Docker image"
	@echo "  docker-run        - Run Docker container"
	@echo "  docker-compose-up - Run Docker containers with docker compose"
	@echo "  docker-compose-down - Stop Docker containers with docker compose"
	@echo "  help              - Show this help"

.PHONY: build build-linux deps deps-update deps-fmt test test-cover test-race code-vet clean run docker-build docker-run docker-compose-up docker-compose-down help