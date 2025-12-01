package log

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name    string
		cfg     LogConfig
		wantErr bool
	}{
		{
			name: "console output",
			cfg: LogConfig{
				Level:  "info",
				Output: "console",
			},
			wantErr: false,
		},
		{
			name: "debug level",
			cfg: LogConfig{
				Level:  "debug",
				Output: "console",
			},
			wantErr: false,
		},
		{
			name: "invalid level defaults to info",
			cfg: LogConfig{
				Level:  "invalid",
				Output: "console",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if logger == nil {
				t.Error("expected logger to be non-nil")
			}
			logger.Close()
		})
	}
}

func TestLogger_SetLevel(t *testing.T) {
	logger, err := NewLogger(LogConfig{
		Level:  "info",
		Output: "console",
	})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	if logger.GetLevel() != "info" {
		t.Errorf("expected level 'info', got %s", logger.GetLevel())
	}

	err = logger.SetLevel("debug")
	if err != nil {
		t.Errorf("SetLevel() error = %v", err)
	}

	if logger.GetLevel() != "debug" {
		t.Errorf("expected level 'debug', got %s", logger.GetLevel())
	}

	err = logger.SetLevel("error")
	if err != nil {
		t.Errorf("SetLevel() error = %v", err)
	}

	if logger.GetLevel() != "error" {
		t.Errorf("expected level 'error', got %s", logger.GetLevel())
	}
}

func TestLogger_WithModule(t *testing.T) {
	logger, err := NewLogger(LogConfig{
		Level:  "info",
		Output: "console",
	})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	moduleLogger := logger.WithModule("collector")
	if moduleLogger == nil {
		t.Error("expected module logger to be non-nil")
	}

	moduleLogger.Info("test message from module")
}

func TestLogger_WithField(t *testing.T) {
	logger, err := NewLogger(LogConfig{
		Level:  "info",
		Output: "console",
	})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	fieldLogger := logger.WithField("request_id", "12345")
	if fieldLogger == nil {
		t.Error("expected field logger to be non-nil")
	}

	fieldLogger.Info("test message with field")
}

func TestLogger_FileOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewLogger(LogConfig{
		Level:      "info",
		Output:     "file",
		FilePath:   logPath,
		MaxSizeMB:  1,
		MaxBackups: 1,
	})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.Info("test message 1")
	logger.Info("test message 2", zap.String("key", "value"))
	logger.Sync()
	logger.Close()

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("expected log file to exist")
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test message 1") {
		t.Error("expected log file to contain 'test message 1'")
	}

	if !strings.Contains(string(content), "test message 2") {
		t.Error("expected log file to contain 'test message 2'")
	}
}

func TestLogger_BothOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := NewLogger(LogConfig{
		Level:      "info",
		Output:     "both",
		FilePath:   logPath,
		MaxSizeMB:  1,
		MaxBackups: 1,
	})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	logger.Info("both output test")
	logger.Sync()
	logger.Close()

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("expected log file to exist")
	}
}

func TestGlobalLogger(t *testing.T) {
	err := Init(LogConfig{
		Level:  "debug",
		Output: "console",
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	Debug("global debug message")
	Info("global info message")
	Warn("global warn message")
	Error("global error message")

	Debugf("debug: %s", "formatted")
	Infof("info: %s", "formatted")

	err = SetGlobalLevel("warn")
	if err != nil {
		t.Errorf("SetGlobalLevel() error = %v", err)
	}
}

func TestLogger_LogLevels(t *testing.T) {
	logger, err := NewLogger(LogConfig{
		Level:  "debug",
		Output: "console",
	})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}
	defer logger.Close()

	logger.Debug("debug message", zap.Int("count", 1))
	logger.Info("info message", zap.String("key", "value"))
	logger.Warn("warn message", zap.Bool("flag", true))
	logger.Error("error message", zap.Error(nil))

	logger.Debugf("debug: %d", 1)
	logger.Infof("info: %s", "test")
	logger.Warnf("warn: %v", true)
	logger.Errorf("error: %s", "test error")
}
