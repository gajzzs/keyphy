#!/bin/bash

# Remove all DROP rules from OUTPUT chain (aggressive cleanup)
echo "Removing all keyphy iptables rules..."

# Get all rule numbers for DROP rules and delete them in reverse order
iptables -L OUTPUT --line-numbers -n | grep "DROP" | awk '{print $1}' | sort -nr | while read line; do
    iptables -D OUTPUT $line 2>/dev/null
done

echo "Cleanup complete. Remaining rules:"
iptables -L OUTPUT -n