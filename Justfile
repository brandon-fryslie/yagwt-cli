# Justfile for YAGWT

# Default recipe (show help)
default:
    @just --list

# Build the binary
build:
    @echo "Building yagwt..."
    @mkdir -p bin
    go build -o bin/yagwt ./cmd/yagwt

# Build for all platforms (release builds)
build-all:
    @echo "Building for all platforms..."
    @mkdir -p dist
    GOOS=darwin GOARCH=amd64 go build -o dist/yagwt-darwin-amd64 ./cmd/yagwt
    GOOS=darwin GOARCH=arm64 go build -o dist/yagwt-darwin-arm64 ./cmd/yagwt
    GOOS=linux GOARCH=amd64 go build -o dist/yagwt-linux-amd64 ./cmd/yagwt
    GOOS=linux GOARCH=arm64 go build -o dist/yagwt-linux-arm64 ./cmd/yagwt
    @echo "Builds complete in dist/"

# Run tests
test:
    go test -v ./...

# Run tests with coverage
test-coverage:
    @echo "Running tests with coverage..."
    go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
    go tool cover -html=coverage.txt -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run tests with coverage percentage
test-cov:
    go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
    go tool cover -func=coverage.txt

# Run linter
lint:
    @echo "Running golangci-lint..."
    golangci-lint run ./...

# Format code
fmt:
    @echo "Formatting code..."
    go fmt ./...
    gofmt -s -w .

# Check formatting (for CI)
fmt-check:
    @echo "Checking code formatting..."
    @test -z "$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

# Tidy dependencies
tidy:
    go mod tidy
    go mod verify

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    rm -rf bin/ dist/ coverage.txt coverage.html
    go clean

# Install binary to local system
install: build
    @echo "Installing yagwt to /usr/local/bin..."
    sudo cp bin/yagwt /usr/local/bin/yagwt
    @echo "Installed successfully!"

# Uninstall binary from local system
uninstall:
    @echo "Uninstalling yagwt..."
    sudo rm -f /usr/local/bin/yagwt
    @echo "Uninstalled successfully!"

# Run all checks (lint, test, build)
check: lint test build
    @echo "All checks passed!"

# Watch for changes and run tests (requires entr)
watch:
    find . -name '*.go' | entr -c just test

# Generate mocks (requires mockgen)
mocks:
    @echo "Generating mocks..."
    go generate ./...

# Run benchmarks
bench:
    go test -bench=. -benchmem ./...

# Release (create git tag and push)
release VERSION:
    @echo "Creating release {{VERSION}}..."
    git tag -a {{VERSION}} -m "Release {{VERSION}}"
    git push origin {{VERSION}}
    @echo "Release {{VERSION}} created and pushed!"

# Download dependencies
deps:
    @echo "Downloading dependencies..."
    go mod download

# Verify dependencies
verify:
    go mod verify

# Update dependencies
update:
    @echo "Updating dependencies..."
    go get -u ./...
    go mod tidy

# Show project statistics
stats:
    @echo "Project Statistics:"
    @echo "-------------------"
    @echo "Lines of Go code:"
    @find . -name '*.go' -not -path './vendor/*' | xargs wc -l | tail -1
    @echo ""
    @echo "Number of packages:"
    @go list ./... | wc -l
    @echo ""
    @echo "Dependencies:"
    @go list -m all | wc -l

# Run security checks
security:
    @echo "Running security checks..."
    go list -json -m all | nancy sleuth

# Generate documentation
docs:
    @echo "Generating documentation..."
    godoc -http=:6060
