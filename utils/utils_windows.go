//go:build windows

package utils

import (
	"os"

	"golang.org/x/sys/windows"
)

// CheckReadyForProcessing returns true if the file exists, has non-zero size, and is not locked by another process.
func CheckReadyForProcessing(path string) bool {
	// 1. Check if file exists and is non-empty
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return false
	}

	// 2. Try to open with exclusive access
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false
	}

	handle, err := windows.CreateFile(
		ptr,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0, // No sharing = exclusive access
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return false // File is likely locked
	}
	windows.CloseHandle(handle)

	return true
}
