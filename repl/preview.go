package repl

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var sedInplace = regexp.MustCompile(`\bsed\b.*\s(-i|--in-place)\b`)
var stripInplace = regexp.MustCompile(`\s(-i\S*|--in-place\S*)`)

func tryDiffPreview(command string) string {
	if sedInplace.MatchString(command) {
		return sedDiffPreview(command)
	}
	return ""
}

func sedDiffPreview(command string) string {
	stripped := stripInplace.ReplaceAllString(command, "")
	out, err := exec.Command("sh", "-c", stripped).Output()
	if err != nil {
		return ""
	}

	file := extractLastArg(command)
	if file == "" {
		return ""
	}
	original, err := os.ReadFile(file)
	if err != nil {
		return ""
	}

	diff := unifiedDiff(string(original), string(out), file)
	if diff == "" {
		return "Diff preview: no changes."
	}
	return fmt.Sprintf("Diff preview:\n%s", diff)
}

func extractLastArg(cmd string) string {
	parts := strings.Fields(cmd)
	for i := len(parts) - 1; i >= 0; i-- {
		if !strings.HasPrefix(parts[i], "-") {
			return parts[i]
		}
	}
	return ""
}

func unifiedDiff(original, modified, label string) string {
	f, err := os.CreateTemp("", "lazy-preview-*")
	if err != nil {
		return ""
	}
	defer os.Remove(f.Name())
	f.WriteString(original)
	f.Close()

	cmd := exec.Command("diff", "-u",
		"--label", label+"  (original)",
		"--label", label+"  (modified)",
		f.Name(), "-")
	cmd.Stdin = bytes.NewBufferString(modified)
	out, _ := cmd.Output() // diff exits 1 when differences exist — ignore error
	return string(out)
}
