package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client communicates with llama-server HTTP API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a client for the given server
func NewClient(server *Server) *Client {
	return &Client{
		baseURL: server.URL(),
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // LLM responses can be slow
		},
	}
}

// CompletionRequest is the request body for /completion
type CompletionRequest struct {
	Prompt      string   `json:"prompt"`
	MaxTokens   int      `json:"n_predict,omitempty"`
	Temperature float64  `json:"temperature,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// CompletionResponse is the response from /completion
type CompletionResponse struct {
	Content string `json:"content"`
}

// Complete sends a prompt and returns the generated text
func (c *Client) Complete(prompt string) (string, error) {
	return c.completeWithOptions(prompt, 256, []string{"\n\n", "```"})
}

func (c *Client) completeWithOptions(prompt string, maxTokens int, stop []string) (string, error) {
	req := CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   maxTokens,
		Temperature: 0.1,
		Stop:        stop,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/completion",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("server error %d: %s", resp.StatusCode, string(respBody))
	}

	var result CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return result.Content, nil
}

// GenerateUndo generates a shell command that reverses the effect of the given command
func (c *Client) GenerateUndo(command string) (string, error) {
	result, err := c.Complete(buildUndoPrompt(command))
	if err != nil {
		return "", err
	}
	return cleanCommand(result), nil
}

// ExplainCommand returns a plain English explanation of what a shell command does
func (c *Client) ExplainCommand(command string) (string, error) {
	result, err := c.completeWithOptions(buildExplainPrompt(command), 512, []string{"---", "```"})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

func buildUndoPrompt(command string) string {
	return `You are a shell command reversal assistant.
Given a shell command that was already executed, output the exact shell command to undo its effect.

RULES:
- Output ONLY the reversal command, nothing else
- No explanations, no markdown, no code fences
- If the command cannot be undone, output exactly: CANNOT_UNDO

EXAMPLES:

Command: mv notes.txt archive/notes.txt
Undo: mv archive/notes.txt notes.txt

Command: mkdir -p /tmp/mydir
Undo: rmdir /tmp/mydir

Command: cp src.go src.go.bak
Undo: rm src.go.bak

Command: rm important.txt
Undo: CANNOT_UNDO

Command: echo hello > out.txt
Undo: CANNOT_UNDO

Command: ` + command + `
Undo:`
}

func buildExplainPrompt(command string) string {
	return `You are a shell command explainer. Explain what the following shell command does in plain English.
Break down each component (flags, arguments, pipes). Be concise but complete. No markdown.

EXAMPLES:

Command: ls -la /tmp
Explanation: Lists all files in /tmp including hidden ones. The -l flag shows a detailed view with permissions, owner, size, and date. The -a flag includes hidden files (names starting with a dot).

Command: find . -name "*.go" | xargs grep -l "TODO"
Explanation: Recursively finds all .go files in the current directory, then searches each one for the text TODO, printing only the filenames that contain it.

Command: ` + command + `
Explanation:`
}

// GenerateCommand uses the LLM to convert natural language to a shell command
func (c *Client) GenerateCommand(input string) (string, error) {
	prompt := buildPrompt(input)
	result, err := c.Complete(prompt)
	if err != nil {
		return "", err
	}
	return cleanCommand(result), nil
}

func buildPrompt(input string) string {
	return `You are a Linux shell command generator for an interactive local CLI assistant.

Your task is to convert the user's request into a shell command or a short shell command chain.

SYSTEM CONTEXT:
- Current working directory: {CWD}
- Home directory: {HOME}
- Known path aliases:
{PATH_ALIASES}

RULES:
- Output ONLY the shell command
- No explanations, no markdown, no code fences
- Prefer standard Linux tools (find, ls, grep, cat, curl, apt, systemctl, ip, ss, ps, mkdir, cp, mv, tar, etc.)
- Prefer safe and minimal commands
- Do NOT invent paths, directories, filenames, services, or package names
- Resolve relative paths against the current working directory
- If the user refers to a known alias like "projects", use the matching known path alias
- If the request is ambiguous or unsafe, output exactly:
echo "Unable to generate safe command"
- Avoid destructive commands unless explicitly requested
- Avoid sudo unless it is clearly required
- Use a single command if possible, but a short chained command is allowed if necessary
- Never output comments

EXAMPLES:

User: list all files including hidden
Command: ls -la

User: find all python files in current directory
Command: find . -type f -name "*.py"

User: show disk usage
Command: df -h

User: check if docker is running
Command: systemctl status docker

User: create a directory called projects
Command: mkdir -p projects

User: find every file ending with .py in projects
Command: find /home/lukas/projects -type f -name "*.py"

User: delete everything
Command: echo "Unable to generate safe command"

User: ` + input + `
Command:`
}

func cleanCommand(raw string) string {
	// Remove common artifacts from LLM output
	cmd := strings.TrimSpace(raw)

	// Remove markdown code blocks if present
	cmd = strings.TrimPrefix(cmd, "```bash")
	cmd = strings.TrimPrefix(cmd, "```sh")
	cmd = strings.TrimPrefix(cmd, "```")
	cmd = strings.TrimSuffix(cmd, "```")

	// Take only first line (command should be single line)
	if idx := strings.Index(cmd, "\n"); idx != -1 {
		cmd = cmd[:idx]
	}

	return strings.TrimSpace(cmd)
}
