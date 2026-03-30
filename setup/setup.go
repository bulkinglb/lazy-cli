package setup

import (
"flag"
"fmt"
"os"
"os/exec"
"path/filepath"

"lazy-cli/config"
"lazy-cli/doctor"
"lazy-cli/llm"
)

// Run executes the setup flow
func Run(args []string) int {
fs := flag.NewFlagSet("setup", flag.ExitOnError)
serverPath := fs.String("llama-server", "", "Path to llama-server binary")
modelPath := fs.String("model", "", "Path to GGUF model file")
port := fs.Int("port", 0, "Port for llama-server (default: from config)")
skipTest := fs.Bool("skip-test", false, "Skip the server test start")
fs.Parse(args)

fmt.Println("=== lazy-cli setup ===")
fmt.Println()

// Load or create config
cfg, err := config.Load()
if err != nil {
fmt.Fprintf(os.Stderr, "Warning: config error: %v\n", err)
}
fmt.Printf("Config: %s\n", cfg.FilePath())

// Ensure directories
if err := config.EnsureDirs(); err != nil {
fmt.Fprintf(os.Stderr, "Error creating directories: %v\n", err)
return 1
}
fmt.Println("Directories: OK")
fmt.Println()

// Apply flags over existing config
if *serverPath != "" {
cfg.ServerPath = *serverPath
}
if *modelPath != "" {
cfg.ModelPath = *modelPath
}
if *port > 0 {
cfg.Port = *port
}

// Auto-resolve llama-server if not configured
if cfg.ServerPath == "" {
if path, err := exec.LookPath("llama-server"); err == nil {
cfg.ServerPath = path
fmt.Printf("Found llama-server on PATH: %s\n\n", path)
} else {
fmt.Println("--- llama-server not found, downloading automatically ---")
binDir := filepath.Join(config.Dir(), "bin")
path, dlErr := DownloadLlamaServer(binDir)
if dlErr != nil {
fmt.Fprintf(os.Stderr, "Auto-download failed: %v\n\n", dlErr)
fmt.Fprintln(os.Stderr, "Provide it manually:")
fmt.Fprintln(os.Stderr, "  lazy-cli setup --llama-server /path/to/llama-server")
return 1
}
cfg.ServerPath = path
fmt.Printf("  ✓ Installed to %s\n\n", path)
}
}

// Auto-download Gemma 3 1B model if not configured
if cfg.ModelPath == "" {
fmt.Println("--- No model configured, downloading Gemma 3 1B automatically ---")
modelsDir := filepath.Join(config.Dir(), "models")
path, dlErr := DownloadGemmaModel(modelsDir)
if dlErr != nil {
fmt.Fprintf(os.Stderr, "Model download failed: %v\n\n", dlErr)
fmt.Fprintln(os.Stderr, "Provide it manually:")
fmt.Fprintln(os.Stderr, "  lazy-cli setup --model /path/to/model.gguf")
return 1
}
cfg.ModelPath = path
fmt.Printf("  ✓ Model saved to %s\n\n", path)
}

// Validate llama-server
fmt.Println("--- Checking llama-server ---")
srvCheck := doctor.CheckServerPath(cfg.ServerPath)
fmt.Println(srvCheck)
if !srvCheck.OK {
fmt.Println()
fmt.Println("Please provide the path to llama-server:")
fmt.Println("  lazy-ai setup --llama-server /path/to/llama-server")
return 1
}

// Validate model file
fmt.Println()
fmt.Println("--- Checking model ---")
mdlCheck := doctor.CheckModelPath(cfg.ModelPath)
fmt.Println(mdlCheck)
if !mdlCheck.OK {
fmt.Println()
fmt.Println("Please provide the path to a .gguf model file:")
fmt.Println("  lazy-ai setup --model /path/to/model.gguf")
return 1
}

// Validate port
fmt.Println()
fmt.Println("--- Checking port ---")
portCheck := doctor.CheckPort(cfg.Port)
fmt.Println(portCheck)
if !portCheck.OK {
fmt.Println()
fmt.Printf("Port %d is not available. Use --port to pick another.\n", cfg.Port)
return 1
}

// Save config with validated values
if err := cfg.Save(); err != nil {
fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
return 1
}
fmt.Println()
fmt.Println("Config saved.")

// Test server start
if *skipTest {
fmt.Println()
fmt.Println("Skipping server test (--skip-test).")
printSuccess(cfg)
return 0
}

fmt.Println()
fmt.Println("--- Testing llama-server ---")
fmt.Println("Starting server (this may take a moment)...")

server := llm.NewServer(cfg.ServerPath, cfg.ModelPath)
server.Port = cfg.Port

if err := server.Start(); err != nil {
fmt.Fprintf(os.Stderr, "\nServer test FAILED: %v\n", err)
fmt.Println()
fmt.Println("Possible causes:")
fmt.Println("  - Model file is corrupted or incompatible")
fmt.Println("  - llama-server binary doesn't match your architecture")
fmt.Println("  - Not enough memory for this model")
fmt.Println()
fmt.Println("Try running llama-server manually to see full output:")
fmt.Printf("  %s -m %s --port %d\n", cfg.ServerPath, cfg.ModelPath, cfg.Port)
return 1
}

// Health check
fmt.Println("Server started. Running health check...")
if err := doctor.HealthCheck("127.0.0.1", cfg.Port); err != nil {
fmt.Fprintf(os.Stderr, "Health check FAILED: %v\n", err)
server.Stop()
return 1
}
fmt.Println("  [OK] Health check passed")

// Stop test server
server.Stop()
fmt.Println("  [OK] Server stopped cleanly")

printSuccess(cfg)
return 0
}

func printSuccess(cfg *config.Config) {
fmt.Println()
fmt.Println("=== Setup complete ===")
fmt.Println()
fmt.Printf("  Server:  %s\n", cfg.ServerPath)
fmt.Printf("  Model:   %s\n", cfg.ModelPath)
fmt.Printf("  Port:    %d\n", cfg.Port)
fmt.Printf("  Config:  %s\n", cfg.FilePath())
fmt.Println()
fmt.Println("Run 'lazy-cli' to start the interactive CLI.")
}
