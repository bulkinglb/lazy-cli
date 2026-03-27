package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lukas/lazy-ai-cli/executor"
	"github.com/lukas/lazy-ai-cli/safety"
)

const defaultPrompt = "lazy-cli> "

// AIHandler processes natural language input and returns a shell command
type AIHandler func(input string) (string, error)

// REPL manages the interactive command loop
type REPL struct {
	prompt    string
	registry  *CommandRegistry
	aiHandler AIHandler
	executor  *executor.Executor
	safety    *safety.Checker
	reader    *bufio.Reader
	writer    io.Writer
	running   bool
	history   []string
}

// New creates a REPL with default settings
func New() *REPL {
	r := &REPL{
		prompt:   defaultPrompt,
		registry: NewCommandRegistry(),
		executor: executor.New(),
		safety:   safety.NewChecker(),
		reader:   bufio.NewReader(os.Stdin),
		writer:   os.Stdout,
		history:  make([]string, 0, 100),
	}
	r.registerBuiltinCommands()
	return r
}

// SetAIHandler sets the handler for natural language input
func (r *REPL) SetAIHandler(h AIHandler) {
	r.aiHandler = h
}

// RegisterCommand adds a custom command
func (r *REPL) RegisterCommand(name, description string, handler CommandHandler) {
	r.registry.Register(name, description, handler)
}

// Run starts the REPL loop
func (r *REPL) Run() error {
	r.running = true
	r.printWelcome()

	for r.running {
		r.print(r.prompt)

		line, err := r.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				r.println("\nGoodbye!")
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		r.history = append(r.history, line)
		r.handleInput(line)
	}

	return nil
}

// Stop terminates the REPL loop
func (r *REPL) Stop() {
	r.running = false
}

func (r *REPL) handleInput(input string) {
	cmd := Parse(input)

	switch cmd.Type {
	case CommandTypeInternal:
		r.handleInternalCommand(cmd)
	case CommandTypeDirect:
		r.executeWithSafety(cmd.RawText)
	case CommandTypeAI:
		r.handleAIInput(cmd)
	}
}

func (r *REPL) handleInternalCommand(cmd Command) {
	handler, ok := r.registry.Get(cmd.Name)
	if !ok {
		r.printf("Unknown command: %s (use %%&help for available commands)\n", cmd.Name)
		return
	}

	if err := handler(cmd.Args); err != nil {
		r.printf("Error: %v\n", err)
	}
}

func (r *REPL) handleAIInput(cmd Command) {
	if r.aiHandler == nil {
		r.println("AI handler not configured")
		return
	}

	result, err := r.aiHandler(cmd.RawText)
	if err != nil {
		r.printf("AI error: %v\n", err)
		return
	}

	r.printf("Command: %s\n", result)
	r.executeWithSafety(result)
}

func (r *REPL) executeWithSafety(command string) {
	// Check safety
	check := r.safety.Check(command)

	switch check.Level {
	case safety.Blocked:
		r.printf("⛔ BLOCKED: %s\n", check.Reason)
		return

	case safety.Dangerous:
		r.printf("⚠️  DANGEROUS: %s\n", check.Reason)
		if !r.confirm("Execute anyway? [y/N]: ") {
			r.println("Cancelled.")
			return
		}

	case safety.Caution:
		r.printf("⚡ Caution: %s\n", check.Reason)
		if !r.confirm("Execute? [Y/n]: ") {
			r.println("Cancelled.")
			return
		}

	case safety.Safe:
		// Auto-execute in yolo mode, otherwise quick confirm
		if r.safety.Mode != safety.ModeYolo {
			if !r.confirm("Execute? [Y/n]: ") {
				r.println("Cancelled.")
				return
			}
		}
	}

	// Execute
	r.println("---")
	result := r.executor.Run(command)
	r.println("---")
	r.println(executor.FormatResult(result))
}

func (r *REPL) confirm(prompt string) bool {
	r.print(prompt)
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.ToLower(strings.TrimSpace(line))

	// Default yes for [Y/n], default no for [y/N]
	if strings.Contains(prompt, "[Y/n]") {
		return line == "" || line == "y" || line == "yes"
	}
	return line == "y" || line == "yes"
}

func (r *REPL) registerBuiltinCommands() {
	r.registry.Register("help", "Show available commands", r.cmdHelp)
	r.registry.Register("exit", "Exit the CLI", r.cmdExit)
	r.registry.Register("quit", "Exit the CLI", r.cmdExit)
	r.registry.Register("history", "Show command history", r.cmdHistory)
	r.registry.Register("config", "Show current configuration", r.cmdConfig)
	r.registry.Register("status", "Show LLM server status", r.cmdStatus)
}

func (r *REPL) cmdHelp(_ string) error {
	r.println("Available commands:")
	for name, desc := range r.registry.List() {
		r.printf("  §%-10s %s\n", name, desc)
	}
	r.println("\nPrefixes:")
	r.println("  !command    Execute shell command directly")
	r.println("  (no prefix) Send to AI for command generation")
	return nil
}

func (r *REPL) cmdExit(_ string) error {
	r.println("Goodbye!")
	r.Stop()
	return nil
}

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

func (r *REPL) cmdConfig(_ string) error {
	r.println("Configuration: (not yet implemented)")
	// TODO: Integrate with config package
	return nil
}

func (r *REPL) cmdStatus(_ string) error {
	r.println("LLM Status: (not yet implemented)")
	// TODO: Integrate with LLM server manager
	return nil
}

func (r *REPL) printWelcome() {
	r.println("lazy-ai-cli - Natural language to shell commands")
	r.println("Type %&help for available commands, or enter a request.")
	r.println("")
}

func (r *REPL) print(s string) {
	fmt.Fprint(r.writer, s)
}

func (r *REPL) println(s string) {
	fmt.Fprintln(r.writer, s)
}

func (r *REPL) printf(format string, args ...any) {
	fmt.Fprintf(r.writer, format, args...)
}
