package repl

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"lazy-cli/config"
	"lazy-cli/logger"
)

// Entry is a convenience alias for logger.Entry used in log display
type Entry = logger.Entry

// registerBuiltinCommands sets up all internal %& commands
func (r *REPL) registerBuiltinCommands() {
	r.registry.Register("help", "Show available commands", r.cmdHelp)
	r.registry.Register("exit", "Exit the CLI", r.cmdExit)
	r.registry.Register("quit", "Exit the CLI", r.cmdExit)
	r.registry.Register("history", "Show command history", r.cmdHistory)
	r.registry.Register("status", "Show current runtime status", r.cmdStatus)
	r.registry.Register("config", "Show or change configuration", r.cmdConfig)
	r.registry.Register("logs", "List session logs or view a session", r.cmdLogs)
	r.registry.Register("clearlogs", "Clear all log files", r.cmdClearLogs)
	r.registry.Register("batch",    "Execute a file of natural language instructions", r.cmdBatch)
	r.registry.Register("watch",    "Re-run a command on a schedule (e.g. watch 30s df -h)", r.cmdWatch)
	r.registry.Register("save",     "Save last command as a named snippet", r.cmdSave)
	r.registry.Register("run",      "Execute a saved snippet by name", r.cmdRun)
	r.registry.Register("snippets", "List all saved snippets", r.cmdSnippets)
}

// --- help ---

func (r *REPL) cmdHelp(_ string) error {
	prefix := r.cfg.Prefix
	r.println("Available commands:")
	for name, desc := range r.registry.List() {
		if name == "quit" {
			continue // don't clutter, it's an alias
		}
		r.printf("  %s%-12s %s\n", prefix, name, desc)
	}
	r.println("\nPrefixes:")
	r.println("  !command       Execute shell command directly")
	r.println("  (no prefix)    Send to AI for command generation")
	return nil
}

// --- exit ---

func (r *REPL) cmdExit(_ string) error {
	r.println("Goodbye!")
	r.Stop()
	return nil
}

// --- history ---

func (r *REPL) cmdHistory(_ string) error {
	if len(r.history) == 0 {
		r.println("No history yet.")
		return nil
	}
	r.println("Command history:")
	for i, h := range r.history {
		r.printf("  %3d: %s\n", i+1, h)
	}
	return nil
}

// --- status ---

func (r *REPL) cmdStatus(_ string) error {
	cwd, _ := os.Getwd()

	r.println("=== Runtime Status ===")
	r.printf("  Mode:            %s\n", r.cfg.Mode)
	r.printf("  Port:            %d\n", r.cfg.Port)
	r.printf("  Command prefix:  %s\n", r.cfg.Prefix)
	r.printf("  Logging:         %s\n", boolYesNo(r.cfg.LogEnabled))

	if r.log != nil {
		r.printf("  Log file:        %s\n", r.log.FilePath())
	} else {
		r.printf("  Log file:        (none)\n")
	}

	r.printf("  Model path:      %s\n", valueOrDefault(r.cfg.ModelPath, "(from .env/flag)"))
	r.printf("  Server path:     %s\n", valueOrDefault(r.cfg.ServerPath, "(from .env/flag)"))

	if r.server != nil {
		running := r.server.IsRunning()
		r.printf("  LLM server:      %s\n", boolStatus(running))
		if running {
			r.printf("  LLM server URL:  %s\n", r.server.URL())
		}
	} else {
		r.printf("  LLM server:      (not configured)\n")
	}

	r.printf("  Working dir:     %s\n", cwd)
	return nil
}

// --- config ---

