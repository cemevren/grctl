# Ground Control CLI

Unified command-line interface for the grctl workflow execution engine.

## Structure

```
cmd/grctl/
├── main.go              # CLI entry point
└── commands/
    ├── root.go          # Root command and version
    └── server.go        # Server management commands
```

## Building

```bash
# From server/ directory
make build

# Or directly with go
go build -o bin/grctl ./cmd/grctl
```

## Usage

### Server Commands

Start the grctl server with embedded NATS:

```bash
grctl server start
grctl server start --config /path/to/config.yaml
grctl server start --log-level debug
```

### Version

```bash
grctl version
```

## Development

The CLI uses [cobra](https://github.com/spf13/cobra) for command structure and integrates with the server logic via `internal/srv` package.

### Adding New Commands

1. Create a new file in `cmd/grctl/commands/`
2. Define your cobra command
3. Register it in `commands/root.go` init function

Example:

```go
// commands/workflow.go
var workflowCmd = &cobra.Command{
    Use:   "workflow",
    Short: "Workflow management commands",
}

// In commands/root.go init()
rootCmd.AddCommand(workflowCmd)
```
