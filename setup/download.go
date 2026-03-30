package setup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	llamaCppRepo   = "ggerganov/llama.cpp"
	gemmaModelFile = "gemma-3-1b-it-Q4_K_M.gguf"
	gemmaModelURL  = "https://huggingface.co/bartowski/gemma-3-1b-it-GGUF/resolve/main/gemma-3-1b-it-Q4_K_M.gguf"
)

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

func getLatestLlamaRelease() (*githubRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/" + llamaCppRepo + "/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func findLlamaAsset(release *githubRelease) (*githubAsset, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	var osPart, archPart string
	switch goos {
	case "linux":
		osPart = "ubuntu"
	case "darwin":
		osPart = "macos"
	default:
		return nil, fmt.Errorf("unsupported OS: %s", goos)
	}
	switch goarch {
	case "amd64":
		archPart = "x64"
	case "arm64":
		archPart = "arm64"
	default:
		return nil, fmt.Errorf("unsupported arch: %s", goarch)
	}

	// Try exact name first: llama-<tag>-bin-<os>-<arch>.zip
	exact := fmt.Sprintf("llama-%s-bin-%s-%s.zip", release.TagName, osPart, archPart)
	for i, a := range release.Assets {
		if a.Name == exact {
			return &release.Assets[i], nil
		}
	}

	// Fallback: any zip matching os+arch keywords
	for i, a := range release.Assets {
		if strings.Contains(a.Name, osPart) &&
			strings.Contains(a.Name, archPart) &&
			strings.HasSuffix(a.Name, ".zip") {
			return &release.Assets[i], nil
		}
	}

	return nil, fmt.Errorf("no prebuilt binary for %s/%s in release %s", goos, goarch, release.TagName)
}

// DownloadLlamaServer fetches the latest llama.cpp release, extracts llama-server,
// and installs it to binDir. Returns the path to the installed binary.
func DownloadLlamaServer(binDir string) (string, error) {
	fmt.Println("  Fetching latest llama.cpp release...")
	release, err := getLatestLlamaRelease()
	if err != nil {
		return "", fmt.Errorf("failed to fetch llama.cpp release info: %w", err)
	}
	fmt.Printf("  Latest version: %s\n", release.TagName)

	asset, err := findLlamaAsset(release)
	if err != nil {
		return "", err
	}

	fmt.Printf("  Downloading %s...\n", asset.Name)
	tmpDir, err := os.MkdirTemp("", "llama-cpp-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	zipPath := filepath.Join(tmpDir, asset.Name)
	if err := downloadWithProgress(asset.BrowserDownloadURL, zipPath); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	fmt.Println("  Extracting...")
	serverBin, err := extractLlamaServer(zipPath, tmpDir)
	if err != nil {
		return "", fmt.Errorf("extraction failed: %w", err)
	}

	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", err
	}
	destPath := filepath.Join(binDir, "llama-server")
	if err := copyFile(serverBin, destPath); err != nil {
		return "", err
	}
	if err := os.Chmod(destPath, 0755); err != nil {
		return "", err
	}
	return destPath, nil
}

// DownloadGemmaModel downloads the Gemma 3 1B GGUF model to modelsDir.
// Skips the download if the file already exists. Returns the path to the model file.
func DownloadGemmaModel(modelsDir string) (string, error) {
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return "", err
	}
	destPath := filepath.Join(modelsDir, gemmaModelFile)

	if info, err := os.Stat(destPath); err == nil && info.Size() > 0 {
		fmt.Printf("  Model already present at %s\n", destPath)
		return destPath, nil
	}

	fmt.Printf("  Downloading %s (~800 MB)...\n", gemmaModelFile)
	if err := downloadWithProgress(gemmaModelURL, destPath); err != nil {
		os.Remove(destPath) // remove partial download
		return "", fmt.Errorf("model download failed: %w", err)
	}
	return destPath, nil
}

// extractLlamaServer finds and extracts the llama-server binary from a zip archive.
func extractLlamaServer(zipPath, destDir string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		base := filepath.Base(f.Name)
		if base == "llama-server" || base == "llama-server.exe" {
			dest := filepath.Join(destDir, base)
			if err := extractZipEntry(f, dest); err != nil {
				return "", err
			}
			return dest, nil
		}
	}
	return "", fmt.Errorf("llama-server binary not found in archive")
}

func extractZipEntry(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func downloadWithProgress(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	lastPrint := time.Now()

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			if time.Since(lastPrint) > 300*time.Millisecond {
				if total > 0 {
					pct := float64(downloaded) / float64(total) * 100
					fmt.Printf("\r  %.1f%% (%s / %s)   ", pct, formatBytes(downloaded), formatBytes(total))
				} else {
					fmt.Printf("\r  %s downloaded   ", formatBytes(downloaded))
				}
				lastPrint = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	if total > 0 {
		fmt.Printf("\r  100.0%% (%s / %s)   \n", formatBytes(downloaded), formatBytes(total))
	} else {
		fmt.Printf("\r  %s downloaded   \n", formatBytes(downloaded))
	}
	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