func (r *REPL) cmdConfig(args string) error {
	if args == "" {
		return r.showConfig()
	}

	parts := strings.SplitN(args, " ", 2)
	key := strings.ToLower(parts[0])
	value := ""
	if len(parts) > 1 {
		value = strings.TrimSpace(parts[1])
	}

	switch key {
	case "show":
		return r.showConfig()

	case "mode":
		if value == "" {
			r.printf("Current mode: %s\n", r.cfg.Mode)
			r.printf("Valid modes: %s\n", strings.Join(config.ValidModes(), ", "))
			return nil
		}
		if !config.IsValidMode(value) {
			return fmt.Errorf("invalid mode %q (valid: %s)", value, strings.Join(config.ValidModes(), ", "))
		}
		r.cfg.Mode = value
		r.applySafetyMode()
		r.printf("Mode set to: %s\n", value)

	case "port":
		if value == "" {
			r.printf("Current port: %d\n", r.cfg.Port)
			return nil
		}
		p, err := strconv.Atoi(value)
		if err != nil || p < 1 || p > 65535 {
			return fmt.Errorf("invalid port %q (must be 1-65535)", value)
		}
		r.cfg.Port = p
		r.printf("Port set to: %d (takes effect on next server restart)\n", p)

	case "prefix":
		if value == "" {
			r.printf("Current prefix: %s\n", r.cfg.Prefix)
			return nil
		}
		if len(value) > 4 {
			return fmt.Errorf("prefix too long (max 4 characters)")
		}
		r.cfg.Prefix = value
		r.printf("Prefix set to: %s\n", value)

	case "logging":
		if value == "" {
			r.printf("Logging: %s\n", boolYesNo(r.cfg.LogEnabled))
			return nil
		}
		switch strings.ToLower(value) {
		case "on", "true", "yes":
			r.cfg.LogEnabled = true
			r.println("Logging enabled")
		case "off", "false", "no":
			r.cfg.LogEnabled = false
			r.println("Logging disabled")
		default:
			return fmt.Errorf("invalid value %q (use on/off)", value)
		}

	case "logpath":
		if value == "" {
			r.printf("Log path: %s\n", r.cfg.LogPath)
			return nil
		}
		r.cfg.LogPath = value
		r.printf("Log path set to: %s (takes effect on next session)\n", value)

	case "model":
		if value == "" {
			if r.cfg.ModelPath == "" {
				r.println("Model path: (not set, using .env or --model flag)")
			} else {
				r.printf("Model path: %s\n", r.cfg.ModelPath)
			}
			return nil
		}
		r.cfg.ModelPath = value
		r.printf("Model path set to: %s (takes effect on next restart)\n", value)

	case "server":
		if value == "" {
			if r.cfg.ServerPath == "" {
				r.println("Server path: (not set, using .env or --server flag)")
			} else {
				r.printf("Server path: %s\n", r.cfg.ServerPath)
			}
			return nil
		}
		r.cfg.ServerPath = value
		r.printf("Server path set to: %s (takes effect on next restart)\n", value)

	case "alias":
		return r.handleAlias(value)

	default:
		r.printf("Unknown config key: %s\n", key)
		r.println("Available keys: mode, port, prefix, logging, logpath, model, server, alias")
		return nil
	}

	return r.cfg.Save()
}

func (r *REPL) showConfig() error {
	r.println("=== Configuration ===")
	r.printf("  mode:       %s\n", r.cfg.Mode)
	r.printf("  port:       %d\n", r.cfg.Port)
	r.printf("  prefix:     %s\n", r.cfg.Prefix)
	r.printf("  logging:    %s\n", boolYesNo(r.cfg.LogEnabled))
	r.printf("  logpath:    %s\n", r.cfg.LogPath)
	r.printf("  model:      %s\n", valueOrDefault(r.cfg.ModelPath, "(not set)"))
	r.printf("  server:     %s\n", valueOrDefault(r.cfg.ServerPath, "(not set)"))

	if len(r.cfg.PathAliases) > 0 {
		r.println("  aliases:")
		for name, path := range r.cfg.PathAliases {
			r.printf("    %-12s → %s\n", name, path)
		}
	} else {
		r.println("  aliases:    (none)")
	}

	r.printf("\n  Config file: %s\n", r.cfg.FilePath())
	r.printf("  Usage: %sconfig <key> <value>\n", r.cfg.Prefix)
	return nil
}

