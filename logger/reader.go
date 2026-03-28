package logger

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SessionInfo describes a single session log file
type SessionInfo struct {
	Path       string
	Name       string // filename without dir
	EntryCount int
	SizeBytes  int64
	IsCurrent  bool
}

// FilePath returns the current session log file path
func (l *Logger) FilePath() string {
	if l.file == nil {
		return ""
	}
	return l.file.Name()
}

// LogDir returns the directory containing log files
func (l *Logger) LogDir() string {
	if l.file == nil {
		return ""
	}
	return filepath.Dir(l.file.Name())
}

// ListSessions returns info about all session log files, sorted oldest first
func (l *Logger) ListSessions() ([]SessionInfo, error) {
	l.file.Sync()

	dir := l.LogDir()
	if dir == "" {
		return nil, fmt.Errorf("no log directory")
	}

	files, err := filepath.Glob(filepath.Join(dir, "session_*.jsonl"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files) // oldest first by timestamp in filename

	currentFile := l.file.Name()
	var sessions []SessionInfo

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}

		entries, _ := readEntriesFromFile(f)

		sessions = append(sessions, SessionInfo{
			Path:       f,
			Name:       strings.TrimSuffix(filepath.Base(f), ".jsonl"),
			EntryCount: len(entries),
			SizeBytes:  info.Size(),
			IsCurrent:  f == currentFile,
		})
	}

	return sessions, nil
}

// ReadFile reads all entries from a specific log file path
func (l *Logger) ReadFile(path string) ([]Entry, error) {
	// If it's the current session, sync first
	if path == l.file.Name() {
		l.file.Sync()
	}
	return readEntriesFromFile(path)
}

// ReadAll reads all entries from the current session log
func (l *Logger) ReadAll() ([]Entry, error) {
	l.file.Sync()
	return readEntriesFromFile(l.file.Name())
}

// ReadAllAcrossSessions reads all entries across all session files
func (l *Logger) ReadAllAcrossSessions() ([]Entry, error) {
	l.file.Sync()

	dir := l.LogDir()
	if dir == "" {
		return nil, fmt.Errorf("no log directory")
	}

	files, err := filepath.Glob(filepath.Join(dir, "session_*.jsonl"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	var entries []Entry
	for _, f := range files {
		fileEntries, err := readEntriesFromFile(f)
		if err != nil {
			continue
		}
		entries = append(entries, fileEntries...)
	}
	return entries, nil
}

// Clear truncates the current session log file
func (l *Logger) Clear() error {
	if l.file == nil {
		return fmt.Errorf("no active log file")
	}
	l.file.Sync()
	if err := l.file.Truncate(0); err != nil {
		return fmt.Errorf("clear log: %w", err)
	}
	if _, err := l.file.Seek(0, 0); err != nil {
		return fmt.Errorf("clear log seek: %w", err)
	}
	return nil
}

// ClearAll removes all session log files from the logs directory,
// then truncates the current session file (which stays open)
func (l *Logger) ClearAll() (int, error) {
	dir := l.LogDir()
	if dir == "" {
		return 0, fmt.Errorf("no log directory")
	}

	files, err := filepath.Glob(filepath.Join(dir, "session_*.jsonl"))
	if err != nil {
		return 0, err
	}

	currentFile := l.file.Name()
	removed := 0

	for _, f := range files {
		if f == currentFile {
			continue
		}
		if err := os.Remove(f); err == nil {
			removed++
		}
	}

	// Truncate current session file instead of deleting it
	l.file.Sync()
	l.file.Truncate(0)
	l.file.Seek(0, 0)
	removed++

	return removed, nil
}

func readEntriesFromFile(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}
