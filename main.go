package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/lukas/lazy-ai-cli/llm"
	"github.com/lukas/lazy-ai-cli/repl"
)

func main() {
	// Load .env file first
	loadEnv(".env")

	// CLI flags (with env fallbacks)
	modelPath := flag.String("model", os.Getenv("LLAMA_MODEL_PATH"), "Path to GGUF model file")
	serverBinary := flag.String("server", envOrDefault("LLAMA_SERVER_PATH", "llama-server"), "Path to llama-server binary")
	port := flag.Int("port", 8080, "Port for llama-server")
	flag.Parse()

	if *modelPath == "" {
		fmt.Fprintln(os.Stderr, "Error: --model is required (or set LLAMA_MODEL_PATH in .env)")
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

	// Create LLM client
	client := llm.NewClient(server)

	// Create REPL
	r := repl.New()
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

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
