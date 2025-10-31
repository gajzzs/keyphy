# Changelog

All notable changes to this project will be documented in this file.

## [1.0.3]

### Added
- Integrated `github.com/shirou/gopsutil/v3` for comprehensive system monitoring
- Enhanced physical device authentication with anti-bypass measures
- Cryptographic integrity protection with SHA256 hashes of original executables
- Device fingerprinting to prevent USB device cloning attacks
- Session-based authentication with tamper-evident tokens
- Continuous integrity monitoring with automatic re-enforcement
- Real-time system resource monitoring (CPU, memory, disk usage)
- Suspicious activity detection for bypass attempts
- Comprehensive status command with system information display

### Security
- Added device fingerprinting to detect cloning attempts
- Implemented cryptographic session tokens with time-limited validity
- Enhanced tamper detection through file integrity monitoring
- Immutable backup files with `chattr +i` attribute (Linux)
- Real-time monitoring for suspicious processes (chattr, xattr, iptables, pfctl)
- Automatic block re-enforcement when tampering detected

### Enhanced
- More reliable process termination using gopsutil `Terminate()` and `Kill()` methods
- Better process discovery with both name and command line matching
- Cross-platform system monitoring capabilities
- Enhanced daemon with system activity monitoring
- Improved app blocker with integrity verification

### Changed
- Removed emoji characters from all print statements for cleaner output
- Enhanced authentication validation with comprehensive status reporting
- Improved system monitoring with configurable check intervals

## [1.0.2]

### Added
- Service watchdog functionality to auto-restart if service is killed manually
- Automatic sudo elevation for all privileged commands (no need to manually type sudo)
- Device authentication requirement for service stop operations
- Service installation check before allowing start operations
- Cross-platform IPv6 blocking support with proper hosts file entries
- Enhanced error handling with hex validation for device keys

### Security
- Fixed command injection vulnerability in service management (CWE-95)
- Added device validation for all service control operations
- Improved file permissions for PID files (CWE-276)
- Enhanced daemon signal handling with double authentication

### Fixed
- macOS device detection with proper plist XML parsing instead of JSON
- IPv6 hosts file syntax errors (removed invalid `:: domain` entries)
- Service configuration conflicts between manual plist and kardianos/service
- Inconsistent error handling across network operations
- Missing error propagation in critical blocking operations

### Changed
- Service stop now requires authentication device to prevent bypass
- All service commands automatically elevate to root when needed
- Improved service status reporting with readable status strings
- Enhanced network blocking with both IPv4 and IPv6 iptables/pfctl rules

## [1.0.1]

### Fixed
- Fixed daemon process monitoring continuing to kill processes after unlock signal
- Added `blocksActive` state tracking to prevent unnecessary process termination
- Resolved issue where applications like htop were killed even when blocks were disabled
- Improved daemon state management for lock/unlock operations

## [1.0.0]

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