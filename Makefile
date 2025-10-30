BINARY_NAME=keyphy
BUILD_DIR=build
INSTALL_DIR=/usr/local/bin
SERVICE_DIR=/etc/systemd/system
VERSION=$(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build install uninstall clean service test deps release

build:
	@echo "Building keyphy..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/keyphy

install: build
	@echo "Installing keyphy..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo mkdir -p /etc/keyphy
	@echo "Installation complete. Run 'keyphy --help' to get started."

service:
ifeq ($(shell uname),Darwin)
	@echo "Installing macOS launchd service..."
	@sudo cp com.keyphy.daemon.plist /Library/LaunchDaemons/
	@sudo launchctl bootstrap system /Library/LaunchDaemons/com.keyphy.daemon.plist
	@echo "macOS service installed"
else
	@echo "Installing Linux systemd service..."
	@sudo cp keyphy.service $(SERVICE_DIR)/
	@sudo systemctl daemon-reload
	@echo "Service installed. Use 'sudo systemctl enable keyphy' to enable on boot."
endif

uninstall:
	@echo "Uninstalling keyphy..."
	@sudo systemctl stop keyphy 2>/dev/null || true
	@sudo systemctl disable keyphy 2>/dev/null || true
	@sudo rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo rm -f $(SERVICE_DIR)/keyphy.service
	@sudo systemctl daemon-reload
	@echo "Uninstallation complete."

clean:
	@echo "Cleaning build files..."
	@rm -rf $(BUILD_DIR)

test:
	@echo "Running tests..."
	@go test ./...

deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

release:
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/keyphy
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/keyphy
	@echo "Release binaries built in $(BUILD_DIR)/"
