package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// execOutput runs a command and returns its stdout bytes and error.
func execOutput(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	return cmd.Output()
}

// readLine reads a single line from os.Stdin.
func readLine(prompt string) (string, error) {
	if prompt != "" {
		fmt.Print(prompt)
	}
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		err := sc.Err()
		if err == nil {
			err = fmt.Errorf("no input")
		}
		return "", err
	}
	return strings.TrimSpace(sc.Text()), nil
}

// readLineHidden reads a secret (API key) without echoing it to the terminal.
// It falls back to plain readLine when the handle/fsys call is unavailable,
// so it also works in non-TTY pipelines (key can still be pasted).
func readLineHidden(prompt string) (string, error) {
	if data, err := getPass(prompt); err == nil {
		return strings.TrimSpace(string(data)), nil
	}
	// No secure terminal read available — accept the key over stdin as-is.
	return readLine(prompt)
}

// askYN prompts y/N.
func askYN(prompt string, def bool) bool {
	v, err := readLine(prompt)
	if err != nil || v == "" {
		return def
	}
	v = strings.ToLower(v)
	return v == "y" || v == "yes"
}

// errorColor wraps s in red for terminals that support it.
func red(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\x1b[31m" + s + "\x1b[0m"
}

func green(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\x1b[32m" + s + "\x1b[0m"
}

func dim(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\x1b[2m" + s + "\x1b[0m"
}

func bold(s string) string {
	if os.Getenv("NO_COLOR") != "" {
		return s
	}
	return "\x1b[1m" + s + "\x1b[0m"
}
