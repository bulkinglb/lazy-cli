package repl

// CommandType distinguishes special commands from AI input
type CommandType int

const (
	CommandTypeAI CommandType = iota // Normal input → send to LLM
	CommandTypeInternal              // Special %&command → handle internally
	CommandTypeDirect                // !command → execute directly
)

// Command represents parsed user input
type Command struct {
	Type    CommandType
	Name    string // For internal commands: "config", "status", etc.
	Args    string // Any arguments after the command name
	RawText string // Original input text
}

// CommandHandler handles a specific internal command
type CommandHandler func(args string) error

// CommandRegistry manages internal command handlers
type CommandRegistry struct {
	handlers     map[string]CommandHandler
	descriptions map[string]string
}

// NewCommandRegistry creates an empty registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		handlers:     make(map[string]CommandHandler),
		descriptions: make(map[string]string),
	}
}

// Register adds a command handler
func (r *CommandRegistry) Register(name, description string, handler CommandHandler) {
	r.handlers[name] = handler
	r.descriptions[name] = description
}

// Get returns a handler if it exists
func (r *CommandRegistry) Get(name string) (CommandHandler, bool) {
	h, ok := r.handlers[name]
	return h, ok
}

// List returns all registered command names with descriptions
func (r *CommandRegistry) List() map[string]string {
	result := make(map[string]string, len(r.descriptions))
	for k, v := range r.descriptions {
		result[k] = v
	}
	return result
}
