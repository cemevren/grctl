#!/usr/bin/env bash
#
# Ground Control Quickstart
# Usage: curl -sSL https://get.grctl.dev/quickstart | bash
#
set -euo pipefail

# --- Configuration -----------------------------------------------------------

grctl_VERSION="${grctl_VERSION:-latest}"
grctl_REPO="cemevren/grctl"
STARTER_REPO="cemevren/grctl_starter"
PROJECT_DIR="${1:-my-grctl-project}"

# --- Helpers -----------------------------------------------------------------

info()  { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
warn()  { printf '\033[1;33mWARN:\033[0m %s\n' "$*"; }
error() { printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2; exit 1; }

command_exists() { command -v "$1" >/dev/null 2>&1; }

detect_platform() {
    local os arch

    case "$(uname -s)" in
        Linux*)  os="linux" ;;
        Darwin*) os="darwin" ;;
        *)       error "Unsupported OS: $(uname -s)" ;;
    esac

    case "$(uname -m)" in
        x86_64|amd64)  arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *)             error "Unsupported architecture: $(uname -m)" ;;
    esac

    echo "${os}_${arch}"
}

# --- Preflight checks --------------------------------------------------------

check_prerequisites() {
    info "Checking prerequisites..."

    if ! command_exists python3; then
        error "Python 3 is required. Install it from https://python.org"
    fi

    local py_version
    py_version=$(python3 -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')
    local py_major py_minor
    py_major=$(echo "$py_version" | cut -d. -f1)
    py_minor=$(echo "$py_version" | cut -d. -f2)

    if [ "$py_major" -lt 3 ] || { [ "$py_major" -eq 3 ] && [ "$py_minor" -lt 13 ]; }; then
        error "Python 3.13+ is required (found $py_version). Update from https://python.org"
    fi
    info "Python $py_version"

    if ! command_exists uv; then
        error "uv is required. Install it: curl -LsSf https://astral.sh/uv/install.sh | sh"
    fi
    info "uv $(uv --version | awk '{print $2}')"

    if ! command_exists git; then
        error "git is required."
    fi
}

# --- Download grctl server binary ----------------------------------------------

resolve_version() {
    if [ "$grctl_VERSION" = "latest" ]; then
        grctl_VERSION=$(curl -sSL "https://api.github.com/repos/${grctl_REPO}/releases/latest" \
            | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')
        if [ -z "$grctl_VERSION" ]; then
            error "Could not determine latest grctl version. Set grctl_VERSION explicitly."
        fi
    fi
    info "grctl version: $grctl_VERSION"
}

install_grctl_server() {
    if command_exists grctl; then
        local existing
        existing=$(grctl --version 2>/dev/null || echo "unknown")
        info "grctl server already installed ($existing). Skipping download."
        return
    fi

    local platform
    platform=$(detect_platform)

    local install_dir="${HOME}/.grctl/bin"
    mkdir -p "$install_dir"

    local download_url="https://github.com/${grctl_REPO}/releases/download/${grctl_VERSION}/grctl_${platform}"
    info "Downloading grctl server from ${download_url}..."

    if ! curl -fsSL -o "${install_dir}/grctl" "$download_url"; then
        error "Failed to download grctl server. Check the version and URL."
    fi

    chmod +x "${install_dir}/grctl"

    # Add to PATH hint
    if [[ ":$PATH:" != *":${install_dir}:"* ]]; then
        warn "${install_dir} is not in your PATH."
        warn "Add it with: export PATH=\"${install_dir}:\$PATH\""
        warn "Or add that line to your shell profile (~/.bashrc, ~/.zshrc, etc.)"
    fi

    info "grctl server installed to ${install_dir}/grctl"
}

# --- Scaffold project --------------------------------------------------------

scaffold_project() {
    if [ -d "$PROJECT_DIR" ]; then
        error "Directory '$PROJECT_DIR' already exists. Choose a different name or remove it."
    fi

    info "Creating project in ./${PROJECT_DIR}..."

    git clone --depth 1 "https://github.com/${STARTER_REPO}.git" "$PROJECT_DIR"
    rm -rf "${PROJECT_DIR}/.git"

    cd "$PROJECT_DIR"

    info "Installing dependencies..."
    uv sync
}

# --- Done! -------------------------------------------------------------------

print_next_steps() {
    printf '\n'
    info "Project ready! Next steps:"
    printf '\n'
    printf '  \033[1m1.\033[0m Start the grctl server (in a separate terminal):\n'
    printf '     \033[36mgrctl server start\033[0m\n\n'
    printf '  \033[1m2.\033[0m Start the worker:\n'
    printf '     \033[36mcd %s && uv run python worker.py\033[0m\n\n' "$PROJECT_DIR"
    printf '  \033[1m3.\033[0m Run a workflow (in another terminal):\n'
    printf '     \033[36mcd %s && uv run python client.py\033[0m\n\n' "$PROJECT_DIR"
}

# --- Main --------------------------------------------------------------------

main() {
    printf '\n\033[1mgrctl Quickstart\033[0m\n\n'

    check_prerequisites
    resolve_version
    install_grctl_server
    scaffold_project
    print_next_steps
}

main
