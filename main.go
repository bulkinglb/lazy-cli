package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"lazy-cli/config"
	"lazy-cli/doctor"
	"lazy-cli/llm"
	"lazy-cli/logger"
	"lazy-cli/repl"
	"lazy-cli/setup"
)

func main() {
	// Check for subcommands before flag parsing
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "setup":
			os.Exit(setup.Run(os.Args[2:]))
		case "status":
			os.Exit(doctor.RunStatus(os.Args[2:]))
		case "doctor":
			os.Exit(doctor.RunDoctor(os.Args[2:]))
		case "help", "--help", "-h":
			printUsage()
			return
		case "version", "--version":
			fmt.Println("lazy-cli v0.1.2")
			return
		}
	}

	// Default: run the interactive REPL
	runREPL()
}

func runREPL() {
	// Load .env file first
	loadEnv(".env")

	// Load persistent config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config error: %v\n", err)
	}

	// CLI flags (with env fallbacks, config as secondary fallback)
	modelPath := flag.String("model", "", "Path to GGUF model file")
	serverBinary := flag.String("server", "", "Path to llama-server binary")
	port := flag.Int("port", cfg.Port, "Port for llama-server")
	flag.Parse()

	// Resolve model path: flag > env > config
	if *modelPath == "" {
		*modelPath = os.Getenv("LLAMA_MODEL_PATH")
	}
	if *modelPath == "" {
		*modelPath = cfg.ModelPath
	}

	// Resolve server binary: flag > env > config > default
	if *serverBinary == "" {
		*serverBinary = os.Getenv("LLAMA_SERVER_PATH")
	}
	if *serverBinary == "" && cfg.ServerPath != "" {
		*serverBinary = cfg.ServerPath
	}
	if *serverBinary == "" {
		*serverBinary = "llama-server"
	}

	// Sync port back to config
	cfg.Port = *port

	if *modelPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --model is required (or run 'lazy-cli setup')")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Quick start:")
		fmt.Fprintln(os.Stderr, "  lazy-cli setup --llama-server /path/to/llama-server --model /path/to/model.gguf")
		os.Exit(1)
	}

	// Create LLM server manager
	server := llm.NewServer(*serverBinary, *modelPath)
	server.Port = *port

	fmt.Println("Starting LLM server...")
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'lazy-cli doctor' to diagnose the issue.")
		os.Exit(1)
	}
	defer server.Stop()

	if server.PID() > 0 {
		fmt.Printf("LLM server started (PID: %d)\n", server.PID())
	} else {
		fmt.Println("Using existing LLM server")
	}

	// Create logger
	var log *logger.Logger
	if cfg.LogEnabled {
		log, err = logger.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: logging disabled: %v\n", err)
		}
	}
	if log != nil {
		defer log.Close()
	}

	// Create LLM client
	client := llm.NewClient(server)

	// Create REPL
	r := repl.New(cfg, log, server)
	r.SetAIHandler(func(input string) (string, error) {
		return client.GenerateCommand(input)
	})

	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		r.Stop()
	}()

	if err := r.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("lazy-cli - Natural language to shell commands")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  lazy-cli              Start interactive CLI (default)")
	fmt.Println("  lazy-cli setup        Configure llama-server and model paths")
	fmt.Println("  lazy-cli status       Show current configuration and file status")
	fmt.Println("  lazy-cli doctor       Run diagnostic checks")
	fmt.Println("  lazy-cli help         Show this help message")
	fmt.Println("  lazy-cli version      Show version")
	fmt.Println()
	fmt.Println("Flags (for interactive mode):")
	fmt.Println("  --model PATH         Path to GGUF model file")
	fmt.Println("  --server PATH        Path to llama-server binary")
	fmt.Println("  --port PORT          Port for llama-server")
	fmt.Println()
	fmt.Println("Setup:")
	fmt.Println("  lazy-cli setup --llama-server /path/to/llama-server --model /path/to/model.gguf")
}

// loadEnv reads a .env file and sets environment variables
func loadEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return // .env is optional
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if key, value, ok := strings.Cut(line, "="); ok {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			value = strings.Trim(value, `"'`) // Remove quotes
			os.Setenv(key, value)
		}
	}
}
