# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

zpscan is a command-line information gathering tool written in Go that provides comprehensive reconnaissance capabilities. It consists of multiple scanning modules designed for security assessment and network reconnaissance.

## Core Architecture

### Main Components

- **cmd/**: CLI command handlers using Cobra framework
  - Each scanning module has its own command file (domainscan.go, ipscan.go, webscan.go, etc.)
  - root.go contains common options and initialization logic
- **pkg/**: Core functionality packages
  - Each scanning module has its own package with runner, config, and result handling
  - Shared utilities for HTTP requests, parsing, and validation
- **internal/utils/**: Internal utility functions for file handling, HTTP operations, parsing
- **config/**: Configuration management

### Module Structure

Each scanning module follows a consistent pattern:
- **runner.go**: Main execution logic
- **config.go**: Module-specific configuration
- **result.go**: Result structure and handling
- **parse.go**: Input parsing and validation

### Key Modules

1. **domainscan**: Subdomain enumeration using subfinder and ksubdomain
2. **ipscan**: Port scanning with naabu integration and service fingerprinting
3. **webscan**: Web asset discovery with favicon analysis, title extraction, and fingerprinting
4. **crack**: Password cracking for common services (FTP, SSH, RDP, databases)
5. **dirscan**: Directory and file enumeration
6. **pocscan**: Vulnerability scanning with support for multiple POC formats (Goby, Xray, Nuclei)
7. **expscan**: Exploit framework based on Nuclei

## Development Commands

### Building
```bash
# Build for current platform
go build -o zpscan main.go

# Build for Docker (referenced in Dockerfile)
make docker
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific module
go test ./pkg/crack/...
go test ./pkg/webscan/...

# Run tests with verbose output
go test -v ./pkg/crack/plugins/
```

### Running
```bash
# Show help
./zpscan -h

# Run specific modules
./zpscan domainscan -i example.com
./zpscan ipscan -i 192.168.1.0/24
./zpscan webscan -i http://example.com
./zpscan crack -i 192.168.1.100:22
```

### Dependencies
- Uses Go modules (go.mod/go.sum)
- Key dependencies include projectdiscovery tools (subfinder, naabu, nuclei)
- Custom forks of some dependencies are used (see replace directives in go.mod)

## Configuration

- Main configuration expected in config.yaml (not in repository)
- Resource files expected in resource/ directory (not in repository)
- Configuration files must be downloaded separately as mentioned in README

## Code Conventions

- Uses Go standard project layout
- Cobra framework for CLI structure
- Structured logging with gologger
- Error handling follows Go conventions
- Module-specific result structures for consistent output formatting
- Concurrent processing using goroutines and channels where appropriate

## Important Notes

- This is a security tool designed for authorized penetration testing and security assessment
- Requires external resource files (config.yaml, resource/) to function properly
- Many modules integrate with external tools and APIs
- Cross-platform support with Docker containerization available