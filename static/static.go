package static

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"log/slog"

	"github.com/justin-molloy/tfagent/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func UploadSFTP(filePath string, transfer config.ConfigEntry) (string, error) {
	const maxRetries = 3
	const retryDelay = 2 * time.Second
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		slog.Info("Attempting SFTP upload", "file", filePath, "attempt", attempt)

		result, err := uploadOnce(filePath, transfer)
		if err == nil {
			slog.Info("Upload succeeded", "file", filePath)
			return result, nil
		}

		lastErr = err
		slog.Warn("Upload attempt failed", "file", filePath, "error", err)

		if attempt < maxRetries {
			slog.Info("Retrying after delay", "delay", retryDelay)
			time.Sleep(retryDelay)
		}
	}

	slog.Error("Upload failed after all retries", "file", filePath, "error", lastErr)
	return "", lastErr
}

func uploadOnce(filePath string, transfer config.ConfigEntry) (string, error) {
	sshConfig := &ssh.ClientConfig{
		User: transfer.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(transfer.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // NOTE: Not for production
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", transfer.Server, transfer.Port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return "", fmt.Errorf("SSH dial failed: %w", err)
	}
	defer conn.Close()

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return "", fmt.Errorf("SFTP client creation failed: %w", err)
	}
	defer sftpClient.Close()

	srcFile, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open local file: %w", err)
	}
	defer srcFile.Close()

	remoteFileName := path.Base(filePath)
	dstPath := path.Join(transfer.RemotePath, remoteFileName)

	dstFile, err := sftpClient.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("failed to create remote file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "", fmt.Errorf("file copy failed: %w", err)
	}

	return "success", nil
}