func (r *REPL) handleAlias(args string) error {
	if args == "" {
		if len(r.cfg.PathAliases) == 0 {
			r.println("No path aliases configured.")
			r.printf("Usage: %sconfig alias <name> <path>  |  %sconfig alias rm <name>\n", r.cfg.Prefix, r.cfg.Prefix)
			return nil
		}
		r.println("Path aliases:")
		for name, path := range r.cfg.PathAliases {
			r.printf("  %-12s → %s\n", name, path)
		}
		return nil
	}

	parts := strings.SplitN(args, " ", 2)
	name := parts[0]

	// Remove alias
	if name == "rm" && len(parts) > 1 {
		target := strings.TrimSpace(parts[1])
		if _, ok := r.cfg.PathAliases[target]; !ok {
			return fmt.Errorf("alias %q not found", target)
		}
		delete(r.cfg.PathAliases, target)
		r.printf("Alias %q removed\n", target)
		return nil
	}

	// Set alias
	if len(parts) < 2 {
		if path, ok := r.cfg.PathAliases[name]; ok {
			r.printf("%s → %s\n", name, path)
		} else {
			r.printf("Alias %q not found\n", name)
		}
		return nil
	}

	path := strings.TrimSpace(parts[1])
	r.cfg.PathAliases[name] = path
	r.printf("Alias set: %s → %s\n", name, path)
	return nil
}

// --- logs ---

func (r *REPL) cmdLogs(args string) error {
	if r.log == nil {
		r.println("Logging is not active.")
		return nil
	}

	args = strings.TrimSpace(args)

	// No args: list all session files
	if args == "" {
		return r.listLogSessions()
	}

	// "all": dump every entry across all sessions
	if args == "all" {
		entries, err := r.log.ReadAllAcrossSessions()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			r.println("No log entries found.")
			return nil
		}
		r.printf("=== All log entries (%d) ===\n", len(entries))
		r.printEntries(entries)
		return nil
	}

	// Number: view that session
	n, err := strconv.Atoi(args)
	if err != nil {
		r.printf("Usage: %slogs          List all session files\n", r.cfg.Prefix)
		r.printf("       %slogs <N>      View session N\n", r.cfg.Prefix)
		r.printf("       %slogs all      View all entries across sessions\n", r.cfg.Prefix)
		return nil
	}

	return r.viewLogSession(n)
}

func (r *REPL) listLogSessions() error {
	sessions, err := r.log.ListSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		r.println("No log files found.")
		return nil
	}

	r.printf("=== Log sessions (%s) ===\n", r.log.LogDir())
	for i, s := range sessions {
		current := " "
		if s.IsCurrent {
			current = "*"
		}
		// Extract readable timestamp from filename: session_2026-03-28T09-32-08 → 2026-03-28 09:32:08
		ts := strings.TrimPrefix(s.Name, "session_")
		ts = strings.Replace(ts, "T", " ", 1)
		ts = strings.ReplaceAll(ts, "-", "-") // keep date dashes
		// Fix time part: replace only the dashes after T (now space)
		parts := strings.SplitN(ts, " ", 2)
		if len(parts) == 2 {
			parts[1] = strings.ReplaceAll(parts[1], "-", ":")
			ts = parts[0] + " " + parts[1]
		}

		r.printf("  %s [%d]  %s  (%d entries)\n", current, i+1, ts, s.EntryCount)
	}
	r.printf("\nUse %slogs <N> to view a session. * = current session\n", r.cfg.Prefix)
	return nil
}

func (r *REPL) viewLogSession(n int) error {
	sessions, err := r.log.ListSessions()
	if err != nil {
		return err
	}

	if n < 1 || n > len(sessions) {
		r.printf("Invalid session number. Use 1-%d.\n", len(sessions))
		return nil
	}

	s := sessions[n-1]
	entries, err := r.log.ReadFile(s.Path)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		r.printf("Session [%d] %s: no entries.\n", n, s.Name)
		return nil
	}

	r.printf("=== Session [%d] %s (%d entries) ===\n", n, s.Name, len(entries))
	r.printEntries(entries)
	return nil
}

func (r *REPL) printEntries(entries []Entry) {
	for _, e := range entries {
		r.printf("[%s] %s", e.Timestamp, e.Type)
		if e.Input != "" {
			r.printf(" | input: %s", e.Input)
		}
		if e.Command != "" {
			r.printf(" | cmd: %s", e.Command)
		}
		if e.Safety != "" {
			r.printf(" | safety: %s", e.Safety)
		}
		if e.ExitCode != nil {
			r.printf(" | exit: %d", *e.ExitCode)
		}
		if e.Error != "" {
			r.printf(" | error: %s", e.Error)
		}
		r.println("")
	}
}

