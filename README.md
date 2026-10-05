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
- 🪶 **Zero Go dependencies** — pure stdlib, single binary, ~5 MB.

## Install

### From source (requires Go 1.27+)

```bash
go install github.com/kripto-geek/liner@latest
```

### Manual

```bash
git clone https://github.com/kripto-geek/liner.git
cd liner
go build -o liner .
sudo mv liner /usr/local/bin/   # or anywhere on your PATH
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

## Configuration

Config is stored at `~/.config/liner/config.json` (respects `$XDG_CONFIG_HOME` and `$LINER_HOME`).

```json
{
  "os": "linux",
  "arch": "amd64",
  "distro": "Arch Linux",
  "shell": "zsh",
  "provider": "openai",
  "model": "gpt-4o-mini",
  "base_url": "https://api.openai.com/v1",
  "key_env": "OPENAI_API_KEY",
  "first_run_done": true
}
```

### Providers

| Provider     | Default Model       | API Key Env Var      | Notes                  |
| ------------ | ------------------- | -------------------- | ---------------------- |
| `openai`     | `gpt-4o-mini`       | `OPENAI_API_KEY`     | Default                |
| `openrouter` | `openai/gpt-4o-mini` | `OPENROUTER_API_KEY` | Access to many models  |
| `kilo`       | `openai/gpt-4o-mini` | `KILO_API_KEY`       |                        |
| `ollama`     | `llama3.1`          | —                    | Local, no key needed   |

### API Key

You can supply the API key in three ways (checked in order):

1. Pasted during `liner setup` (stored in config)
2. Provider-specific env var (e.g. `OPENAI_API_KEY`)
3. Generic `LINER_API_KEY` env var

## How it works

1. On first run, **liner** detects your OS, architecture, distro, and shell — saves to `~/.config/liner/config.json`.
2. Your natural-language query is sent to the configured AI provider along with the full environment context.
3. The AI returns a structured JSON response with exact, copy-paste-ready shell commands.
4. If the AI is confident, **liner** offers to execute the commands for you.

## License

MIT — see [LICENSE](LICENSE).
