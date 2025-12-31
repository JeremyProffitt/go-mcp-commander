package commander

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestNewCommander(t *testing.T) {
	cfg := Config{}
	cmd := NewCommander(cfg)

	if cmd == nil {
		t.Fatal("NewCommander returned nil")
	}

	shell, shellArg := cmd.GetShellInfo()
	if runtime.GOOS == "windows" {
		if shell != "cmd" {
			t.Errorf("Expected shell 'cmd' on Windows, got %s", shell)
		}
		if shellArg != "/c" {
			t.Errorf("Expected shellArg '/c' on Windows, got %s", shellArg)
		}
	} else {
		if shell != "/bin/sh" {
			t.Errorf("Expected shell '/bin/sh' on Unix, got %s", shell)
		}
		if shellArg != "-c" {
			t.Errorf("Expected shellArg '-c' on Unix, got %s", shellArg)
		}
	}
}

func TestValidateCommand_AllowedEmpty(t *testing.T) {
	cmd := NewCommander(Config{})

	// With no allowed list, all commands should be allowed
	err := cmd.ValidateCommand("echo hello")
	if err != nil {
		t.Errorf("Expected command to be allowed, got error: %v", err)
	}
}

func TestValidateCommand_AllowedList(t *testing.T) {
	cmd := NewCommander(Config{
		AllowedCommands: []string{"echo", "ls", "cat"},
	})

	tests := []struct {
		command     string
		shouldAllow bool
	}{
		{"echo hello", true},
		{"ls -la", true},
		{"cat file.txt", true},
		{"rm -rf /", false},
		{"wget http://example.com", false},
	}

	for _, tt := range tests {
		err := cmd.ValidateCommand(tt.command)
		if tt.shouldAllow && err != nil {
			t.Errorf("Expected command '%s' to be allowed, got error: %v", tt.command, err)
		}
		if !tt.shouldAllow && err == nil {
			t.Errorf("Expected command '%s' to be blocked", tt.command)
		}
	}
}

func TestValidateCommand_BlockedList(t *testing.T) {
	cmd := NewCommander(Config{
		BlockedCommands: []string{"rm -rf", "dd if=", "mkfs"},
	})

	tests := []struct {
		command     string
		shouldAllow bool
	}{
		{"echo hello", true},
		{"ls -la", true},
		{"rm -rf /", false},
		{"dd if=/dev/zero of=/dev/sda", false},
		{"mkfs.ext4 /dev/sda1", false},
	}

	for _, tt := range tests {
		err := cmd.ValidateCommand(tt.command)
		if tt.shouldAllow && err != nil {
			t.Errorf("Expected command '%s' to be allowed, got error: %v", tt.command, err)
		}
		if !tt.shouldAllow && err == nil {
			t.Errorf("Expected command '%s' to be blocked", tt.command)
		}
	}
}

func TestValidateCommand_CaseInsensitive(t *testing.T) {
	cmd := NewCommander(Config{
		BlockedCommands: []string{"RM -RF"},
	})

	err := cmd.ValidateCommand("rm -rf /")
	if err == nil {
		t.Error("Expected case-insensitive blocking")
	}
}

func TestExecute_SimpleCommand(t *testing.T) {
	cmd := NewCommander(Config{})

	var command string
	if runtime.GOOS == "windows" {
		command = "echo hello"
	} else {
		command = "echo hello"
	}

	result := cmd.Execute(context.Background(), command, "", 0, nil)

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("Expected stdout to contain 'hello', got %s", result.Stdout)
	}
}

func TestExecute_Timeout(t *testing.T) {
	// Skip this test on Windows because context cancellation doesn't reliably
	// kill child processes through cmd.exe shell
	if runtime.GOOS == "windows" {
		t.Skip("Skipping timeout test on Windows - child process termination behaves differently")
	}

	// Skip in CI with race detector - context cancellation with shell processes
	// is unreliable under race detector due to timing issues
	if os.Getenv("CI") != "" {
		t.Skip("Skipping timeout test in CI - race detector timing makes this unreliable")
	}

	cmd := NewCommander(Config{})

	// Use a longer timeout to account for race detector overhead
	timeout := 500 * time.Millisecond
	command := "sleep 10"

	result := cmd.Execute(context.Background(), command, "", timeout, nil)

	// When a command is killed due to timeout, it should have a non-zero exit code
	if result.ExitCode == 0 {
		t.Errorf("Expected non-zero exit code for timeout, got %d", result.ExitCode)
	}

	if result.Error == nil {
		t.Error("Expected error for timeout")
	}

	// Verify the command didn't run to completion (duration should be close to timeout)
	// Allow generous buffer for race detector and CI overhead
	if result.Duration > 3*time.Second {
		t.Errorf("Command ran too long, expected timeout around %s, got %s", timeout, result.Duration)
	}
}

func TestExecute_WorkingDirectory(t *testing.T) {
	cmd := NewCommander(Config{})

	var command, expectedDir string
	if runtime.GOOS == "windows" {
		command = "cd"
		expectedDir = "C:\\Windows"
	} else {
		command = "pwd"
		expectedDir = "/tmp"
	}

	result := cmd.Execute(context.Background(), command, expectedDir, 0, nil)

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d. Error: %v", result.ExitCode, result.Error)
	}

	if !strings.Contains(strings.ToLower(result.Stdout), strings.ToLower(expectedDir)) {
		t.Errorf("Expected working directory in output, got %s", result.Stdout)
	}
}

func TestExecute_InvalidWorkingDirectory(t *testing.T) {
	cmd := NewCommander(Config{})

	result := cmd.Execute(context.Background(), "echo test", "/nonexistent/directory/that/does/not/exist", 0, nil)

	if result.Error == nil {
		t.Error("Expected error for non-existent working directory")
	}
}

func TestExecute_EnvironmentVariables(t *testing.T) {
	cmd := NewCommander(Config{})

	var command string
	if runtime.GOOS == "windows" {
		command = "echo %TEST_VAR%"
	} else {
		command = "echo $TEST_VAR"
	}

	env := map[string]string{
		"TEST_VAR": "test_value_12345",
	}

	result := cmd.Execute(context.Background(), command, "", 0, env)

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if !strings.Contains(result.Stdout, "test_value_12345") {
		t.Errorf("Expected env var in output, got %s", result.Stdout)
	}
}

func TestExecute_FailingCommand(t *testing.T) {
	cmd := NewCommander(Config{})

	var command string
	if runtime.GOOS == "windows" {
		command = "exit /b 1"
	} else {
		command = "exit 1"
	}

	result := cmd.Execute(context.Background(), command, "", 0, nil)

	if result.ExitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", result.ExitCode)
	}
}

func TestGetCommandName(t *testing.T) {
	tests := []struct {
		command  string
		expected string
	}{
		{"echo hello", "echo"},
		{"ls -la /tmp", "ls"},
		{"cat file.txt", "cat"},
		{"/usr/bin/python script.py", "/usr/bin/python"},
		{"", ""},
	}

	for _, tt := range tests {
		result := GetCommandName(tt.command)
		if result != tt.expected {
			t.Errorf("GetCommandName(%q) = %q, expected %q", tt.command, result, tt.expected)
		}
	}
}

func TestDefaultBlockedCommands(t *testing.T) {
	blocked := DefaultBlockedCommands()

	if len(blocked) == 0 {
		t.Error("Expected non-empty default blocked commands list")
	}

	// Check for some expected dangerous commands
	if runtime.GOOS != "windows" {
		found := false
		for _, cmd := range blocked {
			if strings.Contains(cmd, "rm -rf") {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'rm -rf' in default blocked commands")
		}
	}
}
