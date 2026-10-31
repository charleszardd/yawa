package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

const maximumLogSize = 1 << 20

func Open() (*slog.Logger, io.Closer, error) {
	cacheDirectory, err := os.UserCacheDir()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve local data directory: %w", err)
	}
	logDirectory := filepath.Join(cacheDirectory, "yawa")
	if err := os.MkdirAll(logDirectory, 0o700); err != nil {
		return nil, nil, fmt.Errorf("create log directory: %w", err)
	}
	logPath := filepath.Join(logDirectory, "yawa.log")
	if err := rotateIfNeeded(logPath); err != nil {
		return nil, nil, err
	}
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %w", err)
	}
	return slog.New(slog.NewJSONHandler(logFile, nil)), logFile, nil
}

func rotateIfNeeded(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect log file: %w", err)
	}
	if info.Size() < maximumLogSize {
		return nil
	}
	backupPath := path + ".1"
	_ = os.Remove(backupPath)
	if err := os.Rename(path, backupPath); err != nil {
		return fmt.Errorf("rotate log file: %w", err)
	}
	return nil
}
