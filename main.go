package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const version = "0.2.0"

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "setup":
			runSetup(true)
			return
		case "version", "--version", "-v":
			fmt.Println("liner " + version)
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "config":
			showConfig()
			return
		}
	}

	// Load or create config.
	cfg, err := load()
	if err != nil {
		fatal("loading config: %v", err)
	}
	if cfg == nil {
		cfg = &Config{}
	}

	fresh, err := cfg.ensure()
	if err != nil {
		fatal("initialising config: %v", err)
	}
	if fresh {
		fmt.Println(bold("👋 Welcome to liner!") + " Let's get you set up.\n")
		runSetupWith(cfg)
		fmt.Println()
	}

	// Collect the query: either from args or prompt interactively.
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		q, err := readLine(dim("liner") + bold(" ▸ "))
		if err != nil {
			fatal("reading query: %v", err)
		}
		query = q
	}
	if query == "" {
		fmt.Fprintln(os.Stderr, dim("nothing to do"))
		return
	}

	// Ask the AI.
	fmt.Fprintln(os.Stderr, dim("  thinking…"))
	reply, err := ask(cfg, query)
	if err != nil {
		fatal("%v", err)
	}

	printReply(reply)

	// Offer to run if the model is confident and there are commands.
	if len(reply.Commands) > 0 && reply.Conf {
		fmt.Println()
		if askYN(dim("  run ")+"[Y/n]? ", true) {
			runCommands(cfg, reply.Commands)
		}
	}
}

// printReply renders the AI response nicely to the terminal.
func printReply(r *CommandReply) {
	fmt.Println()
	if len(r.Commands) == 0 {
		fmt.Println(red("  ✗ ") + "No command found.")
		if r.Note != "" {
			fmt.Println(dim("  " + r.Note))
		}
		return
	}

	for i, cmd := range r.Commands {
		if len(r.Commands) > 1 {
			fmt.Printf("  %s %s\n", dim(fmt.Sprintf("[%d]", i+1)), green(cmd))
		} else {
			fmt.Printf("  %s %s\n", green("▸"), green(cmd))
		}
	}

	if r.Note != "" {
		fmt.Println(dim("\n  ℹ  " + r.Note))
	}
}

// runCommands executes each command via the user's shell, streaming
// stdout/stderr so the user sees live output.
func runCommands(cfg *Config, cmds []string) {
	shellBin := resolveShell(cfg.Shell)
	flag := "-c"
	if strings.Contains(cfg.Shell, "powershell") || strings.Contains(cfg.Shell, "pwsh") {
		flag = "-Command"
	}
	for _, c := range cmds {
		fmt.Println(dim("  $ " + c))
		cmd := exec.Command(shellBin, flag, c)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, red("  command failed: ")+err.Error())
			if !askYN(dim("  continue ")+"[y/N]? ", false) {
				return
			}
		}
	}
}

// resolveShell returns the full path to the shell binary. For known
// shell names like "bash" or "zsh" it uses the PATH lookup.
func resolveShell(name string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	// Fallback: use name as-is (absolute path or best-effort).
	return name
}

// ── Setup ──────────────────────────────────────────────────────────────

func runSetup(force bool) {
	cfg, err := load()
	if err != nil {
		fatal("loading config: %v", err)
	}
	if cfg == nil {
		cfg = &Config{}
	}
	// Always re-detect environment on explicit setup.
	cfg.OS, cfg.Arch, cfg.Distro, cfg.Shell = detect()
	cfg.FirstRun = true
	fmt.Println(bold("liner setup"))
	fmt.Println()
	runSetupWith(cfg)
}