// --- clearlogs ---

func (r *REPL) cmdClearLogs(_ string) error {
	if r.log == nil {
		r.println("Logging is not active.")
		return nil
	}

	if !r.confirm("Clear ALL log files? [y/N]: ") {
		r.println("Cancelled.")
		return nil
	}

	count, err := r.log.ClearAll()
	if err != nil {
		return fmt.Errorf("failed to clear logs: %w", err)
	}

	r.printf("Cleared %d log file(s).\n", count)
	return nil
}

// --- batch ---

func (r *REPL) cmdBatch(args string) error {
	path := strings.TrimSpace(args)
	if path == "" {
		r.println("Usage: batch <file>")
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		r.printf("Error reading file: %v\n", err)
		return nil
	}
	var tasks []string
	for _, l := range strings.Split(string(data), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		tasks = append(tasks, l)
	}
	if len(tasks) == 0 {
		r.println("No instructions found in file.")
		return nil
	}
	r.printf("Batch: %d instructions found.\n", len(tasks))
	for i, task := range tasks {
		r.printf("\n[%d/%d] %s\n", i+1, len(tasks), task)
		r.handleAIInputBatch(task)
		if !r.running {
			break
		}
	}
	r.println("\nBatch complete.")
	return nil
}

// --- watch ---

func (r *REPL) cmdWatch(args string) error {
	parts := strings.SplitN(strings.TrimSpace(args), " ", 2)
	if len(parts) < 2 {
		r.println("Usage: watch <interval> <query>  (e.g. watch 30s df -h)")
		return nil
	}
	interval, err := time.ParseDuration(parts[0])
	if err != nil {
		r.printf("Invalid interval %q: %v\n", parts[0], err)
		return nil
	}
	query := parts[1]

	var cmd string
	if strings.HasPrefix(query, "!") {
		cmd = strings.TrimPrefix(query, "!")
	} else if r.aiHandler != nil {
		cmd, err = r.aiHandler(query)
		if err != nil {
			r.printf("AI error: %v\n", err)
			return nil
		}
	} else {
		cmd = query
	}

	r.printf("Watch command : %s\n", cmd)
	r.printf("Interval      : %s\n", parts[0])
	if !r.confirm("Start watching? [Y/n]: ") {
		return nil
	}

	r.println("Running (Ctrl+C to stop)...")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	r.printf("[%s]\n", time.Now().Format("15:04:05"))
	r.executeNoConfirm(cmd)

	for r.running {
		<-ticker.C
		if !r.running {
			break
		}
		r.printf("\n[%s]\n", time.Now().Format("15:04:05"))
		r.executeNoConfirm(cmd)
	}
	return nil
}

// --- snippets ---

func (r *REPL) cmdSave(args string) error {
	name := strings.TrimSpace(args)
	if name == "" {
		r.println("Usage: save <name>")
		return nil
	}
	if r.lastCmd == "" {
		r.println("No command to save.")
		return nil
	}
	r.cfg.Snippets[name] = r.lastCmd
	if err := r.cfg.Save(); err != nil {
		return err
	}
	r.printf("Saved %q: %s\n", name, r.lastCmd)
	return nil
}

func (r *REPL) cmdRun(args string) error {
	name := strings.TrimSpace(args)
	if name == "" {
		r.println("Usage: run <name>")
		return nil
	}
	cmd, ok := r.cfg.Snippets[name]
	if !ok {
		r.printf("No snippet named %q. Use %ssnippets to list saved snippets.\n", name, r.cfg.Prefix)
		return nil
	}
	r.printf("Running snippet %q: %s\n", name, cmd)
	r.executeWithSafety(cmd, "")
	return nil
}

func (r *REPL) cmdSnippets(_ string) error {
	if len(r.cfg.Snippets) == 0 {
		r.printf("No snippets saved. Use %ssave <name> after running a command.\n", r.cfg.Prefix)
		return nil
	}
	r.println("Saved snippets:")
	for name, cmd := range r.cfg.Snippets {
		r.printf("  %-20s %s\n", name, cmd)
	}
	return nil
}

// --- helpers ---

func boolYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func boolStatus(b bool) string {
	if b {
		return "running"
	}
	return "stopped"
}

func valueOrDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
