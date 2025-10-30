# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in Keyphy, please report it responsibly:

1. **Do not** create a public GitHub issue for security vulnerabilities
2. Email security reports to: [security@keyphy.dev] (replace with actual email)
3. Include detailed information about the vulnerability
4. Provide steps to reproduce if possible

## Security Features

Keyphy implements several security measures:

- **Cryptographic Authentication**: Uses PBKDF2 for secure key derivation
- **System-Level Enforcement**: Cannot be bypassed through normal user operations
- **Process Protection**: Self-monitoring and protection mechanisms
- **Configuration Protection**: Immutable file attributes prevent tampering
- **Root Privilege Requirements**: Critical operations require root access

## Security Considerations

- Keyphy requires root privileges for system-level blocking
- Authentication device should be kept secure
- Regular security updates are recommended
- Monitor system logs for unusual activity

## Known Limitations

- Requires Linux with iptables support
- Needs root access for full functionality
- May conflict with other security software
- Advanced users with root access can potentially bypass protections