func runSetupWith(cfg *Config) {
	// Show detected environment.
	fmt.Println(dim("  Detected environment:"))
	fmt.Printf("    OS:     %s (%s)\n", cfg.OS, cfg.Arch)
	if cfg.Distro != "" && cfg.Distro != cfg.OS {
		fmt.Printf("    Distro: %s\n", cfg.Distro)
	}
	fmt.Printf("    Shell:  %s\n", cfg.Shell)
	fmt.Println()

	// Choose provider.
	fmt.Println(dim("  Available providers:"))
	provNames := []string{"openai", "openrouter", "kilo", "ollama"}
	for i, name := range provNames {
		spec := providers[name]
		marker := " "
		if name == cfg.Provider {
			marker = green("●")
		}
		extra := ""
		if spec.NoKey {
			extra = dim(" (no API key needed)")
		}
		fmt.Printf("    %s [%d] %s%s\n", marker, i+1, name, extra)
	}
	fmt.Println()
	choice, _ := readLine(dim("  provider ") + fmt.Sprintf("[1-%d, default: %s]: ", len(provNames), cfg.Provider))
	if choice != "" {
		idx := 0
		if _, err := fmt.Sscanf(choice, "%d", &idx); err == nil && idx >= 1 && idx <= len(provNames) {
			cfg.Provider = provNames[idx-1]
		} else {
			// Maybe they typed the name directly.
			for _, n := range provNames {
				if strings.EqualFold(choice, n) {
					cfg.Provider = n
					break
				}
			}
		}
	}

	spec := providers[cfg.Provider]
	cfg.BaseURL = spec.BaseURL
	cfg.Model = spec.Model
	cfg.KeyEnv = spec.KeyEnv

	// Custom model override.
	modelInput, _ := readLine(dim("  model ") + fmt.Sprintf("[default: %s]: ", cfg.Model))
	if modelInput != "" {
		cfg.Model = modelInput
	}

	// API key if needed.
	if !spec.NoKey {
		_, hasKey := cfg.apiKey()
		if hasKey {
			fmt.Println(dim("  API key: ") + green("✓ already set"))
			if askYN(dim("  replace key? ")+"[y/N]: ", false) {
				hasKey = false
			}
		}
		if !hasKey {
			fmt.Println(dim(fmt.Sprintf("  (or set env var %s)", cfg.keyEnvOr("LINER_API_KEY"))))
			key, _ := readLineHidden(dim("  API key: "))
			if key != "" {
				cfg.APIKey = key
				fmt.Println(green(" ✓"))
			}
		}
	}

	if err := cfg.Save(); err != nil {
		fatal("saving config: %v", err)
	}

	p, _ := configPath()
	fmt.Println(dim("\n  Config saved → ") + p)
	fmt.Println(dim("  Run ") + bold("liner setup") + dim(" to change settings anytime."))
}

// ── Utilities ──────────────────────────────────────────────────────────

func showConfig() {
	cfg, err := load()
	if err != nil {
		fatal("%v", err)
	}
	if cfg == nil {
		fmt.Println(dim("  no config found — run ") + bold("liner setup"))
		return
	}
	p, _ := configPath()
	fmt.Println(dim("  config: ") + p)
	fmt.Printf("  os:       %s (%s)\n", cfg.OS, cfg.Arch)
	if cfg.Distro != "" {
		fmt.Printf("  distro:   %s\n", cfg.Distro)
	}
	fmt.Printf("  shell:    %s\n", cfg.Shell)
	fmt.Printf("  provider: %s\n", cfg.Provider)
	fmt.Printf("  model:    %s\n", cfg.Model)
	fmt.Printf("  base_url: %s\n", cfg.BaseURL)
	_, hasKey := cfg.apiKey()
	if hasKey {
		fmt.Println("  api_key:  " + green("✓ set"))
	} else {
		fmt.Println("  api_key:  " + red("✗ not set"))
	}
}

func printHelp() {
	fmt.Println(bold("liner") + " — AI-powered, OS-aware terminal command assistant\n")
	fmt.Println("USAGE:")
	fmt.Println("  liner <natural language query>   Ask for a command")
	fmt.Println("  liner                            Interactive prompt mode")
	fmt.Println("  liner setup                      Configure provider & API key")
	fmt.Println("  liner config                     Show current configuration")
	fmt.Println("  liner version                    Print version")
	fmt.Println("  liner help                       Show this help")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  liner merge this branch into main")
	fmt.Println("  liner list all docker containers including stopped ones")
	fmt.Println("  liner find files larger than 100MB in home directory")
	fmt.Println("  liner compress this folder into a tar.gz")
	fmt.Println()
	fmt.Println(dim("On first run, liner detects your OS, architecture, shell, and"))
	fmt.Println(dim("distro — then stores it so the AI always generates the right commands."))
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, red("liner: ")+format+"\n", a...)
	os.Exit(1)
}
