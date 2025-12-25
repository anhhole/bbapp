package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Logger handles activity logging
type Logger struct {
	logDir string
	file   *os.File
}

// NewLogger creates new logger
func NewLogger(logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	filename := filepath.Join(logDir, fmt.Sprintf("bbapp_%s.jsonl",
		time.Now().Format("2006-01-02")))

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &Logger{
		logDir: logDir,
		file:   file,
	}, nil
}

// LogGift logs a gift event
func (l *Logger) LogGift(bigoRoomId, nickname, giftName string, value int64) error {
	entry := map[string]interface{}{
		"timestamp":  time.Now().Unix(),
		"type":       "GIFT",
		"bigoRoomId": bigoRoomId,
		"nickname":   nickname,
		"giftName":   giftName,
		"value":      value,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = l.file.Write(append(data, '\n'))
	return err
}

// Close closes logger
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
