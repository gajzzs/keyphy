#!/bin/bash

# Remove only keyphy-specific iptables rules
echo "Removing keyphy iptables rules..."

# Remove Google IP blocks (YouTube)
youtube_ips=("142.250.0.0/15" "172.217.0.0/16" "216.58.192.0/19" "74.125.0.0/16")
for ip in "${youtube_ips[@]}"; do
    while iptables -D OUTPUT -d "$ip" -j DROP 2>/dev/null; do
        echo "Removed IP block: $ip"
    done
done

# Remove DoH server blocks
doh_servers=("1.1.1.1" "8.8.8.8" "9.9.9.9" "208.67.222.222")
for server in "${doh_servers[@]}"; do
    while iptables -D OUTPUT -p tcp -d "$server" --dport 443 -j DROP 2>/dev/null; do
        echo "Removed DoH block: $server"
    done
done

# Remove domain string matching rules
domains=("youtube.com" "www.youtube.com")
for domain in "${domains[@]}"; do
    # Remove DNS rules
    while iptables -D OUTPUT -p udp --dport 53 -m string --string "$domain" --algo bm -j DROP 2>/dev/null; do
        echo "Removed DNS UDP rule: $domain"
    done
    while iptables -D OUTPUT -p tcp --dport 53 -m string --string "$domain" --algo bm -j DROP 2>/dev/null; do
        echo "Removed DNS TCP rule: $domain"
    done
    # Remove HTTP/HTTPS rules
    while iptables -D OUTPUT -p tcp --dport 80 -m string --string "$domain" --algo bm -j DROP 2>/dev/null; do
        echo "Removed HTTP rule: $domain"
    done
    while iptables -D OUTPUT -p tcp --dport 443 -m string --string "$domain" --algo bm -j DROP 2>/dev/null; do
        echo "Removed HTTPS rule: $domain"
    done
    # Remove systemd-resolved rules
    while iptables -D OUTPUT -d 127.0.0.53 -p udp --dport 53 -m string --string "$domain" --algo bm -j DROP 2>/dev/null; do
        echo "Removed systemd-resolved UDP rule: $domain"
    done
    while iptables -D OUTPUT -d 127.0.0.53 -p tcp --dport 53 -m string --string "$domain" --algo bm -j DROP 2>/dev/null; do
        echo "Removed systemd-resolved TCP rule: $domain"
    done
done

echo "Keyphy cleanup complete. Remaining rules:"
iptables -L OUTPUT -n