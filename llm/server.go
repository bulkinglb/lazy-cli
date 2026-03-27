package llm

import (
	"fmt"
	"net/http"
	"os/exec"
	"syscall"
	"time"
)

// Server manages the llama-server process lifecycle
type Server struct {
	BinaryPath string // Path to llama-server binary
	ModelPath  string // Path to GGUF model file
	Host       string
	Port       int
	CtxSize    int // Context size (default 2048)

	cmd     *exec.Cmd
	started bool
}

// NewServer creates a server manager with defaults
func NewServer(binaryPath, modelPath string) *Server {
	return &Server{
		BinaryPath: binaryPath,
		ModelPath:  modelPath,
		Host:       "127.0.0.1",
		Port:       8080,
		CtxSize:    2048,
	}
}

// IsRunning checks if llama-server is responding on the configured port
func (s *Server) IsRunning() bool {
	url := fmt.Sprintf("http://%s:%d/health", s.Host, s.Port)
	client := &http.Client{Timeout: 2 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// Start launches llama-server as a subprocess
func (s *Server) Start() error {
	if s.IsRunning() {
		s.started = false // We didn't start it, someone else did
		return nil
	}

	args := []string{
		"-m", s.ModelPath,
		"--host", s.Host,
		"--port", fmt.Sprintf("%d", s.Port),
		"-c", fmt.Sprintf("%d", s.CtxSize),
	}

	s.cmd = exec.Command(s.BinaryPath, args...)
	s.cmd.Stdout = nil // Suppress output (or redirect to log file)
	s.cmd.Stderr = nil
	// Start in new process group so we can kill it cleanly
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start llama-server: %w", err)
	}

	s.started = true

	// Wait for server to be ready
	if err := s.waitReady(30 * time.Second); err != nil {
		s.Stop()
		return err
	}

	return nil
}

// Stop gracefully terminates the server if we started it
func (s *Server) Stop() error {
	if !s.started || s.cmd == nil || s.cmd.Process == nil {
		return nil
	}

	// Send SIGTERM for graceful shutdown
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// If SIGTERM fails, force kill
		s.cmd.Process.Kill()
	}

	// Wait for process to exit (with timeout)
	done := make(chan error, 1)
	go func() { done <- s.cmd.Wait() }()

	select {
	case <-done:
		// Clean exit
	case <-time.After(5 * time.Second):
		// Force kill if still running
		s.cmd.Process.Kill()
	}

	s.started = false
	s.cmd = nil
	return nil
}

// URL returns the base URL for the server
func (s *Server) URL() string {
	return fmt.Sprintf("http://%s:%d", s.Host, s.Port)
}

func (s *Server) waitReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s.IsRunning() {
			return nil
		}
		// Check if process died
		if s.cmd.ProcessState != nil && s.cmd.ProcessState.Exited() {
			return fmt.Errorf("llama-server exited unexpectedly")
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("llama-server failed to start within %v", timeout)
}

// PID returns the process ID if running
func (s *Server) PID() int {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Pid
	}
	return 0
}
