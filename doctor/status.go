package doctor

import (
	"fmt"
	"os"

	"lazy-cli/config"
)

// RunStatus displays the current configuration and file status
func RunStatus(args []string) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config error: %v\n", err)
	}

	fmt.Println("=== lazy-cli status ===")
	fmt.Println()

	// Config
	fmt.Printf("  Config file:   %s", cfg.FilePath())
	if FileExists(cfg.FilePath()) {
		fmt.Println(" (exists)")
	} else {
		fmt.Println(" (missing)")
	}

	// Server binary
	fmt.Printf("  Server path:   %s", valueOr(cfg.ServerPath, "(not set)"))
	if cfg.ServerPath != "" {
		if _, ok := ExecutableExists(cfg.ServerPath); ok {
			fmt.Println(" (found)")
		} else {
			fmt.Println(" (NOT FOUND)")
		}
	} else {
		fmt.Println()
	}

	// Model file
	fmt.Printf("  Model path:    %s", valueOr(cfg.ModelPath, "(not set)"))
	if cfg.ModelPath != "" {
		if FileExists(cfg.ModelPath) {
			if IsGGUF(cfg.ModelPath) {
				fmt.Println(" (valid GGUF)")
			} else {
				fmt.Println(" (exists, NOT GGUF)")
			}
		} else {
			fmt.Println(" (NOT FOUND)")
		}
	} else {
		fmt.Println()
	}

	// Port
	fmt.Printf("  Port:          %d", cfg.Port)
	if PortInUse(cfg.Port) {
		fmt.Println(" (in use)")
	} else {
		fmt.Println(" (available)")
	}

	// Other config
	fmt.Printf("  Mode:          %s\n", cfg.Mode)
	fmt.Printf("  Prefix:        %s\n", cfg.Prefix)
	fmt.Printf("  Logging:       %s\n", boolYesNo(cfg.LogEnabled))
	fmt.Printf("  Log path:      %s\n", cfg.LogPath)

	// Setup validity
	fmt.Println()
	valid := cfg.ServerPath != "" && cfg.ModelPath != ""
	if valid {
		_, srvOK := ExecutableExists(cfg.ServerPath)
		mdlOK := FileExists(cfg.ModelPath) && IsGGUF(cfg.ModelPath)
		if srvOK && mdlOK {
			fmt.Println("  Setup: VALID - ready to run")
		} else {
			fmt.Println("  Setup: INCOMPLETE - run 'lazy-cli doctor' for details")
		}
	} else {
		fmt.Println("  Setup: NOT CONFIGURED - run 'lazy-cli setup' first")
	}

	return 0
}

func valueOr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func boolYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
