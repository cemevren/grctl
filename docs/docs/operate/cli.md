---
title: CLI
---

The `grctl` Command Line Interface (CLI) provides tools to manage workflow executions.

## Usage

```bash
grctl [global flags] [command] [options]
```

Use `grctl help` or `grctl [command] --help` for detailed CLI information.

## Global Flags

These flags can be provided to the `grctl` command to control global settings:

- `-l, --log-level string`: Log level (`debug`, `info`, `warn`, `error`). 
- `-s, --server-url string`: grctl server address (e.g. `localhost:4225` or `prod.example.com:4225`). Default is `localhost:4225`.

### Environment Variables

Some global settings can be configured via environment variables:

- `LOG_LEVEL`: Sets the log level. Equivalent to `-l, --log-level`.

## Top-Level Commands

### `grctl version`

Prints out the version, commit, and build date of the `grctl` binary.

## Workflow Commands

The `workflow` subcommand group provides tools to manage, inspect, and interact with workflow runs.

| Command | Description |
| :--- | :--- |
| `start` | Start a new workflow run.<br><br>**Options:**<br>- `--type string` (required): The workflow type.<br>- `--id string`: Workflow ID (auto-generated if empty).<br>- `--input string`: JSON input string.<br>- `--input-file string`: Path to a JSON file. |
| `cancel` | Cancel a running workflow.<br><br>**Options:**<br>- `--id string` (required): Workflow ID.<br>- `--reason string`: The cancellation reason. |
| `list` | Display all workflow runs in an interactive table view (TUI). |
| `history` | Display run event history in an interactive table view (TUI).<br><br>**Options:**<br>- `--id string` (required): Workflow ID.<br>- `--run-id string`: Workflow run ID (defaults to latest run). |

*(Note: For the `start` command, `--input` and `--input-file` are mutually exclusive)*

### Usage Examples

**Start a workflow with inline JSON:**
```bash
grctl workflow start --type OrderFulfillment --input '{"orderId": "12345"}'
```

**Start a workflow using a JSON file:**
```bash
grctl workflow start --type OrderFulfillment --input-file ./payload.json
```

**Cancel a running workflow:**
```bash
grctl workflow cancel --id "wf-abc-123" --reason "Manual intervention required"
```

## Interactive UI (TUI)

!!! info
    The `list` and `history` commands launch an interactive **Terminal User Interface (TUI)**. They are not meant to be piped standard output to other commands (like `grep` or `jq`). Use arrow keys or `j`/`k` to navigate and `q` or `ctrl+c` to exit.
