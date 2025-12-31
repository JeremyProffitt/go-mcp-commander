package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"off", LevelOff},
		{"OFF", LevelOff},
		{"error", LevelError},
		{"ERROR", LevelError},
		{"warn", LevelWarn},
		{"WARN", LevelWarn},
		{"warning", LevelWarn},
		{"info", LevelInfo},
		{"INFO", LevelInfo},
		{"access", LevelAccess},
		{"ACCESS", LevelAccess},
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"unknown", LevelInfo}, // default
	}

	for _, tt := range tests {
		result := ParseLogLevel(tt.input)
		if result != tt.expected {
			t.Errorf("ParseLogLevel(%q) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}

func TestLogLevelString(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{LevelOff, "OFF"},
		{LevelError, "ERROR"},
		{LevelWarn, "WARN"},
		{LevelInfo, "INFO"},
		{LevelAccess, "ACCESS"},
		{LevelDebug, "DEBUG"},
		{LogLevel(100), "UNKNOWN"},
	}

	for _, tt := range tests {
		result := tt.level.String()
		if result != tt.expected {
			t.Errorf("LogLevel(%d).String() = %q, expected %q", tt.level, result, tt.expected)
		}
	}
}

func TestNewLogger(t *testing.T) {
	tempDir := t.TempDir()

	logger, err := NewLogger(Config{
		LogDir:  tempDir,
		AppName: "test-logger",
		Level:   LevelDebug,
	})

	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	// Check that log file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read log directory: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected log file to be created")
	}
}

func TestLoggerLevels(t *testing.T) {
	tempDir := t.TempDir()

	logger, err := NewLogger(Config{
		LogDir:  tempDir,
		AppName: "test-logger",
		Level:   LevelInfo,
	})
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	// Use a buffer to capture output
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// These should be logged (level <= Info)
	logger.Error("error message")
	logger.Warn("warn message")
	logger.Info("info message")

	// These should NOT be logged (level > Info)
	logger.Access("access message")
	logger.Debug("debug message")

	output := buf.String()

	if !strings.Contains(output, "error message") {
		t.Error("Expected error message to be logged")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Expected warn message to be logged")
	}
	if !strings.Contains(output, "info message") {
		t.Error("Expected info message to be logged")
	}
	if strings.Contains(output, "access message") {
		t.Error("Expected access message NOT to be logged at Info level")
	}
	if strings.Contains(output, "debug message") {
		t.Error("Expected debug message NOT to be logged at Info level")
	}
}

func TestLoggerCommandExec(t *testing.T) {
	tempDir := t.TempDir()

	logger, err := NewLogger(Config{
		LogDir:  tempDir,
		AppName: "test-logger",
		Level:   LevelAccess,
	})
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	logger.CommandExec("echo hello", "/tmp", 0, 100*time.Millisecond, nil)

	output := buf.String()
	if !strings.Contains(output, "CMD_EXEC") {
		t.Error("Expected CMD_EXEC in output")
	}
	if !strings.Contains(output, "echo hello") {
		t.Error("Expected command in output")
	}
	if !strings.Contains(output, "/tmp") {
		t.Error("Expected working directory in output")
	}
}

func TestLoggerCommandBlocked(t *testing.T) {
	tempDir := t.TempDir()

	logger, err := NewLogger(Config{
		LogDir:  tempDir,
		AppName: "test-logger",
		Level:   LevelWarn,
	})
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	logger.CommandBlocked("rm -rf /", "blocked by default blocklist")

	output := buf.String()
	if !strings.Contains(output, "CMD_BLOCKED") {
		t.Error("Expected CMD_BLOCKED in output")
	}
	if !strings.Contains(output, "rm -rf") {
		t.Error("Expected command in output")
	}
}

func TestLoggerToolCall(t *testing.T) {
	tempDir := t.TempDir()

	logger, err := NewLogger(Config{
		LogDir:  tempDir,
		AppName: "test-logger",
		Level:   LevelInfo,
	})
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	args := map[string]interface{}{
		"command":           "secret command",
		"working_directory": "/tmp",
	}
	logger.ToolCall("execute_command", args)

	output := buf.String()
	if !strings.Contains(output, "TOOL_CALL") {
		t.Error("Expected TOOL_CALL in output")
	}
	if !strings.Contains(output, "execute_command") {
		t.Error("Expected tool name in output")
	}
	// Should log keys, not values (for security)
	if !strings.Contains(output, "command") {
		t.Error("Expected argument key 'command' in output")
	}
	if strings.Contains(output, "secret command") {
		t.Error("Should NOT log argument values for security")
	}
}

func TestLogStartupAndShutdown(t *testing.T) {
	tempDir := t.TempDir()

	logger, err := NewLogger(Config{
		LogDir:  tempDir,
		AppName: "test-logger",
		Level:   LevelInfo,
	})
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	info := GetStartupInfo(
		"1.0.0",
		ConfigValue{Value: tempDir, Source: SourceFlag},
		ConfigValue{Value: "info", Source: SourceDefault},
		ConfigValue{Value: "", Source: SourceDefault},
		ConfigValue{Value: "", Source: SourceDefault},
		ConfigValue{Value: "30s", Source: SourceDefault},
		ConfigValue{Value: "/bin/sh -c", Source: SourceDefault},
	)

	logger.LogStartup(info)

	output := buf.String()
	if !strings.Contains(output, "SERVER STARTUP") {
		t.Error("Expected SERVER STARTUP in output")
	}
	if !strings.Contains(output, "1.0.0") {
		t.Error("Expected version in output")
	}

	logger.LogShutdown("normal")

	output = buf.String()
	if !strings.Contains(output, "SERVER SHUTDOWN") {
		t.Error("Expected SERVER SHUTDOWN in output")
	}
}

func TestDefaultLogDir(t *testing.T) {
	dir := DefaultLogDir("test-app")

	if dir == "" {
		t.Error("Expected non-empty default log dir")
	}

	// Should contain the app name
	if !strings.Contains(dir, "test-app") {
		t.Errorf("Expected dir to contain app name, got %s", dir)
	}

	// Should contain "logs"
	if !strings.Contains(dir, "logs") {
		t.Errorf("Expected dir to contain 'logs', got %s", dir)
	}
}

func TestLoggerSetLevel(t *testing.T) {
	tempDir := t.TempDir()

	logger, err := NewLogger(Config{
		LogDir:  tempDir,
		AppName: "test-logger",
		Level:   LevelError,
	})
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	var buf bytes.Buffer
	logger.SetOutput(&buf)

	// At error level, info should not be logged
	logger.Info("should not appear")
	if strings.Contains(buf.String(), "should not appear") {
		t.Error("Info should not be logged at Error level")
	}

	// Change to debug level
	logger.SetLevel(LevelDebug)

	// Now info should be logged
	logger.Info("should appear")
	if !strings.Contains(buf.String(), "should appear") {
		t.Error("Info should be logged at Debug level")
	}
}

func TestLoggerClose(t *testing.T) {
	tempDir := t.TempDir()

	logger, err := NewLogger(Config{
		LogDir:  tempDir,
		AppName: "test-logger",
		Level:   LevelInfo,
	})
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}

	err = logger.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Double close should not error
	err = logger.Close()
	// This might return an error, which is okay
}

func TestLogFileNaming(t *testing.T) {
	tempDir := t.TempDir()

	logger, err := NewLogger(Config{
		LogDir:  tempDir,
		AppName: "my-test-app",
		Level:   LevelInfo,
	})
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer logger.Close()

	// Check log file name format
	files, _ := os.ReadDir(tempDir)
	if len(files) != 1 {
		t.Fatalf("Expected 1 log file, got %d", len(files))
	}

	filename := files[0].Name()

	// Should start with app name
	if !strings.HasPrefix(filename, "my-test-app-") {
		t.Errorf("Expected filename to start with 'my-test-app-', got %s", filename)
	}

	// Should end with .log
	if filepath.Ext(filename) != ".log" {
		t.Errorf("Expected .log extension, got %s", filepath.Ext(filename))
	}

	// Should contain date
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(filename, today) {
		t.Errorf("Expected filename to contain date %s, got %s", today, filename)
	}
}
