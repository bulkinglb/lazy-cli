package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry represents a single log record
type Entry struct {
	Timestamp    string `json:"ts"`
	Type         string `json:"type"`
	Input        string `json:"input,omitempty"`
	Command      string `json:"command,omitempty"`
	Safety       string `json:"safety,omitempty"`
	SafetyReason string `json:"safety_reason,omitempty"`
	ExitCode     *int   `json:"exit_code,omitempty"`
	Stdout       string `json:"stdout,omitempty"`
	Stderr       string `json:"stderr,omitempty"`
	DurationMs   int64  `json:"duration_ms,omitempty"`
	Error        string `json:"error,omitempty"`
}

// Logger writes structured JSON log entries to a session file
type Logger struct {
	file      *os.File
	sessionID string
}

// New creates a logger with a new session file in ~/.lazy-cli/logs/
func New() (*Logger, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("logger: %w", err)
	}

	logDir := filepath.Join(home, ".lazy-cli", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("logger: create log dir: %w", err)
	}

	sessionID := time.Now().Format("2006-01-02T15-04-05")
	filename := filepath.Join(logDir, "session_"+sessionID+".jsonl")

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("logger: open log file: %w", err)
	}

	return &Logger{file: file, sessionID: sessionID}, nil
}

// LogInteraction logs a full AI-driven interaction
func (l *Logger) LogInteraction(input, command, safety, safetyReason string, exitCode int, stdout, stderr string, duration time.Duration) {
	ec := exitCode
	l.write(Entry{
		Type:         "interaction",
		Input:        input,
		Command:      command,
		Safety:       safety,
		SafetyReason: safetyReason,
		ExitCode:     &ec,
		Stdout:       truncate(stdout, 4096),
		Stderr:       truncate(stderr, 4096),
		DurationMs:   duration.Milliseconds(),
	})
}

// LogDirect logs a direct shell command (! prefix)
func (l *Logger) LogDirect(command, safety, safetyReason string, exitCode int, stdout, stderr string, duration time.Duration) {
	ec := exitCode
	l.write(Entry{
		Type:         "direct",
		Command:      command,
		Safety:       safety,
		SafetyReason: safetyReason,
		ExitCode:     &ec,
		Stdout:       truncate(stdout, 4096),
		Stderr:       truncate(stderr, 4096),
		DurationMs:   duration.Milliseconds(),
	})
}

// LogBlocked logs a command that was blocked by the safety checker
func (l *Logger) LogBlocked(input, command, safetyReason string) {
	l.write(Entry{
		Type:         "blocked",
		Input:        input,
		Command:      command,
		Safety:       "blocked",
		SafetyReason: safetyReason,
	})
}

// LogError logs an error event
func (l *Logger) LogError(context string, err error) {
	l.write(Entry{
		Type:  "error",
		Input: context,
		Error: err.Error(),
	})
}

// Close flushes and closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// SessionID returns the current session identifier
func (l *Logger) SessionID() string {
	return l.sessionID
}

func (l *Logger) write(e Entry) {
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	data = append(data, '\n')
	l.file.Write(data)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
