package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config is everything liner persists so the model always knows the target
// OS/shell without re-detecting on every run.
type Config struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Distro   string `json:"distro,omitempty"`
	Shell    string `json:"shell"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key,omitempty"` // only set if user pasted one interactively
	KeyEnv   string `json:"key_env,omitempty"` // preferred env var for the key
	FirstRun bool   `json:"first_run_done"`
}

// providerSpec carries per-provider defaults. All speak the OpenAI-compatible
// /v1/chat/completions API, which keeps this dependency-free and universal.
type providerSpec struct {
	BaseURL string
	Model   string
	KeyEnv  string
	NoKey   bool
}

var providers = map[string]providerSpec{
	"openai":     {BaseURL: "https://api.openai.com/v1", Model: "gpt-4o-mini", KeyEnv: "OPENAI_API_KEY"},
	"openrouter": {BaseURL: "https://openrouter.ai/api/v1", Model: "openai/gpt-4o-mini", KeyEnv: "OPENROUTER_API_KEY"},
	"kilo":       {BaseURL: "https://api.kilocode.ai/v1", Model: "openai/gpt-4o-mini", KeyEnv: "KILO_API_KEY"},
	"ollama":     {BaseURL: "http://localhost:11434/v1", Model: "llama3.1", NoKey: true},
}

// configPath resolves where config.json lives.
func configPath() (string, error) {
	if p := os.Getenv("LINER_HOME"); p != "" {
		return filepath.Join(p, "config.json"), nil
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "liner", "config.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "liner", "config.json"), nil
}

// detect returns OS/arch/distro/shell for the machine liner runs on.
func detect() (osName, arch, distro, shell string) {
	osName = runtime.GOOS
	arch = runtime.GOARCH
	shell = detectShell()
	distro = detectDistro(osName)
	return
}

func detectShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	if s := os.Getenv("SHELL"); s != "" {
		return filepath.Base(s)
	}
	return "bash"
}

func detectDistro(osName string) string {
	switch osName {
	case "linux":
		if data, err := os.ReadFile("/etc/os-release"); err == nil {
			var pretty, name string
			for _, line := range strings.Split(string(data), "\n") {
				if v, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
					pretty = unquote(v)
				} else if v, ok := strings.CutPrefix(line, "NAME="); ok {
					name = unquote(v)
				}
			}
			if pretty != "" {
				return pretty
			}
			if name != "" {
				return name
			}
		}
		return "linux"
	case "darwin":
		if out, err := runOutput("sw_vers", "-productVersion"); err == nil {
			return "macOS " + strings.TrimSpace(out)
		}
		return "macOS"
	case "windows":
		return "windows"
	default:
		return osName
	}
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func runOutput(name string, args ...string) (string, error) {
	out, err := execOutput(name, args...)
	return string(out), err
}

// Load reads config, or nil if absent.
func load() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("corrupt config %s: %w", p, err)
	}
	return &c, nil
}

// Save writes config with 0600 (may contain a pasted API key).
func (c *Config) Save() error {
	p, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(p, append(b, '\n'), 0o600)
}

// apiKey resolves the effective key: explicit config value, then preferred
// env var, then a per-provider default env var.
func (c *Config) apiKey() (string, bool) {
	if c.APIKey != "" {
		return c.APIKey, true
	}
	if c.KeyEnv != "" {
		if k := os.Getenv(c.KeyEnv); k != "" {
			return k, true
		}
	}
	if spec, ok := providers[c.Provider]; ok {
		if k := os.Getenv(spec.KeyEnv); k != "" {
			return k, true
		}
	}
	if k := os.Getenv("LINER_API_KEY"); k != "" {
		return k, true
	}
	return "", false
}

// keyEnvOr returns the recommended environment variable name for the API key.
// If the config has one, use it; otherwise fall back to the provider's default;
// if neither exists, return the supplied fallback.
func (c *Config) keyEnvOr(fallback string) string {
	if c.KeyEnv != "" {
		return c.KeyEnv
	}
	if spec, ok := providers[c.Provider]; ok && spec.KeyEnv != "" {
		return spec.KeyEnv
	}
	return fallback
}

// needsKey reports whether the configured provider requires an API key.
func (c *Config) needsKey() bool {
	if spec, ok := providers[c.Provider]; ok {
		return !spec.NoKey
	}
	return true
}

// ensure performs first-run setup: detect environment, apply provider
// defaults, and persist. Returns whether config was freshly created.
func (c *Config) ensure() (bool, error) {
	fresh := !c.FirstRun
	if c.OS == "" {
		c.OS, c.Arch, c.Distro, c.Shell = detect()
	}
	if c.Provider == "" {
		c.Provider = "openrouter"
	}
	if spec, ok := providers[c.Provider]; ok {
		if c.BaseURL == "" {
			c.BaseURL = spec.BaseURL
		}
		if c.Model == "" {
			c.Model = spec.Model
		}
		if c.KeyEnv == "" && spec.KeyEnv != "" {
			c.KeyEnv = spec.KeyEnv
		}
	}
	if fresh {
		c.FirstRun = true
		if err := c.Save(); err != nil {
			return false, err
		}
	}
	return fresh, nil
}
