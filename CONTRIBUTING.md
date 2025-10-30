# Contributing to Keyphy

Thank you for your interest in contributing to Keyphy! This document provides guidelines for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct (to be added).

## How to Contribute

### Reporting Bugs

1. Check if the bug has already been reported in [Issues](https://github.com/gajzzs/keyphy/issues)
2. If not, create a new issue with:
   - Clear description of the bug
   - Steps to reproduce
   - Expected vs actual behavior
   - System information (OS, Go version, etc.)

### Suggesting Features

1. Check existing [Issues](https://github.com/gajzzs/keyphy/issues) and [Discussions](https://github.com/gajzzs/keyphy/discussions)
2. Create a new issue or discussion with:
   - Clear description of the feature
   - Use case and motivation
   - Possible implementation approach

### Development Setup

1. Fork the repository
2. Clone your fork: `git clone https://github.com/yourusername/keyphy.git`
3. Install Go 1.21 or later
4. Install dependencies: `make deps`
5. Build the project: `make build`
6. Run tests: `make test`

### Pull Request Process

1. Create a feature branch: `git checkout -b feature/your-feature-name`
2. Make your changes
3. Add tests for new functionality
4. Ensure all tests pass: `make test`
5. Update documentation if needed
6. Commit with clear messages
7. Push to your fork
8. Create a Pull Request

### Coding Standards

- Follow Go conventions and best practices
- Use `gofmt` for code formatting
- Add comments for exported functions and types
- Write tests for new functionality
- Keep commits atomic and well-described

### Security Considerations

- Be mindful of security implications in system-level operations
- Test thoroughly on different Linux distributions
- Consider edge cases and error handling
- Follow secure coding practices

## Development Guidelines

### Project Structure

```
keyphy/
├── cmd/keyphy/          # Main application entry point
├── internal/            # Internal packages
│   ├── app/            # CLI commands
│   ├── blocker/        # Blocking implementations
│   ├── config/         # Configuration management
│   ├── crypto/         # Cryptographic functions
│   ├── device/         # USB device detection
│   └── service/        # Daemon service
├── .github/workflows/   # CI/CD workflows
└── docs/               # Documentation
```

### Testing

- Write unit tests for all new functions
- Test on multiple Linux distributions
- Include integration tests where appropriate
- Test with different hardware configurations

### Documentation

- Update README.md for user-facing changes
- Add inline code comments
- Update CHANGELOG.md for releases
- Create wiki pages for complex features

## Release Process

1. Update version in appropriate files
2. Update CHANGELOG.md
3. Create and push a version tag
4. GitHub Actions will automatically create a release

## Questions?

Feel free to ask questions in [Discussions](https://github.com/gajzzs/keyphy/discussions) or create an issue.