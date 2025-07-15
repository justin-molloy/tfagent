package logdata

import (
	"log/slog"
	"os"
)

// SetupLogger creates and configures an slog logger that writes to the provided file path.
// It returns the file handle (which should be closed by the caller) and any error encountered.
func SetupLogger(logFilePath string, level slog.Level) (*os.File, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: level,
	})

	slog.Info("Program started. Future log messages written to file: ", "path", logFilePath)

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logFile, nil
}
