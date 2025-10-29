BINARY_NAME=keyphy
BUILD_DIR=build
INSTALL_DIR=/usr/local/bin
SERVICE_DIR=/etc/systemd/system

.PHONY: build install uninstall clean service

build:
	@echo "Building keyphy..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/keyphy

install: build
	@echo "Installing keyphy..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	@sudo chmod +x $(INSTALL_DIR)/$(BINARY_NAME)
	@sudo mkdir -p /etc/keyphy
	@echo "Installation complete. Run 'keyphy --help' to get started."

service:
	@echo "Installing systemd service..."
	@sudo sh -c 'cat > $(SERVICE_DIR)/keyphy.service << EOF
[Unit]
Description=Keyphy Access Control Daemon
After=network.target

[Service]
Type=simple
ExecStart=$(INSTALL_DIR)/$(BINARY_NAME) service start
ExecStop=$(INSTALL_DIR)/$(BINARY_NAME) service stop
Restart=always
RestartSec=5
User=root
KillMode=mixed
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
EOF'
	@sudo systemctl daemon-reload
	@echo "Service installed. Use 'sudo systemctl enable keyphy' to enable on boot."

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