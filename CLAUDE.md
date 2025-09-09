# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`procmate` is a Go-based command-line process management tool that serves as a modern replacement for shell scripts. It manages and monitors long-running processes through centralized YAML configuration files, featuring lifecycle management, daemon monitoring, and systemd integration.

## Build Commands

### Building the Project
```bash
# Static build for Linux amd64 (production)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o procmate main.go

# Development build
go build -o procmate main.go

# Cross-platform builds with GoReleaser
goreleaser build --snapshot --clean
```

### Running the Application
```bash
# Run with default config search paths
./procmate [command]

# Run with specific config file
./procmate --config /path/to/config.yaml [command]

# Available commands: start, stop, status, restart, watch, logs
```

### Testing & Quality
```bash
# Run tests (if any exist)
go test ./...

# Format code
go fmt ./...

# Lint (requires golangci-lint)
golangci-lint run

# Update dependencies
go mod tidy
```

## Architecture

### Code Structure
- `main.go`: Entry point, delegates to `cmd.Execute()`
- `cmd/`: Cobra CLI commands (`root.go`, `start.go`, `stop.go`, `status.go`, `watch.go`, `log.go`)
- `pkg/config/`: Configuration management with Viper, handles YAML parsing and includes
- `pkg/process/`: Core process management logic (start, stop, status, dependencies, logging)

### Configuration System
The tool uses a hierarchical configuration system:

1. **Main config** (`config.yaml`): Global settings and include directives
2. **Service configs** (`conf.d/*.yaml`): Individual process definitions loaded via glob patterns
3. **Config search paths**: Current dir → `/etc/procmate/` → `~/.config/procmate/`

### Key Components

#### Configuration Loading (`pkg/config/config.go`)
- Uses Viper for YAML parsing
- Supports `include` directive with glob patterns for modular service definitions
- Handles duplicate process names with warnings (last definition wins)
- Global `Cfg` variable holds loaded configuration

#### Process Management (`pkg/process/`)
- `start.go`: Process launching with environment variables and log redirection
- `stop.go`: Process termination with timeout handling
- `status.go`: Process state checking (running/ready status)
- `dependency.go`: Dependency resolution and ordering
- `log.go`: Log file management with lumberjack rotation
- `paths.go`: Runtime file path management (PID files, logs)

#### CLI Commands (`cmd/`)
- Built with Cobra framework
- Persistent `--config` flag for all commands
- Commands: `start`, `stop`, `status`, `restart`, `watch` (daemon), `logs`

### Process Lifecycle
1. **Configuration**: Processes defined in YAML with dependencies, timeouts, environment
2. **Dependency Resolution**: Automatic startup ordering based on `depends_on`
3. **Execution**: Processes run via `bash -c` with custom working directory and environment
4. **Monitoring**: PID file tracking and port-based readiness checks
5. **Logging**: Automatic stdout/stderr capture with rotation via lumberjack

### Key Features
- **Systemd Integration**: Designed to run as a systemd service via `install.sh`
- **Log Rotation**: Configurable log management (size, age, compression)
- **Process Dependencies**: Automatic ordering and dependency checking
- **Daemon Mode**: `watch` command for continuous monitoring and restart
- **Static Binary**: CGO-disabled builds for portability across Linux systems

## Configuration Schema

### Global Settings
```yaml
settings:
  runtime_dir: /tmp/procmate          # PID and log file location
  default_start_timeout_sec: 60       # Default process start timeout
  default_stop_timeout_sec: 10        # Default process stop timeout
  watch_interval_sec: 10              # Daemon polling interval
  log_options:                        # Log rotation settings
    max_size_mb: 10000
    max_backups: 10
    max_age_days: 30
    compress: true
    localTime: true
```

### Process Definition
```yaml
processes:
  - name: service-name                # Unique identifier
    group: group-name                 # Logical grouping
    command: "executable with args"   # Shell command to run
    workdir: "/path/to/workdir"       # Working directory
    port: 8080                        # Port for readiness checks
    enabled: true                     # Whether to include in bulk operations
    depends_on: [other-service]       # Process dependencies
    environment:                      # Environment variables
      KEY: value
    start_timeout_sec: 30             # Override default timeout
    stop_timeout_sec: 5               # Override default timeout
```

## Development Guidelines

- Use existing error handling patterns from the codebase
- Follow the established package structure (`cmd/` for CLI, `pkg/` for logic)
- Maintain configuration compatibility when making changes
- Consider systemd integration when modifying daemon behavior
- Use the global `config.Cfg` for accessing configuration throughout the codebase
- Process management functions should handle both single processes and bulk operations