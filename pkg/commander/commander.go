package commander

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/google/shlex"
)

// Config holds the commander configuration
type Config struct {
	// AllowedCommands is a list of allowed command prefixes (empty means allow all)
	AllowedCommands []string
	// BlockedCommands is a list of blocked command prefixes
	BlockedCommands []string
	// DefaultTimeout is the default command timeout
	DefaultTimeout time.Duration
	// Shell is the shell to use for command execution
	Shell string
	// ShellArg is the argument to pass to the shell for command execution
	ShellArg string
}

// Commander handles command execution with security controls
type Commander struct {
	config Config
}

// Result holds the result of a command execution
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// NewCommander creates a new Commander with the given configuration
func NewCommander(cfg Config) *Commander {
	// Set default shell based on OS
	if cfg.Shell == "" {
		if runtime.GOOS == "windows" {
			cfg.Shell = "cmd"
			cfg.ShellArg = "/c"
		} else {
			cfg.Shell = "/bin/sh"
			cfg.ShellArg = "-c"
		}
	}

	// Set default timeout
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}

	return &Commander{config: cfg}
}

// DefaultBlockedCommands returns a list of commonly dangerous commands
func DefaultBlockedCommands() []string {
	if runtime.GOOS == "windows" {
		return []string{
			"format",
			"del /s",
			"rd /s",
			"rmdir /s",
			"reg delete",
			"net user",
			"net localgroup",
			"shutdown",
			"restart",
		}
	}
	return []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs",
		"dd if=",
		":(){:|:&};:",
		"chmod -R 777 /",
		"chown -R",
		"> /dev/sda",
		"shutdown",
		"reboot",
		"halt",
		"poweroff",
		"init 0",
		"init 6",
	}
}

// ValidateCommand checks if a command is allowed to run
func (c *Commander) ValidateCommand(command string) error {
	command = strings.TrimSpace(command)
	commandLower := strings.ToLower(command)

	// Check blocked commands first
	for _, blocked := range c.config.BlockedCommands {
		blockedLower := strings.ToLower(strings.TrimSpace(blocked))
		if strings.HasPrefix(commandLower, blockedLower) || strings.Contains(commandLower, blockedLower) {
			return fmt.Errorf("command blocked: matches blocked pattern '%s'", blocked)
		}
	}

	// If allowed commands list is empty, allow all (except blocked)
	if len(c.config.AllowedCommands) == 0 {
		return nil
	}

	// Check if command starts with an allowed prefix
	for _, allowed := range c.config.AllowedCommands {
		allowedLower := strings.ToLower(strings.TrimSpace(allowed))
		if strings.HasPrefix(commandLower, allowedLower) {
			return nil
		}
	}

	return fmt.Errorf("command not allowed: does not match any allowed command patterns")
}

// Execute runs a command with the given options
func (c *Commander) Execute(ctx context.Context, command string, workDir string, timeout time.Duration, env map[string]string) *Result {
	start := time.Now()
	result := &Result{}

	// Use default timeout if not specified
	if timeout == 0 {
		timeout = c.config.DefaultTimeout
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, c.config.Shell, c.config.ShellArg, command)

	// Set working directory if specified
	if workDir != "" {
		// Validate working directory exists
		if _, err := os.Stat(workDir); os.IsNotExist(err) {
			result.Error = fmt.Errorf("working directory does not exist: %s", workDir)
			result.Duration = time.Since(start)
			result.ExitCode = -1
			return result
		}
		cmd.Dir = workDir
	}

	// Set environment variables
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.Duration = time.Since(start)

	if err != nil {
		result.Error = err
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			result.ExitCode = -1
			result.Error = fmt.Errorf("command timed out after %s", timeout)
		} else {
			result.ExitCode = -1
		}
	} else {
		result.ExitCode = 0
	}

	return result
}

// GetCommandName extracts the command name from a command string
func GetCommandName(command string) string {
	parts, err := shlex.Split(command)
	if err != nil || len(parts) == 0 {
		// Fallback to simple space split
		parts = strings.Fields(command)
		if len(parts) == 0 {
			return command
		}
	}
	return parts[0]
}

// GetShellInfo returns information about the configured shell
func (c *Commander) GetShellInfo() (shell, shellArg string) {
	return c.config.Shell, c.config.ShellArg
}

// GetDefaultTimeout returns the default timeout
func (c *Commander) GetDefaultTimeout() time.Duration {
	return c.config.DefaultTimeout
}
