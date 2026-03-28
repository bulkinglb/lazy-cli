package doctor

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// CheckResult is a single diagnostic outcome
type CheckResult struct {
	Name   string
	OK     bool
	Detail string
}

func (r CheckResult) String() string {
	mark := "OK"
	if !r.OK {
		mark = "FAIL"
	}
	if r.Detail != "" {
		return fmt.Sprintf("  [%s] %s: %s", mark, r.Name, r.Detail)
	}
	return fmt.Sprintf("  [%s] %s", mark, r.Name)
}

// FileExists checks that a path exists and is a regular file
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// ExecutableExists checks that a binary can be found via PATH or absolute path
func ExecutableExists(path string) (string, bool) {
	// Try absolute/relative path first
	if FileExists(path) {
		return path, true
	}
	// Try PATH lookup
	p, err := exec.LookPath(path)
	if err != nil {
		return "", false
	}
	return p, true
}

// IsGGUF checks if a file starts with the GGUF magic bytes ("GGUF" = 0x46475547 LE)
func IsGGUF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	var magic uint32
	if err := binary.Read(f, binary.LittleEndian, &magic); err != nil {
		return false
	}
	// "GGUF" in little-endian: G=0x47, G=0x47, U=0x55, F=0x46
	return magic == 0x46554747
}

// PortAvailable checks if a TCP port is free to bind on localhost
func PortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// PortInUse returns true if something is already listening on the port
func PortInUse(port int) bool {
	return !PortAvailable(port)
}

// HealthCheck performs an HTTP GET to the llama-server /health endpoint
func HealthCheck(host string, port int) error {
	url := fmt.Sprintf("http://%s:%d/health", host, port)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}
	return nil
}

// CheckServerPath validates the llama-server binary
func CheckServerPath(path string) CheckResult {
	if path == "" {
		return CheckResult{"llama-server", false, "not configured"}
	}
	resolved, found := ExecutableExists(path)
	if !found {
		return CheckResult{"llama-server", false, fmt.Sprintf("not found: %s", path)}
	}
	return CheckResult{"llama-server", true, resolved}
}

// CheckModelPath validates the model file
func CheckModelPath(path string) CheckResult {
	if path == "" {
		return CheckResult{"model file", false, "not configured"}
	}
	if !FileExists(path) {
		return CheckResult{"model file", false, fmt.Sprintf("not found: %s", path)}
	}
	if !IsGGUF(path) {
		return CheckResult{"model file", false, fmt.Sprintf("not a valid GGUF file: %s", path)}
	}
	info, _ := os.Stat(path)
	sizeMB := info.Size() / (1024 * 1024)
	return CheckResult{"model file", true, fmt.Sprintf("%s (%d MB)", path, sizeMB)}
}

// CheckPort validates port configuration
func CheckPort(port int) CheckResult {
	if port < 1 || port > 65535 {
		return CheckResult{"port", false, fmt.Sprintf("invalid: %d", port)}
	}
	if PortInUse(port) {
		return CheckResult{"port", false, fmt.Sprintf("%d is already in use", port)}
	}
	return CheckResult{"port", true, fmt.Sprintf("%d (available)", port)}
}

// CheckConfig validates that the config directory and file exist
func CheckConfig(cfgPath string) CheckResult {
	if cfgPath == "" {
		return CheckResult{"config", false, "no config path"}
	}
	if !FileExists(cfgPath) {
		return CheckResult{"config", false, fmt.Sprintf("not found: %s", cfgPath)}
	}
	return CheckResult{"config", true, cfgPath}
}
