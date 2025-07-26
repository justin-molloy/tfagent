package static

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
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

	// read and validate private key

	key, err := os.ReadFile(transfer.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("unable to read private key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return "", fmt.Errorf("unable to parse private key: %w", err)
	}

	// attempt file transfer {maxRetries} times

	for attempt := 1; attempt <= maxRetries; attempt++ {
		slog.Info("Attempting SFTP upload", "file", filePath, "attempt", attempt)

		result, err := uploadOnce(filePath, transfer, signer)
		if err == nil {
			return result, nil // successful transfer
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

func uploadOnce(filePath string, transfer config.ConfigEntry, signer ssh.Signer) (string, error) {

	sshConfig := &ssh.ClientConfig{
		User: transfer.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // NOTE: Not safe for production
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", transfer.Server, transfer.Port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return "failed", fmt.Errorf("SSH dial failed: %w", err)
	}
	defer conn.Close()

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return "failed", fmt.Errorf("SFTP client creation failed: %w", err)
	}
	defer sftpClient.Close()

	srcFile, err := os.Open(filePath)
	if err != nil {
		return "failed", fmt.Errorf("failed to open local file: %w", err)
	}
	defer srcFile.Close()

	// filepath uses the local OS file type, but path always assumes linux.
	// It's a good guess that a remote sftp target is unlikely to be Windows.

	remoteFileName := filepath.Base(filePath)
	dstPath := path.Join(transfer.RemotePath, remoteFileName)

	dstFile, err := sftpClient.Create(dstPath)
	if err != nil {
		return "failed", fmt.Errorf("failed to create remote file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return "failed", fmt.Errorf("file copy failed: %w", err)
	}

	return "success", nil
}
