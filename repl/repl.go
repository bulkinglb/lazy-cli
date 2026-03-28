package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/lukas/lazy-ai-cli/config"
	"github.com/lukas/lazy-ai-cli/executor"
	"github.com/lukas/lazy-ai-cli/llm"
	"github.com/lukas/lazy-ai-cli/logger"
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
	log       *logger.Logger
	cfg       *config.Config
	server    *llm.Server
	reader    *bufio.Reader
	writer    io.Writer
	running   bool
	history   []string
}

// New creates a REPL with config, logger, and server reference
func New(cfg *config.Config, log *logger.Logger, server *llm.Server) *REPL {
	r := &REPL{
		prompt:   defaultPrompt,
		registry: NewCommandRegistry(),
		executor: executor.New(),
		safety:   safety.NewChecker(),
		log:      log,
		cfg:      cfg,
		server:   server,
		reader:   bufio.NewReader(os.Stdin),
		writer:   os.Stdout,
		history:  make([]string, 0, 100),
	}
	r.applySafetyMode()
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
	cmd := Parse(input, r.cfg.Prefix)

	switch cmd.Type {
	case CommandTypeInternal:
		r.handleInternalCommand(cmd)
	case CommandTypeDirect:
		r.executeWithSafety(cmd.RawText, "")
	case CommandTypeAI:
		r.handleAIInput(cmd)
	}
}

func (r *REPL) handleInternalCommand(cmd Command) {
	handler, ok := r.registry.Get(cmd.Name)
	if !ok {
		r.printf("Unknown command: %s (use %shelp for available commands)\n", cmd.Name, r.cfg.Prefix)
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
		r.logError("ai_generate", err)
		return
	}

	r.printf("Command: %s\n", result)
	r.executeWithSafety(result, cmd.RawText)
}

func (r *REPL) executeWithSafety(command, userInput string) {
	check := r.safety.Check(command)

	switch check.Level {
	case safety.Blocked:
		r.printf("BLOCKED: %s\n", check.Reason)
		r.logBlocked(userInput, command, check.Reason)
		return

	case safety.Dangerous:
		r.printf("DANGEROUS: %s\n", check.Reason)
		if !r.confirm("Execute anyway? [y/N]: ") {
			r.println("Cancelled.")
			return
		}

	case safety.Caution:
		r.printf("Caution: %s\n", check.Reason)
		if !r.confirm("Execute? [Y/n]: ") {
			r.println("Cancelled.")
			return
		}

	case safety.Safe:
		if r.safety.Mode != safety.ModeYolo {
			if !r.confirm("Execute? [Y/n]: ") {
				r.println("Cancelled.")
				return
			}
		}
	}

	r.println("---")
	start := time.Now()
	result := r.executor.Run(command)
	duration := time.Since(start)
	r.println("---")
	r.println(executor.FormatResult(result))

	if r.log != nil && r.cfg.LogEnabled {
		if userInput != "" {
			r.log.LogInteraction(userInput, command, check.Level.String(), check.Reason, result.ExitCode, result.Stdout, result.Stderr, duration)
		} else {
			r.log.LogDirect(command, check.Level.String(), check.Reason, result.ExitCode, result.Stdout, result.Stderr, duration)
		}
	}
}

func (r *REPL) confirm(prompt string) bool {
	r.print(prompt)
	line, err := r.reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.ToLower(strings.TrimSpace(line))

	if strings.Contains(prompt, "[Y/n]") {
		return line == "" || line == "y" || line == "yes"
	}
	return line == "y" || line == "yes"
}

// applySafetyMode syncs the safety checker mode with the config mode string
func (r *REPL) applySafetyMode() {
	switch r.cfg.Mode {
	case "ultra-safe":
		r.safety.Mode = safety.ModeUltraSafe
	case "yolo":
		r.safety.Mode = safety.ModeYolo
	default:
		r.safety.Mode = safety.ModeNormal
	}
}

func (r *REPL) printWelcome() {
	r.println("lazy-ai-cli - Natural language to shell commands")
	r.printf("Type %shelp for available commands, or enter a request.\n", r.cfg.Prefix)
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

func (r *REPL) logError(context string, err error) {
	if r.log != nil {
		r.log.LogError(context, err)
	}
}

func (r *REPL) logBlocked(input, command, reason string) {
	if r.log != nil {
		r.log.LogBlocked(input, command, reason)
	}
}
