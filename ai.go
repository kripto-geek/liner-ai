package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CommandReply is what the model is asked to produce.
type CommandReply struct {
	Commands []string `json:"commands"`
	Note     string   `json:"note"` // short human context: why, caveats, interactive prompts
	Conf     bool     `json:"confidence"`
}

// ask sends the user's request to the configured OpenAI-compatible endpoint
// and returns the parsed reply.
func ask(c *Config, question string) (*CommandReply, error) {
	key, hasKey := c.apiKey()
	if c.needsKey() && !hasKey {
		return nil, fmt.Errorf("no API key for provider %q (set %s or run %s)",
			c.Provider, c.keyEnvOr("LINER_API_KEY"), bold("liner setup"))
	}

	// Client with a generous-ish but bounded timeout.
	client := &http.Client{Timeout: 120 * time.Second}

	reqBody := map[string]any{
		"model":       c.Model,
		"temperature": 0.1,
		"messages": []map[string]any{
			{"role": "system", "content": systemPrompt(c)},
			{"role": "user", "content": question},
		},
	}
	buf, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", strings.TrimRight(c.BaseURL, "/")+"/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", c.BaseURL, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		// Try to extract a clean error message from the provider's JSON response.
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		hint := ""
		if resp.StatusCode == 400 || resp.StatusCode == 401 || resp.StatusCode == 404 {
			hint = "\n  → run " + bold("liner setup") + " to change model or provider"
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("provider %s (model %s) returned %d:\n  %s%s",
				c.Provider, c.Model, resp.StatusCode, apiErr.Error.Message, hint)
		}
		return nil, fmt.Errorf("provider %s returned %d: %s%s", c.Provider, resp.StatusCode,
			truncate(string(body), 300), hint)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("bad response from %s: %w", c.Provider, err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response from %s", c.Provider)
	}

	return parseCommandReply(parsed.Choices[0].Message.Content)
}

// systemPrompt encodes the whole "OS-aware, generate exact shell commands"
// contract plus the detected environment so the model never has to guess.
func systemPrompt(c *Config) string {
	env := map[string]string{
		"os":        c.OS,
		"arch":      c.Arch,
		"shell":     c.Shell,
	}
	if c.Distro != "" {
		env["distro"] = c.Distro
	}
	envJSON, _ := json.Marshal(env)

	return `You are "liner", a terminal command assistant. The user asks for a
command in plain language; you reply with the exact command(s) to run.

TARGET ENVIRONMENT (authoritative — never ask the user, never guess):
` + string(envJSON) + `

Rules:
- Output ONLY JSON, matching exactly:
  {"commands": ["<shell line 1>", "<shell line 2>", ...], "note": "<one short sentence: caveats/interactive prompts>", "confidence": true}
- "commands" is a list of raw shell lines in execution order. One task may
  need multiple lines (e.g. fetch + merge, or an export followed by the
  command). Each line must be a complete, copy-pasteable command for the
  target shell — no backticks, no trailing punctuation, no markdown.
- Match the shell: for powershell use Windows syntax; for bash/zsh use
  POSIX syntax; never emit cross-shell nonsense.
- If a step is interactive or destructive (passwords, prompts, --force, rm),
  say so in "note" and prefer safe flags where one exists.
- Prefer the most standard, non-destructive tool for the job (git, coreutils,
  native OS tools). Do not invent flags that may not exist on that OS.
- If the request is impossible or you don't know a reliable command, still
  return valid JSON: use "commands": [] and explain in "note" what is missing.
- Never add prose outside the JSON object. No code fences.`
}

// parseCommandReply extracts the JSON object from a possibly-fenced model
// reply and validates it.
func parseCommandReply(content string) (*CommandReply, error) {
	content = strings.TrimSpace(content)

	// Strip a leading/trailing markdown fence if present.
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	// Find the outermost { ... } so stray prose before/after doesn't break us.
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("model returned no JSON object:\n%s", truncate(content, 400))
	}
	jsonPart := content[start : end+1]

	var reply CommandReply
	if err := json.Unmarshal([]byte(jsonPart), &reply); err != nil {
		return nil, fmt.Errorf("could not parse model JSON: %w\nraw: %s", err, truncate(jsonPart, 300))
	}

	// Normalize commands: drop empty/whitespace-only lines.
	out := make([]string, 0, len(reply.Commands))
	for _, cmd := range reply.Commands {
		if s := strings.TrimSpace(cmd); s != "" {
			out = append(out, s)
		}
	}
	reply.Commands = out
	return &reply, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + " …"
}
