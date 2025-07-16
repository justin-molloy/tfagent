package static

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/justin-molloy/tfagent/config"
)

// UploadFile connects via SFTP and uploads a local file
func UploadFile(filePath string, transfers []config.ConfigEntry) (string, error) {

	fileAbs, err := filepath.Abs(filePath)

	if err != nil {
		return "", fmt.Errorf("could not resolve absolute path for %s: %w", filePath, err)
	}

	for _, entry := range transfers {
		dirAbs, err := filepath.Abs(entry.SourceDirectory)
		if err != nil {
			slog.Warn(fmt.Sprintf("Could not resolve absolute path for config entry: %s", entry.SourceDirectory))
			continue
		}

		if strings.HasPrefix(fileAbs, dirAbs+string(os.PathSeparator)) || fileAbs == dirAbs {
			slog.Info("Entry name", "name", entry.Name)
		}
	}

	return "success", nil
}

/*

func UploadFile(filePath string, transfers []config.ConfigEntry) error {
	// 1. Create SSH config
	sshConfig := &ssh.ClientConfig{
		User: cfg.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // NOTE: For testing only
	}

	// 2. Dial SSH
	addr := fmt.Sprintf("%s:%s", cfg.Server, cfg.Port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("failed to dial SSH: %w", err)
	}
	defer conn.Close()

	// 3. Open SFTP session
	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// 4. Open local file
	srcFile, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer srcFile.Close()

	// 5. Create remote file
	remoteFileName := path.Base(localFilePath)
	dstFile, err := sftpClient.Create(path.Join(cfg.RemotePath, remoteFileName))
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}
	defer dstFile.Close()

	// 6. Copy contents
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("file copy failed: %w", err)
	}

	return nil
}

*/
