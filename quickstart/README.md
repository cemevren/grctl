# Ground Control Quickstart Script

Bootstrap script that sets up a ready-to-run grctl project with a single command.

## Usage

```bash
curl -sSL https://get.grctl.dev/quickstart | bash
```

With a custom project name:

```bash
curl -sSL https://get.grctl.dev/quickstart | bash -s -- my-project
```

Pin a specific grctl version:

```bash
grctl_VERSION=v0.2.0 curl -sSL https://get.grctl.dev/quickstart | bash
```

## What it does

1. **Checks prerequisites** — Verifies Python 3.13+, uv, and git are installed. Exits with a clear error message if anything is missing.

2. **Installs the grctl server** — Detects OS (Linux/macOS) and architecture (amd64/arm64), downloads the matching pre-built binary from GitHub releases, and places it in `~/.grctl/bin/`. Skips this step if `grctl` is already on PATH.

3. **Scaffolds the project** — Clones the [grctl_starter](https://github.com/cemevren/grctl_starter) template repo into the target directory and removes the `.git` history so the user starts fresh.

4. **Installs dependencies** — Runs `uv sync` to install the grctl Python SDK and all dependencies.

5. **Prints next steps** — Three commands to get a workflow running:
   - `grctl server start` — start the server
   - `uv run python worker.py` — start the worker
   - `uv run python client.py` — trigger a workflow

## Configuration

| Variable | Default | Description |
|---|---|---|
| `grctl_VERSION` | `latest` | Server binary version to install. Resolved from GitHub releases API. |
| First argument | `my-grctl-project` | Name of the project directory to create. |

## Hosting

The script is designed to be served from `https://get.grctl.dev/quickstart`. This can be a redirect to the raw file on GitHub:

```
https://raw.githubusercontent.com/cemevren/grctl/main/quickstart/install.sh
```
