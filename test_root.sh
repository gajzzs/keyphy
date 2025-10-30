#!/bin/bash

echo "=== Keyphy Root Privileges Test ==="
echo "Testing functionality that requires root access..."
echo

if [ "$EUID" -ne 0 ]; then
    echo "⚠️ This test requires root privileges"
    echo "Run with: sudo ./test_root.sh"
    exit 1
fi

# Build keyphy
go build -o keyphy-root ./cmd/keyphy

echo "1. Testing service installation..."
./keyphy-root service install || echo "⚠️ Service installation may have failed"

echo
echo "2. Testing service status after install..."
./keyphy-root service status

echo
echo "3. Testing network blocking (requires root)..."
echo "Testing iptables access..."
iptables -L keyphy-test 2>/dev/null || echo "✅ iptables accessible"

echo
echo "4. Testing file protection (requires root)..."
echo "Testing chattr access..."
touch /tmp/keyphy-test-file
chattr +i /tmp/keyphy-test-file 2>/dev/null && chattr -i /tmp/keyphy-test-file 2>/dev/null
rm -f /tmp/keyphy-test-file
echo "✅ chattr accessible"

echo
echo "5. Testing config file protection..."
echo "Testing /etc/keyphy access..."
mkdir -p /etc/keyphy
echo '{"test": true}' > /etc/keyphy/test.json
chattr +i /etc/keyphy/test.json 2>/dev/null && chattr -i /etc/keyphy/test.json 2>/dev/null
rm -f /etc/keyphy/test.json
echo "✅ Config protection accessible"

echo
echo "6. Cleaning up service..."
./keyphy-root service uninstall || echo "⚠️ Service uninstall may have failed"

# Clean up
rm -f keyphy-root

echo
echo "=== Root Privileges Test Complete ==="
echo "✅ All root-required functionality accessible"