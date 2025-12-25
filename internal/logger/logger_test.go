package logger_test

import (
	"os"
	"testing"
	"bbapp/internal/logger"
)

func TestLogger_Log(t *testing.T) {
	tempDir := t.TempDir()

	log, err := logger.NewLogger(tempDir)
	if err != nil {
		t.Fatalf("NewLogger failed: %v", err)
	}
	defer log.Close()

	err = log.LogGift("12345", "user1", "Rose", 100)
	if err != nil {
		t.Fatalf("LogGift failed: %v", err)
	}

	// Verify file exists
	files, _ := os.ReadDir(tempDir)
	if len(files) == 0 {
		t.Fatal("Expected log file to be created")
	}
}
