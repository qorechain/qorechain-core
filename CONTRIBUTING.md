# Contributing to QoreChain

Thank you for your interest in contributing to QoreChain! This document provides guidelines for contributing to the project.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/qorechain-core.git`
3. Create a feature branch: `git checkout -b feature/your-feature`
4. Make your changes
5. Run tests: `CGO_ENABLED=1 go test ./...`
6. Submit a pull request

## Development Setup

### Prerequisites

- Go 1.25 or later
- CGO enabled (`CGO_ENABLED=1`)
- Docker and Docker Compose (for integration testing)
- Rust toolchain (only if modifying PQC libraries)

### Building

```bash
CGO_ENABLED=1 go build -o qorechaind ./cmd/qorechaind/
```

### Running Tests

```bash
# Unit tests
CGO_ENABLED=1 go test ./...

# Integration tests
docker compose -f docker-compose.test.yml up --build
```

## Code Style

- Follow standard Go conventions (`gofmt`, `golint`)
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions focused and small

## Pull Request Guidelines

1. **One feature per PR** — Keep PRs focused on a single change
2. **Write tests** — All new code should have tests
3. **Update docs** — If your change affects APIs or user-facing behavior, update documentation
4. **Pass CI** — All CI checks must pass before merging
5. **Sign commits** — Use `git commit -s` to sign your commits (DCO)

## Reporting Issues

- Use GitHub Issues with the appropriate template
- Include steps to reproduce for bugs
- Include expected vs actual behavior
- Attach relevant logs

## Security

For security vulnerabilities, please see [SECURITY.md](SECURITY.md). Do NOT create public issues for security bugs.

## License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.
