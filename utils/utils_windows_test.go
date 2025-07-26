//go:build windows
// +build windows

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestCheckReadyForProcessing(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")

	// Write some data to make it non-empty
	content := []byte("hello world")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// Call the function under test
	ready := CheckReadyForProcessing(filePath)
	if !ready {
		t.Errorf("Expected file to be ready for processing, but got false")
	}
}

func TestCheckReadyForProcessing_FileLocked(t *testing.T) {
	// Step 1: Create a temporary non-empty file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "lockedfile.txt")
	err := os.WriteFile(filePath, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Step 2: Open file with exclusive access (no sharing)
	ptr, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		t.Fatalf("Failed to convert path to UTF16: %v", err)
	}

	handle, err := windows.CreateFile(
		ptr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, // no sharing allowed
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		t.Fatalf("Failed to open file with exclusive lock: %v", err)
	}
	defer windows.CloseHandle(handle) // Ensure file is closed at end

	// Step 3: Call the function while the file is locked
	isReady := CheckReadyForProcessing(filePath)
	if isReady {
		t.Errorf("Expected file to be locked and not ready, but got true")
	}
}
