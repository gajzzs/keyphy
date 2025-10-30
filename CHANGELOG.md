# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2024-01-XX

### Added
- Initial release of Keyphy
- Application blocking via D-Bus monitoring and executable replacement
- Website blocking using iptables and hosts file modification
- File/folder blocking using permission changes and filesystem attributes
- USB device authentication with cryptographic security
- Daemon service for continuous monitoring
- System-level enforcement that cannot be bypassed through normal user operations
- Command-line interface with comprehensive commands
- Systemd service integration

### Security Features
- PBKDF2-based key derivation from device UUID and name
- Cryptographic device authentication
- System-level blocking mechanisms
- Process protection and self-monitoring
- Configuration file protection with immutable attributes

### Commands
- `keyphy add` - Add apps, websites, or file paths to blocking list
- `keyphy unblock` - Remove blocking rules
- `keyphy reset` - Reset system and remove all blocks
- `keyphy list` - List all blocked items
- `keyphy device` - Manage authentication devices
- `keyphy service` - Manage daemon service
- `keyphy lock/unlock` - Control blocking state with device authentication