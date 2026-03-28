package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lukas/lazy-ai-cli/config"
	"github.com/lukas/lazy-ai-cli/llm"
	"github.com/lukas/lazy-ai-cli/logger"
	"github.com/lukas/lazy-ai-cli/repl"
)

func main() {
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
		fmt.Fprintln(os.Stderr, "Error: --model is required (or set LLAMA_MODEL_PATH in .env, or §config model <path>)")
		os.Exit(1)
	}

	// Create LLM server manager
	server := llm.NewServer(*serverBinary, *modelPath)
	server.Port = *port

	fmt.Println("Starting LLM server...")
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
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
