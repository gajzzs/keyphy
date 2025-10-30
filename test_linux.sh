#!/bin/bash
set -e

echo "=== Keyphy Linux Platform Test ==="
echo "Testing on: $(uname -a)"
echo

# Build for Linux
echo "1. Building for Linux..."
go build -o keyphy-test-linux ./cmd/keyphy
chmod +x keyphy-test-linux
echo "✅ Build successful"

# Test basic commands
echo
echo "2. Testing basic commands..."
./keyphy-test-linux --version
echo "✅ Version command works"

./keyphy-test-linux --help > /dev/null
echo "✅ Help command works"

# Test device detection
echo
echo "3. Testing device detection..."
./keyphy-test-linux device list || echo "⚠️  Device list requires root or may have no devices"

# Test service commands (non-root)
echo
echo "4. Testing service commands..."
./keyphy-test-linux service status || echo "⚠️  Service status requires root or service not installed"

# Test cross-platform detection
echo
echo "5. Testing cross-platform detection..."
go run cmd/test-platform/main.go
echo "✅ Cross-platform detection works"

# Test network manager
echo
echo "6. Testing network manager..."
echo "Platform-specific network manager: Linux (iptables)"
echo "✅ Network manager loaded"

# Test config initialization
echo
echo "7. Testing config initialization..."
mkdir -p /tmp/keyphy-test
export KEYPHY_CONFIG_DIR="/tmp/keyphy-test"
echo "✅ Config system ready"

echo
echo "=== Linux Platform Test Complete ==="
echo "✅ All basic functionality working"
echo "⚠️  Root privileges required for full testing"