# lazy-cli

A lightweight interactive CLI tool that converts natural language into shell commands using a local LLM via [llama.cpp](https://github.com/ggml-org/llama.cpp).

Designed to run on low-resource hardware (Raspberry Pi, old laptops) with small GGUF models. No cloud, no API keys, no GUI — just a terminal.

Supports **Linux** and **macOS** on both **AMD64** and **ARM64**.

---

## Table of Contents

- [Installation](#installation)
  - [Quick Install](#quick-install-recommended)
  - [Manual Download](#manual-download)
  - [Build from Source](#build-from-source)
- [Quick Start](#quick-start)
- [CLI Commands](#cli-commands)
- [Interactive Mode](#interactive-mode)
  - [Example Session](#example-session)
- [Internal Commands](#internal-commands)
  - [`§config` — Configuration Management](#config--configuration-management)
  - [`§logs` — Log Viewer](#logs--log-viewer)
- [Safety System](#safety-system)
  - [Safety Levels](#safety-levels)
  - [Safety Modes](#safety-modes)
  - [Blocked Patterns](#blocked-patterns-always-refused)
  - [Dangerous Patterns](#dangerous-patterns)
  - [Caution Patterns](#caution-patterns)
- [Logging](#logging)
- [Configuration File](#configuration-file)
- [Project Structure](#project-structure)
- [LLM Server Management](#llm-server-management)
- [Troubleshooting](#troubleshooting)
- [License](#license)

---

## Installation

### Quick Install (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/bulkinglb/lazy-cli/main/install.sh | sh
```

Or with `wget`:

```bash
wget -qO- https://raw.githubusercontent.com/bulkinglb/lazy-cli/main/install.sh | sh
```

This automatically detects your OS and architecture, downloads and extracts the correct binary, and installs it to `/usr/local/bin` (falls back to `~/.local/bin` if sudo is unavailable).

### Manual Download

Download the archive for your platform from the [Releases](https://github.com/bulkinglb/lazy-cli/releases) page:

| Platform | Archive |
|---|---|
| Linux (AMD64) | `lazy-cli-linux-amd64.tar.gz` |
| Linux (ARM64 / Raspberry Pi) | `lazy-cli-linux-arm64.tar.gz` |
| macOS (Intel) | `lazy-cli-macos-amd64.tar.gz` |
| macOS (Apple Silicon) | `lazy-cli-macos-arm64.tar.gz` |

```bash
# Example: Linux AMD64
curl -fsSL https://github.com/bulkinglb/lazy-cli/releases/latest/download/lazy-cli-linux-amd64.tar.gz | tar -xz
sudo mv lazy-cli /usr/local/bin/
```

### Build from Source

Requires Go 1.21+.

```bash
git clone https://github.com/bulkinglb/lazy-cli.git
cd lazy-cli
make build        # builds for current platform → ./lazy-cli
make install      # installs to ~/go/bin/lazy-cli
```

To cross-compile for all platforms:

```bash
make all
# Produces:
#   lazy-cli-linux-amd64
#   lazy-cli-linux-arm64
#   lazy-cli-macos-amd64
#   lazy-cli-macos-arm64
```

---

## Quick Start

```bash
# 1. Setup (one time) — auto-downloads llama-server and Gemma 3 1B if not already installed
lazy-cli setup

# 2. Run
lazy-cli
```

Or provide your own llama-server and model:

```bash
lazy-cli setup --llama-server /path/to/llama-server --model /path/to/model.gguf
```

After setup, paths are saved to `~/.lazy-cli/config.json`. Future runs need no flags.

---

## CLI Commands

### `lazy-cli` — Start Interactive Mode

Launches the REPL. Starts the LLM server automatically, stops it on exit.

```
lazy-cli [--model PATH] [--server PATH] [--port PORT]
```

| Flag | Description | Default |
|---|---|---|
| `--model` | Path to GGUF model file | from .env or config |
| `--server` | Path to llama-server binary | from .env or config |
| `--port` | Port for llama-server | from config (8085) |

**Resolution order** for model/server paths: CLI flag > environment variable > config file.

### `lazy-cli setup` — First-Time Configuration

Validates paths, saves them to config, and test-starts the server. If `--llama-server` or `--model` are not provided, **automatically downloads** the latest llama.cpp release and Gemma 3 1B.

```
lazy-cli setup [--llama-server PATH] [--model PATH] [--port PORT] [--skip-test]
```

What it does:
1. Creates `~/.lazy-cli/` and `~/.lazy-cli/logs/` if missing
2. If no llama-server is configured, checks PATH — then auto-downloads from llama.cpp releases to `~/.lazy-cli/bin/`
3. If no model is configured, auto-downloads `gemma-3-1b-it-Q4_K_M.gguf` (~800 MB) to `~/.lazy-cli/models/`
4. Validates the llama-server binary exists and is executable
5. Validates the model file exists and has valid GGUF magic bytes
6. Checks the port is available
7. Saves everything to `~/.lazy-cli/config.json`
8. Test-starts the server and runs a health check (unless `--skip-test`)

### `lazy-cli status` — Show Current State

```
$ lazy-cli status

=== lazy-cli status ===

  Config file:   /home/user/.lazy-cli/config.json (exists)
  Server path:   /home/user/llama.cpp/build/bin/llama-server (found)
  Model path:    /home/user/models/gemma-3-1b-it-Q4_K_M.gguf (valid GGUF)
  Port:          8085 (available)
  Mode:          normal
  ...
  Setup: VALID - ready to run
```

### `lazy-cli doctor` — Run Diagnostic Checks

Checks everything needed to run — config, directories, binary, model, port, and does a full server launch + API health test.

```
$ lazy-cli doctor

=== lazy-cli doctor ===

  [OK] config: /home/user/.lazy-cli/config.json
  [OK] directories: OK
  [OK] llama-server: /home/user/llama.cpp/build/bin/llama-server
  [OK] model file: /home/user/models/gemma-3-1b-it-Q4_K_M.gguf (768 MB)
  [OK] port: 8085 (available)
  [OK] server launch + API: server started, health check passed

All checks passed. Ready to run.
```

### `lazy-cli help` — Show Usage

### `lazy-cli version` — Show Version

---

## Interactive Mode

Once running, the REPL accepts three types of input:

| Input | Type | Example |
|---|---|---|
| Plain text | AI command generation | `install docker` |
| `!command` | Direct shell execution | `!ls -la` |
| `§command` | Internal REPL command | `§help` |

The internal command prefix is configurable (default `§`).

### Example Session

```
lazy-cli> install docker
Command: sudo apt install docker.io
⚡ Caution: runs as root
Execute? [Y/n]: y
---
...
---
✓ Command completed successfully

lazy-cli> !uname -a
Execute? [Y/n]:
---
Linux myhost 6.6.87 ...
---
✓ Command completed successfully

lazy-cli> §status
=== Runtime Status ===
  Mode:            normal
  Port:            8085
  ...
```

---

## Internal Commands

All internal commands start with the configured prefix (default `§`).

| Command | Description |
|---|---|
| `§help` | Show all available commands |
| `§status` | Show runtime status (mode, port, server, paths, CWD) |
| `§config` | Show or change configuration |
| `§history` | Show command history for this session |
| `§logs` | List log files or view a specific session |
| `§clearlogs` | Delete all log files (with confirmation) |
| `§exit` | Exit the CLI (also: `§quit`) |

### `§config` — Configuration Management

```
§config                     Show all configuration
§config <key>               Show value for a key
§config <key> <value>       Set a value (persists to disk)
```

| Key | Values | Description |
|---|---|---|
| `mode` | `ultra-safe`, `normal`, `yolo` | Safety mode |
| `port` | `1`-`65535` | llama-server port |
| `prefix` | 1-4 characters | Internal command prefix |
| `logging` | `on`/`off` | Enable or disable logging |
| `logpath` | path | Log directory |
| `model` | path | GGUF model file path |
| `server` | path | llama-server binary path |
| `alias` | `<name> <path>` | Path aliases (see below) |

**Path aliases** give named shortcuts to directories:

```
§config alias projects /home/user/Projects
§config alias docs /home/user/Documents
§config alias rm projects              # remove an alias
§config alias                          # list all aliases
```

### `§logs` — Log Viewer

```
§logs           List all session log files
§logs 3         View entries from session #3
§logs all       View all entries across all sessions
```

Output shows session files with entry counts:

```
=== Log sessions (/home/user/.lazy-cli/logs/) ===
    [1]  2026-03-28 09:32:08  (5 entries)
  * [2]  2026-03-28 10:15:22  (0 entries)

Use §logs <N> to view a session. * = current session
```

---

## Safety System

Every command — whether generated by AI or entered directly with `!` — passes through the safety checker before execution.

### Safety Levels

| Level | Behavior |
|---|---|
| **Safe** | Prompt `[Y/n]` (auto-execute in yolo mode) |
| **Caution** | Prompt `[Y/n]` with warning |
| **Dangerous** | Prompt `[y/N]` with strong warning (default: no) |
| **Blocked** | Refused outright, cannot execute |

### Safety Modes

| Mode | Effect |
|---|---|
| `ultra-safe` | Everything requires confirmation. Caution → Dangerous. |
| `normal` | Default behavior. |
| `yolo` | Only Blocked commands are stopped. Safe commands auto-execute. Dangerous → Caution. |

Change with `§config mode yolo`.

### Blocked Patterns (always refused)

| Pattern | Reason |
|---|---|
| `rm -rf /` | Removes root filesystem |
| `mkfs.*` | Formats filesystem |
| `dd ... of=/dev/sdX` | Overwrites disk device |
| Fork bombs | Fork bomb |
| `chmod 777 /` | Opens root permissions |
| `chown ... /` | Changes root ownership |

### Dangerous Patterns

| Pattern | Reason |
|---|---|
| `sudo rm` | Sudo remove |
| `rm -rf` | Recursive/force delete |
| `> /etc/...` | Overwrites system config |
| `curl ... \| sh` | Pipes remote script to shell |
| `shutdown`, `reboot` | System shutdown/reboot |
| `systemctl stop/disable` | Disables system service |

### Caution Patterns

| Pattern | Reason |
|---|---|
| `sudo ...` | Runs as root |
| `rm`, `mv` | Deletes/moves files |
| `chmod`, `chown` | Changes permissions |
| `apt remove`, `pip uninstall` | Removes packages |

---

## Logging

Every AI interaction and direct command is logged as JSONL to `~/.lazy-cli/logs/`.

Each session creates a file named `session_<timestamp>.jsonl`.

### Log Entry Types

| Type | When |
|---|---|
| `interaction` | AI-generated command was executed |
| `direct` | `!command` was executed |
| `blocked` | Command was refused by safety checker |
| `error` | Internal error (e.g. LLM failure) |

### Log Entry Fields

```json
{
  "ts": "2026-03-28T10:05:10Z",
  "type": "interaction",
  "input": "install docker",
  "command": "sudo apt install docker.io",
  "safety": "caution",
  "safety_reason": "runs as root",
  "exit_code": 0,
  "duration_ms": 3420
}
```

---

## Configuration File

Stored at `~/.lazy-cli/config.json`.

```json
{
  "mode": "normal",
  "port": 8085,
  "prefix": "§",
  "log_enabled": true,
  "log_path": "/home/user/.lazy-cli/logs",
  "model_path": "/home/user/.lazy-cli/models/gemma-3-1b-it-Q4_K_M.gguf",
  "server_path": "/home/user/.lazy-cli/bin/llama-server",
  "path_aliases": {
    "projects": "/home/user/Projects"
  }
}
```

The config is created automatically on first run. Edit it with `§config` inside the REPL or by hand.

---

## Project Structure

```
lazy-cli/
├── main.go              Entry point, subcommand routing
├── config/
│   └── config.go        Persistent JSON configuration
├── doctor/
│   ├── check.go         Validation helpers (file, binary, GGUF, port)
│   ├── doctor.go        'doctor' subcommand
│   └── status.go        'status' subcommand
├── executor/
│   └── executor.go      Shell command execution via sh -c
├── llm/
│   ├── client.go        LLM HTTP client + prompt construction
│   └── server.go        llama-server process lifecycle
├── logger/
│   ├── logger.go        JSONL session logging
│   └── reader.go        Log file reading + session listing
├── repl/
│   ├── command.go       Command types and registry
│   ├── commands.go      Built-in command implementations
│   ├── parser.go        Input parsing (AI / direct / internal)
│   └── repl.go          Interactive REPL loop
├── safety/
│   └── safety.go        Regex-based command safety classification
├── setup/
│   ├── setup.go         'setup' subcommand
│   └── download.go      Auto-download logic for llama.cpp and Gemma model
├── go.mod
├── Makefile             Build and release automation
├── install.sh           One-line installer script
└── .gitignore
```

**Zero external dependencies.** Built entirely on the Go standard library.

---

## LLM Server Management

The CLI manages `llama-server` automatically:

- **Starts on demand** when the interactive CLI launches
- **Reuses** an already-running instance if one is detected via `/health`
- **Stops cleanly** on exit (SIGTERM → 5s grace → SIGKILL)
- Runs in a separate process group for clean shutdown

Server configuration:
- Host: `127.0.0.1` (localhost only)
- Context size: 2048 tokens
- Timeout: 30 seconds for startup, 120 seconds for LLM responses

---

## Troubleshooting

### Server won't start
```bash
lazy-cli doctor
```
This will pinpoint the exact issue — missing binary, bad model, port conflict.

### "model is required" error
Run setup — it will auto-download everything:
```bash
lazy-cli setup
# or manually:
lazy-cli setup --llama-server /path/to/llama-server --model /path/to/model.gguf
```

### Port already in use
```bash
lazy-cli setup --port 8090
# or inside the REPL:
§config port 8090
```

### Reset configuration
Delete `~/.lazy-cli/config.json` — it will be recreated with defaults on next run.

---

## License

MIT
