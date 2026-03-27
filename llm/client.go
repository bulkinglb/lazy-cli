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
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"n_predict,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// CompletionResponse is the response from /completion
type CompletionResponse struct {
	Content string `json:"content"`
}

// Complete sends a prompt and returns the generated text
func (c *Client) Complete(prompt string) (string, error) {
	req := CompletionRequest{
		Prompt:      prompt,
		MaxTokens:   256,
		Temperature: 0.1, // Low temp for deterministic command generation
		Stop:        []string{"\n\n", "```"},
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
	return `You are a Linux shell command generator. Convert the user request into a single shell command.

RULES:
- Output ONLY the command, nothing else
- No explanations, no markdown, no code blocks
- Use standard Linux tools (coreutils, apt, systemctl, etc.)
- Prefer safe commands (avoid rm -rf, sudo when unnecessary)
- If unsure, output: echo "Unable to generate safe command"

EXAMPLES:
User: list all files including hidden
Command: ls -la

User: find all python files in current directory
Command: find . -name "*.py"

User: show disk usage
Command: df -h

User: check if docker is running
Command: systemctl status docker

User: download a file from url
Command: curl -O https://example.com/file

User: show my ip address
Command: ip addr show

User: count lines in file.txt
Command: wc -l file.txt

User: search for "error" in logs
Command: grep -r "error" /var/log/

User: create a directory called projects
Command: mkdir -p projects

User: show running processes
Command: ps aux

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
