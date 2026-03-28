package executor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// Result contains the outcome of a command execution
type Result struct {
	ExitCode int
	Err      error
	Stdout   string
	Stderr   string
}

// Success returns true if command exited with code 0
func (r Result) Success() bool {
	return r.ExitCode == 0 && r.Err == nil
}

// Executor runs shell commands
type Executor struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

// New creates an executor with default streams (os.Stdout, os.Stderr, os.Stdin)
func New() *Executor {
	return &Executor{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
	}
}

// Run executes a command string via sh -c
// Output streams live to configured writers
func (e *Executor) Run(command string) Result {
	cmd := exec.Command("sh", "-c", command)

	// Connect streams for live output
	cmd.Stdout = e.Stdout
	cmd.Stderr = e.Stderr
	cmd.Stdin = e.Stdin

	// Run and wait for completion
	err := cmd.Run()

	return Result{
		ExitCode: exitCode(err),
		Err:      err,
	}
}

// RunSilent executes a command and captures output instead of streaming
func (e *Executor) RunSilent(command string) (stdout, stderr string, result Result) {
	cmd := exec.Command("sh", "-c", command)

	outPipe, _ := cmd.StdoutPipe()
	errPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return "", "", Result{ExitCode: -1, Err: err}
	}

	outBytes, _ := io.ReadAll(outPipe)
	errBytes, _ := io.ReadAll(errPipe)

	err := cmd.Wait()

	return string(outBytes), string(errBytes), Result{
		ExitCode: exitCode(err),
		Err:      err,
	}
}

// exitCode extracts the exit code from an error
func exitCode(err error) int {
	if err == nil {
		return 0
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}

	return -1 // Unknown error
}

// FormatResult returns a human-readable status line
func FormatResult(r Result) string {
	if r.Success() {
		return "✓ Command completed successfully"
	}
	if r.ExitCode > 0 {
		return fmt.Sprintf("✗ Command failed (exit code: %d)", r.ExitCode)
	}
	return fmt.Sprintf("✗ Command error: %v", r.Err)
}
