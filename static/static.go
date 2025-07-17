package static

import (
	"fmt"

	"github.com/justin-molloy/tfagent/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// UploadFile connects via SFTP and uploads a local file
func UploadSFTP(filePath string, transfer config.ConfigEntry) (string, error) {
	sshConfig := &ssh.ClientConfig{
		User: transfer.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(transfer.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // NOTE: For testing only
	}

	addr := fmt.Sprintf("%s:%s", transfer.Server, transfer.Port)
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return "", fmt.Errorf("failed to dial SSH: %w", err)
	}
	defer conn.Close()

	sftpClient, err := sftp.NewClient(conn)
	if err != nil {
		return "", fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	return "success", nil
}

/*

func UploadFile(filePath string, transfers []config.ConfigEntry) error {


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
