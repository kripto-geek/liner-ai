# liner

> AI-powered, OS-aware terminal command assistant — zero dependencies, single binary.

Describe what you want in plain English, and **liner** returns the exact shell command(s) for your operating system, architecture, shell, and distro.

```
$ liner merge this branch into main
  [1] git fetch origin
  [2] git merge origin/main

  ℹ  Make sure you have no uncommitted changes before merging.

  run [Y/n]?
```

## Features

- 🔍 **Auto-detects OS, arch, shell, and distro** on first run — persists to config so the AI always knows your environment.
- 🤖 **Multiple AI providers** — OpenAI, OpenRouter, Kilo, or local Ollama (no API key needed).
- 📋 **Copy-paste ready** — commands are raw shell lines, never wrapped in markdown.
- ▶️ **One-key execution** — press Enter to run all commands sequentially (only when the AI is confident).
- 🛡️ **Safety notes** — destructive or interactive commands are flagged in the response.
- 🪶 **Zero Go dependencies** — pure stdlib, single binary, ~10 MB.
- 🖥️ **Cross-platform** — works on Linux, macOS, and Windows out of the box.

## Install

### From source (requires Go 1.27+)

**Linux / macOS:**

```bash
go install github.com/kripto-geek/liner-ai@latest
# Rename the binary for convenience:
mv "$(go env GOPATH)/bin/liner-ai" "$(go env GOPATH)/bin/liner"
```

**Windows (PowerShell):**

```powershell
go install github.com/kripto-geek/liner-ai@latest
# Rename the binary for convenience:
Rename-Item "$(go env GOPATH)\bin\liner-ai.exe" "liner.exe"
```

### Build from source

**Linux / macOS:**

```bash
git clone https://github.com/kripto-geek/liner-ai.git
cd liner-ai
go build -o liner .
sudo mv liner /usr/local/bin/   # or anywhere on your PATH
```

**Windows (PowerShell):**

```powershell
git clone https://github.com/kripto-geek/liner-ai.git
cd liner-ai
go build -o liner.exe .
# Move liner.exe to a directory in your PATH, e.g.:
Move-Item liner.exe "$env:USERPROFILE\go\bin\liner.exe"
```

### Cross-compile

Build for any platform from any platform:

```bash
# Windows
GOOS=windows GOARCH=amd64 go build -o liner.exe .

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o liner .

# Linux
GOOS=linux GOARCH=amd64 go build -o liner .
```

## Quick start

```bash
# First run triggers interactive setup
liner find all git repos in my home directory

# Or run setup explicitly
liner setup
```

## Usage

```
liner <natural language query>   Ask for a command
liner                            Interactive prompt mode
liner setup                      Configure provider & API key
liner config                     Show current configuration
liner version                    Print version
liner help                       Show this help
```

### Examples

```bash
liner merge this branch into main
liner list all docker containers including stopped ones
liner find files larger than 100MB in home directory
liner compress this folder into a tar.gz
liner show my public IP address
liner kill the process using port 3000
liner create a systemd service for my app
```

On Windows, the same queries produce Windows-native commands:

```powershell
liner find files larger than 100MB in home directory
# → Get-ChildItem -Path $HOME -Recurse | Where-Object { $_.Length -gt 100MB }

liner kill the process using port 3000
# → Stop-Process -Id (Get-NetTCPConnection -LocalPort 3000).OwningProcess -Force
```

## Configuration

Config is stored at:
- **Linux / macOS:** `~/.config/liner/config.json` (respects `$XDG_CONFIG_HOME` and `$LINER_HOME`)
- **Windows:** `%USERPROFILE%\.config\liner\config.json` (respects `%LINER_HOME%`)

```json
{
  "os": "linux",
  "arch": "amd64",
  "distro": "Arch Linux",
  "shell": "bash",
  "provider": "openrouter",
  "model": "openai/gpt-4o-mini",
  "base_url": "https://openrouter.ai/api/v1",
  "key_env": "OPENROUTER_API_KEY",
  "first_run_done": true
}
```

### Providers

| Provider     | Default Model        | API Key Env Var      | Notes                 |
| ------------ | -------------------- | -------------------- | --------------------- |
| `openrouter` | `openai/gpt-4o-mini` | `OPENROUTER_API_KEY` | Default, many models  |
| `openai`     | `gpt-4o-mini`        | `OPENAI_API_KEY`     |                       |
| `kilo`       | `openai/gpt-4o-mini` | `KILO_API_KEY`       |                       |
| `ollama`     | `llama3.1`           | —                    | Local, no key needed  |

### API Key

You can supply the API key in three ways (checked in order):

1. Pasted during `liner setup` (stored in config)
2. Provider-specific env var (e.g. `OPENROUTER_API_KEY`)
3. Generic `LINER_API_KEY` env var

## How it works

1. On first run, **liner** detects your OS, architecture, distro, and shell — saves to config.
2. Your natural-language query is sent to the configured AI provider along with the full environment context.
3. The AI returns a structured JSON response with exact, copy-paste-ready shell commands tailored to your OS and shell.
4. If the AI is confident, **liner** offers to execute the commands for you.

## Platform support

| Platform              | Build | Run | Secure key input |
| --------------------- | ----- | --- | ---------------- |
| Linux (any distro)    | ✅    | ✅  | ✅ echo-off       |
| macOS (Intel & ARM)   | ✅    | ✅  | ⚠️ plain stdin    |
| Windows (x64 & ARM)   | ✅    | ✅  | ⚠️ plain stdin    |

> **Note:** On macOS and Windows, the API key may be visible when pasting during `liner setup`. Use an environment variable instead for secure key management.

## License

MIT — see [LICENSE](LICENSE).
