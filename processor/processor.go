package processor

import (
	"log/slog"
	"strings"

	"github.com/justin-molloy/tfagent/config"
	"github.com/justin-molloy/tfagent/selector"
	"github.com/justin-molloy/tfagent/sendfile"
)

func StartProcessor(
	cfg *config.ConfigData,
	fileQueue <-chan string,
	processingSet *selector.FileSelector,
) {
	for file := range fileQueue {
		slog.Info("Processing file from queue", "file", file)

		matched := false
		for _, entry := range cfg.Transfers {
			if strings.HasPrefix(file, entry.SourceDirectory) {
				matched = true

				var result string
				var err error

				switch entry.TransferType {
				case "sftp":
					result, err = sendfile.UploadSFTP(file, &entry)
				case "local":
					slog.Warn("Local transfer not implemented", "file", file)
				case "scp":
					slog.Warn("SCP transfer not implemented", "file", file)
				default:
					slog.Warn("Unsupported Transfer Type", "file", file, "type", entry.TransferType)
				}

				if err != nil {
					slog.Error("Upload failed", "file", file, "error", err)
				} else {
					slog.Info("Upload complete", "file", file, "result", result)
				}

				processingSet.Delete(file)
				slog.Info("Removed from processing set", "file", file)
				break
			}
		}

		if !matched {
			slog.Warn("File did not match any transfer config", "file", file)
		}
	}
}

/*
				slog.Info("Starting simulated processing", "file", file)
				slog.Info("Sending using transfertype", "transfertype", entry.TransferType)
				time.Sleep(2 * time.Second)
				slog.Info("Finished processing", "file", file)

		var result string

		switch cfg.Transfer.TransferType {
		case "sftp":
			result, err = sendfile.UploadSFTP(file, *transfer)
		case "local":
			slog.Warn("Local transfer not implemented", "file", file)
			err = fmt.Errorf("local transfer not implemented")
		case "scp":
			slog.Warn("SCP transfer not implemented", "file", file)
			err = fmt.Errorf("SCP transfer not implemented")
		default:
			err = fmt.Errorf("unsupported transfer type: %s", transfer.TransferType)
		}

		if err != nil {
			slog.Error("Upload failed", "file", file, "error", err)
		} else {
			slog.Info("Upload complete", "file", file, "result", result)
		}

		processingSet.Delete(file)
	}
*/
