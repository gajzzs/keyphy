# Under Development
## Keyphy - System Access Control via External Device Authentication

Keyphy is a Linux CLI application that blocks access to applications, websites, and files/folders until authenticated with an external USB device. It uses the device's UUID and name to generate cryptographic keys for secure authentication.

## Features

- **Application Blocking**: Block specific applications at the system level using D-Bus monitoring
- **Website Blocking**: Block websites using iptables and hosts file modification  
- **File/Folder Blocking**: Block access to files and directories using permission changes and filesystem attributes
- **USB Device Authentication**: Use external USB devices as authentication keys
- **Cryptographic Security**: Generate secure keys from device UUID and name using PBKDF2
- **System-Level Enforcement**: Cannot be bypassed through normal user operations
- **Daemon Service**: Continuous monitoring and automatic blocking/unblocking

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd keyphy

# Install dependencies
make deps

# Build and install
make install

# Install systemd service (optional)
make service
```

## Usage

### Device Management

```bash
# List available USB devices
keyphy device list

# Select authentication device
keyphy device select <device-uuid>
```

### Blocking Operations

```bash
# Block an application
keyphy block app firefox

# Block a website
keyphy block website facebook.com

# Block a file or folder
keyphy block path /home/user/games
```

### Unblocking

```bash
# Remove any blocking rule
keyphy unblock firefox
keyphy unblock facebook.com
keyphy unblock /home/user/games
```

### Service Management

```bash
# Start daemon (requires root)
sudo keyphy service start

# Stop daemon
sudo keyphy service stop

# Check status
keyphy service status
```

### List Current Rules

```bash
# Show all blocked items and configuration
keyphy list
```

## How It Works

1. **Device Authentication**: When you select a USB device, keyphy generates a cryptographic key from the device's UUID and name using PBKDF2 key derivation.

2. **System-Level Blocking**: 
   - Applications are blocked using D-Bus monitoring and process termination
   - Websites are blocked using iptables rules and hosts file entries
   - Files/folders are blocked by removing permissions and setting immutable attributes

3. **Continuous Monitoring**: The daemon service continuously monitors for:
   - New process launches of blocked applications
   - Network connections to blocked domains
   - USB device connection/disconnection events

4. **Automatic Unblocking**: When the authenticated USB device is connected, all blocking rules are temporarily lifted. When disconnected, blocks are re-applied.

## Security Features

- **Cryptographic Authentication**: Uses PBKDF2 with 10,000 iterations for key derivation
- **System-Level Enforcement**: Blocks are enforced at kernel/system level, not user level
- **Multiple Blocking Methods**: Uses redundant blocking mechanisms (iptables + hosts, permissions + immutable attributes)
- **Root Privileges**: Daemon runs with root privileges for system-level access control

## Requirements

- Linux operating system
- Root access for daemon operations
- iptables for network blocking
- systemd for service management (optional)

## Configuration

Configuration is stored in `/etc/keyphy/config.json`:

```json
{
  "blocked_apps": ["firefox", "chrome"],
  "blocked_websites": ["facebook.com", "twitter.com"],
  "blocked_paths": ["/home/user/games", "/opt/steam"],
  "auth_device": "device-uuid-here",
  "service_enabled": true
}
```

## Building from Source

```bash
# Install Go 1.21 or later
# Clone repository
git clone <repository-url>
cd keyphy

# Build
go build -o keyphy ./cmd/keyphy

# Or use make
make build
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Warning

This tool modifies system-level settings including iptables rules, file permissions, and process management. Use with caution and ensure you have proper backups and recovery methods available.
