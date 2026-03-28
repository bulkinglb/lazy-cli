# lazy-cli

A lightweight interactive CLI tool that converts natural language into shell commands using a local LLM via [llama.cpp](https://github.com/ggml-org/llama.cpp).

Designed to run on low-resource hardware (Raspberry Pi, old laptops) with small GGUF models. No cloud, no API keys, no GUI — just a terminal.

Supports **Linux** and **macOS** on both **AMD64** and **ARM64**.

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

This automatically detects your OS and architecture, downloads the correct binary, and installs it to `/usr/local/bin`.

### Manual Download

Download the binary for your platform from the [Releases](https://github.com/bulkinglb/lazy-cli/releases) page:

| Platform | Binary |
|---|---|
| Linux (AMD64) | `lazy-cli-linux-amd64` |
| Linux (ARM64 / Raspberry Pi) | `lazy-cli-linux-arm64` |
| macOS (Intel) | `lazy-cli-macos-amd64` |
| macOS (Apple Silicon) | `lazy-cli-macos-arm64` |

```bash
# Example: Linux AMD64
curl -fsSL https://github.com/bulkinglb/lazy-cli/releases/latest/download/lazy-cli-linux-amd64 -o lazy-cli
chmod +x lazy-cli
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
# 1. Setup (one time) — provide your llama-server and a GGUF model
lazy-cli setup \
  --llama-server /path/to/llama-server \
  --model /path/to/model.gguf

# 2. Run
lazy-cli
```

After setup, paths are saved to `~/.lazy-cli/config.json`. Future runs need no flags.

---

## Requirements

- A [llama.cpp](https://github.com/ggml-org/llama.cpp) `llama-server` binary
- A GGUF model file (e.g. `gemma-3-1b-it-Q4_K_M.gguf`, `phi-2-Q4_K_M.gguf`)
- Linux or macOS

---

## CLI Commands

### `lazy-ai` — Start Interactive Mode

Launches the REPL. Starts the LLM server automatically, stops it on exit.

```
lazy-ai [--model PATH] [--server PATH] [--port PORT]
```

| Flag | Description | Default |
|---|---|---|
| `--model` | Path to GGUF model file | from .env or config |
| `--server` | Path to llama-server binary | from .env or config |
| `--port` | Port for llama-server | from config (8085) |

**Resolution order** for model/server paths: CLI flag > environment variable > config file.

### `lazy-ai setup` — First-Time Configuration

Validates paths, saves them to config, and test-starts the server.

```
lazy-ai setup --llama-server PATH --model PATH [--port PORT] [--skip-test]
```

What it does:
1. Creates `~/.lazy-cli/` and `~/.lazy-cli/logs/` if missing
2. Validates the llama-server binary exists and is executable
3. Validates the model file exists and has valid GGUF magic bytes
4. Checks the port is available
5. Saves everything to `~/.lazy-cli/config.json`
6. Test-starts the server and runs a health check (unless `--skip-test`)

### `lazy-ai status` — Show Current State

```
$ lazy-ai status

=== lazy-cli status ===

  Config file:   /home/user/.lazy-cli/config.json (exists)
  Server path:   /home/user/llama.cpp/build/bin/llama-server (found)
  Model path:    /home/user/models/gemma-3-1b-it-Q4_K_M.gguf (valid GGUF)
  Port:          8085 (available)
  Mode:          normal
  ...
  Setup: VALID - ready to run
```

### `lazy-ai doctor` — Run Diagnostic Checks

Checks everything needed to run — config, directories, binary, model, port, and does a full server launch + API health test.

```
$ lazy-ai doctor

=== lazy-cli doctor ===

  [OK] config: /home/user/.lazy-cli/config.json
  [OK] directories: OK
  [OK] llama-server: /home/user/llama.cpp/build/bin/llama-server
  [OK] model file: /home/user/models/gemma-3-1b-it-Q4_K_M.gguf (768 MB)
  [OK] port: 8085 (available)
  [OK] server launch + API: server started, health check passed

All checks passed. Ready to run.
```

### `lazy-ai help` — Show Usage

### `lazy-ai version` — Show Version

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

All internal commands start with the configured prefix (default `%&`).

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
  "prefix": "%&",
  "log_enabled": true,
  "log_path": "/home/user/.lazy-cli/logs",
  "model_path": "/path/to/model.gguf",
  "server_path": "/path/to/llama-server",
  "path_aliases": {
    "projects": "/home/user/Projects"
  }
}
```

The config is created automatically on first run. Edit it with `§config` inside the REPL or by hand.

---

## Environment Variables

Optional — used as fallbacks when CLI flags are not provided.

| Variable | Purpose |
|---|---|
| `LLAMA_MODEL_PATH` | Path to GGUF model file |
| `LLAMA_SERVER_PATH` | Path to llama-server binary |

Can be set in a `.env` file in the working directory (loaded automatically).

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
│   └── setup.go         'setup' subcommand
├── go.mod
├── Makefile             Build and release automation
├── install.sh           One-line installer script
├── .env                 Local environment (gitignored)
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
lazy-ai doctor
```
This will pinpoint the exact issue — missing binary, bad model, port conflict.

### "model is required" error
Run setup to save paths permanently:
```bash
lazy-ai setup --llama-server /path/to/llama-server --model /path/to/model.gguf
```

### Port already in use
```bash
lazy-ai setup --port 8090
# or inside the REPL:
§config port 8090
```

### Reset configuration
Delete `~/.lazy-cli/config.json` — it will be recreated with defaults on next run.

---

## License

MIT
