.PHONY: build run test fmt vet clean all

# Binary name
BINARY_NAME=blascet-photo-advisor
BINARY_PATH=./cmd/blascet-photo-advisor

# Build the binary
build:
	go build -o $(BINARY_NAME) $(BINARY_PATH)

# Run the application
run: build
	./$(BINARY_NAME)

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean

# Build, vet, and test
all: fmt vet build test
