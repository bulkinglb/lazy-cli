package safety

import (
	"regexp"
	"strings"
)

// RiskLevel indicates how dangerous a command is
type RiskLevel int

const (
	Safe      RiskLevel = iota // Green light
	Caution                    // Needs confirmation
	Dangerous                  // Strong warning + confirmation
	Blocked                    // Never execute
)

func (r RiskLevel) String() string {
	return [...]string{"safe", "caution", "dangerous", "blocked"}[r]
}

// Mode controls how strict the safety checks are
type Mode int

const (
	ModeUltraSafe Mode = iota // Confirm everything, block more
	ModeNormal                // Default behavior
	ModeYolo                  // Only block catastrophic commands
)

// Result of analyzing a command
type Result struct {
	Level   RiskLevel
	Reason  string
	Command string
}

// Checker analyzes commands for safety
type Checker struct {
	Mode Mode
}

// NewChecker creates a safety checker with default mode
func NewChecker() *Checker {
	return &Checker{Mode: ModeNormal}
}

// Check analyzes a command and returns a safety result
func (c *Checker) Check(cmd string) Result {
	cmd = strings.TrimSpace(cmd)

	// Always blocked - catastrophic commands
	if reason := c.matchBlocked(cmd); reason != "" {
		return Result{Blocked, reason, cmd}
	}

	// Dangerous commands
	if reason := c.matchDangerous(cmd); reason != "" {
		if c.Mode == ModeYolo {
			return Result{Caution, reason, cmd}
		}
		return Result{Dangerous, reason, cmd}
	}

	// Caution commands
	if reason := c.matchCaution(cmd); reason != "" {
		if c.Mode == ModeUltraSafe {
			return Result{Dangerous, reason, cmd}
		}
		if c.Mode == ModeYolo {
			return Result{Safe, reason, cmd}
		}
		return Result{Caution, reason, cmd}
	}

	// Ultra safe mode: confirm everything
	if c.Mode == ModeUltraSafe {
		return Result{Caution, "ultra-safe mode requires confirmation", cmd}
	}

	return Result{Safe, "", cmd}
}

// ShouldExecute returns true if the command can run (possibly after confirmation)
func (r Result) ShouldExecute() bool {
	return r.Level != Blocked
}

// NeedsConfirmation returns true if user must confirm
func (r Result) NeedsConfirmation(mode Mode) bool {
	switch r.Level {
	case Blocked:
		return false // Can't execute anyway
	case Dangerous:
		return true
	case Caution:
		return mode != ModeYolo
	case Safe:
		return mode == ModeUltraSafe
	}
	return false
}

var blockedPatterns = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`rm\s+(-[rfRv]+\s+)*(/|/\*|\*/)`), "removes root filesystem"},
	{regexp.MustCompile(`rm\s+-[rfRv]*\s+/`), "removes root filesystem"},
	{regexp.MustCompile(`mkfs\.`), "formats filesystem"},
	{regexp.MustCompile(`dd\s+.*of=/dev/[sh]d[a-z]`), "overwrites disk device"},
	{regexp.MustCompile(`>\s*/dev/[sh]d[a-z]`), "overwrites disk device"},
	{regexp.MustCompile(`:\(\)\s*\{\s*:\|\:&\s*\}\s*;`), "fork bomb"},
	{regexp.MustCompile(`chmod\s+(-[Rv]+\s+)*777\s+/`), "opens root permissions"},
	{regexp.MustCompile(`chown\s+(-[Rv]+\s+)*.*\s+/\s*$`), "changes root ownership"},
}

var dangerousPatterns = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`sudo\s+rm\s`), "sudo remove"},
	{regexp.MustCompile(`rm\s+(-[rfRv]+\s+)+`), "recursive/force delete"},
	{regexp.MustCompile(`>\s*/etc/`), "overwrites system config"},
	{regexp.MustCompile(`curl.*\|\s*(sudo\s+)?sh`), "pipes remote script to shell"},
	{regexp.MustCompile(`wget.*\|\s*(sudo\s+)?sh`), "pipes remote script to shell"},
	{regexp.MustCompile(`shutdown`), "system shutdown"},
	{regexp.MustCompile(`reboot`), "system reboot"},
	{regexp.MustCompile(`init\s+[06]`), "system shutdown/reboot"},
	{regexp.MustCompile(`systemctl\s+(stop|disable|mask)\s`), "disables system service"},
}

var cautionPatterns = []struct {
	pattern *regexp.Regexp
	reason  string
}{
	{regexp.MustCompile(`^sudo\s`), "runs as root"},
	{regexp.MustCompile(`rm\s`), "deletes files"},
	{regexp.MustCompile(`mv\s`), "moves/renames files"},
	{regexp.MustCompile(`cp\s+-[rf]`), "copies recursively"},
	{regexp.MustCompile(`chmod\s`), "changes permissions"},
	{regexp.MustCompile(`chown\s`), "changes ownership"},
	{regexp.MustCompile(`apt\s+(remove|purge)`), "removes packages"},
	{regexp.MustCompile(`pacman\s+-R`), "removes packages"},
	{regexp.MustCompile(`pip\s+uninstall`), "uninstalls packages"},
	{regexp.MustCompile(`npm\s+(uninstall|rm)`), "uninstalls packages"},
}

func (c *Checker) matchBlocked(cmd string) string {
	for _, p := range blockedPatterns {
		if p.pattern.MatchString(cmd) {
			return p.reason
		}
	}
	return ""
}

func (c *Checker) matchDangerous(cmd string) string {
	for _, p := range dangerousPatterns {
		if p.pattern.MatchString(cmd) {
			return p.reason
		}
	}
	return ""
}

func (c *Checker) matchCaution(cmd string) string {
	for _, p := range cautionPatterns {
		if p.pattern.MatchString(cmd) {
			return p.reason
		}
	}
	return ""
}
