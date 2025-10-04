# snapdir

[![Go Version](https://img.shields.io/badge/Go-1.23.4-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A professional-grade CLI tool for creating and restoring directory snapshots. snapdir captures complete directory structures and file contents into portable JSON snapshots, making it perfect for project templates, backups, and reproducible environments.

## Features

- **Complete Snapshots**: Captures directory structure, file contents, and permissions
- **Smart Filtering**: Automatic `.gitignore` support + custom ignore patterns
- **File Size Protection**: Configurable limits (default 100MB) prevent memory issues
- **Cross-Platform**: Handles path separators correctly on Windows, macOS, and Linux
- **Verbose Mode**: Detailed logging for debugging and monitoring
- **Version Tracking**: Snapshots include version metadata
- **Safety Checks**: Prevents accidental overwrites
- **Comprehensive Testing**: 60.6% test coverage with table-driven tests

## Installation

### From Source

```bash
git clone https://github.com/supperdoggy/snapdir.git
cd snapdir
go build -o snapdir ./cmd/
```

### Using Go Install

```bash
go install github.com/supperdoggy/snapdir/cmd@latest
```

## Quick Start

### Create a Snapshot

```bash
snapdir clone ./myproject snapshot.json
```

### Restore from Snapshot

```bash
snapdir restore snapshot.json ./restored-project
```

## Usage

### Commands

#### `clone` - Create a snapshot

```bash
snapdir clone <source_dir> <output.json> [flags]
```

**Arguments:**
- `source_dir`: Directory to snapshot
- `output.json`: Output JSON file path

**Flags:**
- `-v, --verbose`: Enable verbose logging
- `--ignore <patterns>`: Additional ignore patterns (comma-separated)
- `--version`: Show version information

**Examples:**

```bash
# Basic snapshot
snapdir clone ./myproject snapshot.json

# With verbose output
snapdir clone ./myproject snapshot.json -v

# With custom ignore patterns
snapdir clone ./myproject snapshot.json --ignore "*.tmp,*.cache,dist"

# Combine flags
snapdir clone ./myproject snapshot.json -v --ignore "node_modules,*.log"
```

#### `restore` - Restore from snapshot

```bash
snapdir restore <config.json> <destination_dir> [flags]
```

**Arguments:**
- `config.json`: Snapshot JSON file
- `destination_dir`: Where to restore (must not exist)

**Flags:**
- `-v, --verbose`: Enable verbose logging
- `--version`: Show version information

**Examples:**

```bash
# Basic restore
snapdir restore snapshot.json ./restored

# With verbose output
snapdir restore snapshot.json ./restored -v
```

## How It Works

### .gitignore Support

snapdir automatically respects `.gitignore` patterns in the source directory:

1. Reads `.gitignore` from the source directory root
2. Parses patterns (supports comments and empty lines)
3. Applies patterns during snapshot creation
4. Always excludes `.git` directory

**Example .gitignore:**
```gitignore
# Dependencies
node_modules/
vendor/

# Build outputs
dist/
build/
*.exe

# Logs
*.log
logs/

# Environment
.env
.env.local
```

### Custom Ignore Patterns

Additional patterns can be specified via the `--ignore` flag:

```bash
snapdir clone ./project snapshot.json --ignore "temp,*.bak,cache"
```

Patterns support:
- Wildcards: `*.log`, `*.tmp`
- Directory names: `node_modules`, `.cache`
- Exact filenames: `debug.log`

### Snapshot Format

Snapshots are stored as JSON with the following structure:

```json
{
  "version": "1.0.0",
  "files": [
    {
      "path": "src/main.go",
      "contents": "package main...",
      "is_dir": false,
      "mode": 420
    },
    {
      "path": "src",
      "is_dir": true,
      "mode": 493
    }
  ]
}
```

**Fields:**
- `version`: snapdir version used to create snapshot
- `path`: Relative path (uses forward slashes)
- `contents`: File contents (omitted for directories)
- `is_dir`: Boolean indicating directory
- `mode`: Unix file permissions (octal in decimal)

## Use Cases

### Project Templates

Create reusable project templates:

```bash
# Create template
snapdir clone ./my-template template.json

# Use template for new projects
snapdir restore template.json ./new-project-1
snapdir restore template.json ./new-project-2
```

### Environment Reproduction

Capture and share development environments:

```bash
# Developer A creates snapshot
snapdir clone ./working-env env-snapshot.json

# Developer B reproduces environment
snapdir restore env-snapshot.json ./my-env
```

### Backup and Archival

Quick directory backups:

```bash
# Create timestamped backup
snapdir clone ./important-files backup-$(date +%Y%m%d).json -v
```

### Testing Fixtures

Create test data fixtures:

```bash
# Snapshot test data
snapdir clone ./test-data test-fixture.json

# Restore for each test run
snapdir restore test-fixture.json ./test-temp
```

## Development

### Prerequisites

- Go 1.23.4 or higher

### Building

```bash
go build -o snapdir ./cmd/
```

### Running Tests

```bash
# Run all tests
go test ./cmd/

# Run with verbose output
go test -v ./cmd/

# Run with coverage
go test -cover ./cmd/

# Generate coverage report
go test -coverprofile=coverage.out ./cmd/
go tool cover -html=coverage.out
```

### Project Structure

```
snapdir/
├── cmd/
│   ├── main.go          # Main application logic
│   └── main_test.go     # Comprehensive test suite
├── go.mod               # Go module definition
├── .gitignore          # Git ignore patterns
└── README.md           # This file
```

## Technical Details

### Limits and Constraints

- **Max file size**: 100MB (files larger than this are skipped)
- **Path format**: Uses forward slashes in snapshots (cross-platform)
- **Permissions**: Preserves Unix file permissions (mode)
- **Encoding**: UTF-8 for file contents

### Error Handling

snapdir provides detailed error messages with context:

- Path validation errors
- File access errors
- JSON parsing errors
- Directory creation errors

All errors use Go's error wrapping (`%w`) for proper error chains.

### Safety Features

- **No overwrites**: Restore fails if destination exists
- **Path validation**: Checks for empty and non-existent paths
- **File size limits**: Prevents memory exhaustion
- **Skip on errors**: Invalid patterns logged but don't stop execution

## Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for new functionality
4. Ensure all tests pass (`go test ./...`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with Go's standard library
- Uses `filepath.WalkDir` for efficient directory traversal
- Inspired by modern backup and templating tools

---

**Made with Go** | [Report Bug](https://github.com/supperdoggy/snapdir/issues) | [Request Feature](https://github.com/supperdoggy/snapdir/issues)