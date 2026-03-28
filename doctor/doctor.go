package doctor

import (
	"fmt"
	"os"

	"lazy-cli/config"
	"lazy-cli/llm"
)

// RunDoctor runs all diagnostic checks and reports results
func RunDoctor(args []string) int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config error: %v\n", err)
	}

	fmt.Println("=== lazy-cli doctor ===")
	fmt.Println()

	allOK := true

	// 1. Config file
	r := CheckConfig(cfg.FilePath())
	fmt.Println(r)
	if !r.OK {
		allOK = false
	}

	// 2. Directories
	dirCheck := checkDirs()
	fmt.Println(dirCheck)
	if !dirCheck.OK {
		allOK = false
	}

	// 3. llama-server binary
	srvCheck := CheckServerPath(cfg.ServerPath)
	fmt.Println(srvCheck)
	if !srvCheck.OK {
		allOK = false
	}

	// 4. Model file
	mdlCheck := CheckModelPath(cfg.ModelPath)
	fmt.Println(mdlCheck)
	if !mdlCheck.OK {
		allOK = false
	}

	// 5. Port
	portCheck := CheckPort(cfg.Port)
	fmt.Println(portCheck)
	if !portCheck.OK {
		allOK = false
	}

	// 6. Server launch test (only if everything above passed)
	if srvCheck.OK && mdlCheck.OK && portCheck.OK {
		launchCheck := testServerLaunch(cfg)
		fmt.Println(launchCheck)
		if !launchCheck.OK {
			allOK = false
		}
	} else {
		fmt.Println("  [SKIP] server launch: fix above issues first")
	}

	// Summary
	fmt.Println()
	if allOK {
		fmt.Println("All checks passed. Ready to run.")
	} else {
		fmt.Println("Some checks failed. Fix the issues above, then re-run 'lazy-ai doctor'.")
	}

	if allOK {
		return 0
	}
	return 1
}

func checkDirs() CheckResult {
	home, err := os.UserHomeDir()
	if err != nil {
		return CheckResult{"directories", false, fmt.Sprintf("cannot determine home: %v", err)}
	}
	dirs := []string{
		home + "/.lazy-cli",
		home + "/.lazy-cli/logs",
	}
	for _, d := range dirs {
		info, err := os.Stat(d)
		if err != nil {
			return CheckResult{"directories", false, fmt.Sprintf("missing: %s", d)}
		}
		if !info.IsDir() {
			return CheckResult{"directories", false, fmt.Sprintf("not a directory: %s", d)}
		}
	}
	return CheckResult{"directories", true, "OK"}
}

func testServerLaunch(cfg *config.Config) CheckResult {
	server := llm.NewServer(cfg.ServerPath, cfg.ModelPath)
	server.Port = cfg.Port

	if err := server.Start(); err != nil {
		return CheckResult{"server launch", false, err.Error()}
	}
	defer server.Stop()

	if err := HealthCheck("127.0.0.1", cfg.Port); err != nil {
		return CheckResult{"API health", false, err.Error()}
	}

	return CheckResult{"server launch + API", true, "server started, health check passed"}
}
