package repl

import "strings"

const directPrefix = "!"

// Parse analyzes user input and returns a Command
// cmdPrefix is the configurable internal command prefix (e.g. "%&")
func Parse(input, cmdPrefix string) Command {
	input = strings.TrimSpace(input)

	if input == "" {
		return Command{Type: CommandTypeAI, RawText: input}
	}

	// Check for internal command prefix
	if strings.HasPrefix(input, cmdPrefix) {
		return parseInternalCommand(input, cmdPrefix)
	}

	// Check for direct command prefix
	if strings.HasPrefix(input, directPrefix) {
		return Command{
			Type:    CommandTypeDirect,
			RawText: strings.TrimPrefix(input, directPrefix),
		}
	}

	// Default: AI input
	return Command{
		Type:    CommandTypeAI,
		RawText: input,
	}
}

func parseInternalCommand(input, prefix string) Command {
	// Remove prefix
	content := strings.TrimPrefix(input, prefix)
	content = strings.TrimSpace(content)

	// Split into command name and args
	parts := strings.SplitN(content, " ", 2)
	name := strings.ToLower(parts[0])

	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	return Command{
		Type:    CommandTypeInternal,
		Name:    name,
		Args:    args,
		RawText: input,
	}
}
