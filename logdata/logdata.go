// logdata/logdata.go
package logdata

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// EnsureLogDir checks that the path exists and is a writable directory.
// If it doesn't exist, it tries to create it.

func ConfigLogger(path string) (string, *os.File, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", nil, fmt.Errorf("could not get absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(absPath, 0755); err != nil {
				return "", nil, fmt.Errorf("failed to create log directory: %w", err)
			}
		} else {
			return "", nil, fmt.Errorf("error accessing path: %w", err)
		}
	} else if !info.IsDir() {
		return "", nil, fmt.Errorf("log destination exists but is not a directory: %s", absPath)
	}

	now := time.Now()
	timestamp := now.Format("20060102")
	uniqueSuffix := generateRandomHex(3)
	filename := fmt.Sprintf("%s_%s.log", timestamp, uniqueSuffix)

	logFile := filepath.Join(absPath, filename)
	f, err := os.Create(logFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create log file: %w", err)
	}
	log.SetOutput(f)

	return logFile, f, nil
}

func generateRandomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "randerr"
	}
	return hex.EncodeToString(b)
}

func Write(level string, message string) {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Printf("[%s] [??:??] %s", level, message)
		return
	}

	funcName := runtime.FuncForPC(pc).Name()
	log.Printf("[%s] [%s:%d %s] %s", level, file, line, funcName, message)
}

func Info(msg string) {
	Write("INFO", msg)
}

func Warn(msg string) {
	Write("WARN", msg)
}

func Error(msg string) {
	Write("ERROR", msg)
	panic(msg)
}
