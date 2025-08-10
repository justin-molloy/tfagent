package processor

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
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

		// Use index form to avoid pointer-to-range-variable bug.
		for i := range cfg.Transfers {
			entry := cfg.Transfers[i]
			if !strings.HasPrefix(file, entry.SourceDirectory) {
				continue
			}
			matched = true

			var (
				result string
				err    error
			)

			switch entry.TransferType {
			case "sftp":
				// Prefer passing the entry by VALUE unless UploadSFTP needs to mutate it.
				// If UploadSFTP wants a pointer, pass &cfg.Transfers[i] (not &entry).
				result, err = sendfile.UploadSFTP(file, entry) // value
				// result, err = sendfile.UploadSFTP(file, &cfg.Transfers[i]) // pointer version
			case "local":
				slog.Warn("Local transfer not implemented", "file", file)
			case "scp":
				slog.Warn("SCP transfer not implemented", "file", file)
			default:
				slog.Warn("Unsupported transfer type", "file", file, "type", entry.TransferType)
			}

			if err != nil {
				slog.Error("Upload failed", "file", file, "error", err)
				// On error → fail action
				if aerr := ActionOnFail(entry, file); aerr != nil {
					slog.Warn("ActionOnFail error", "file", file, "error", aerr)
				}
			} else {
				slog.Info("Upload complete", "file", file, "result", result)
				// On success → success action
				if aerr := ActionOnSuccess(entry, file); aerr != nil {
					slog.Warn("ActionOnSuccess error", "file", file, "error", aerr)
				}
			}

			processingSet.Delete(file)
			slog.Info("Removed from processing set", "file", file)
			break
		}

		if !matched {
			slog.Warn("File did not match any transfer config", "file", file)
		}
	}
}

func ActionOnSuccess(transfer config.ConfigEntry, file string) error {
	slog.Debug("Action on success",
		"name", transfer.Name,
		"file", file,
		"action", transfer.ActionOnSuccess,
	)

	switch strings.ToLower(strings.TrimSpace(transfer.ActionOnSuccess)) {
	case "archive":
		archiveDest := strings.TrimSpace(transfer.ArchiveDest)
		if archiveDest == "" {
			archiveDest = filepath.Join(transfer.SourceDirectory, "archive")
			slog.Debug("No ArchiveDest specified, defaulting to", "path", archiveDest)
		}

		// Ensure archive directory exists (try configured; fall back to source/archive)
		if err := os.MkdirAll(archiveDest, 0o755); err != nil {
			slog.Warn("Failed to create archive directory, falling back to source_directory/archive",
				"configured", archiveDest, "error", err)
			archiveDest = filepath.Join(transfer.SourceDirectory, "archive")
			if mkErr := os.MkdirAll(archiveDest, 0o755); mkErr != nil {
				slog.Error("Failed to create fallback archive directory", "error", mkErr)
				return errors.New("unable to create archive directory")
			}
		}

		destPath := filepath.Join(archiveDest, filepath.Base(file))
		if err := os.Rename(file, destPath); err != nil {
			slog.Error("Failed to move file to archive", "file", file, "dest", destPath, "error", err)
			return err
		}

		slog.Info("File archived successfully", "file", file, "dest", destPath)
		return nil

	case "delete":
		if err := os.Remove(file); err != nil {
			slog.Error("Failed to remove file after successful transfer", "file", file, "error", err)
			return err
		}
		slog.Info("File removed after successful transfer", "file", file)
		return nil

	case "", "none":
		// No-op
		slog.Info("No further action after successful transfer", "file", file)
		return nil

	default:
		// Unrecognised action -> treat as no-op but log it
		slog.Warn("Unknown ActionOnSuccess; no action taken",
			"action", transfer.ActionOnSuccess, "file", file)
		return nil
	}
}

func ActionOnFail(transfer config.ConfigEntry, file string) error {
	slog.Debug("Action on Fail",
		"name", transfer.Name,
		"file", file,
		"action", transfer.ActionOnFail,
	)

	switch strings.ToLower(strings.TrimSpace(transfer.ActionOnFail)) {
	case "archive":
		failDest := strings.TrimSpace(transfer.FailDest)
		if failDest == "" {
			failDest = filepath.Join(transfer.SourceDirectory, "fail")
			slog.Debug("No ArchiveDest specified, defaulting to", "path", failDest)
		}

		// Ensure fail directory exists (try configured; fall back to source/fail)
		if err := os.MkdirAll(failDest, 0o755); err != nil {
			slog.Warn("Failed to create fail directory, falling back to source_directory/fail",
				"configured", failDest, "error", err)
			failDest = filepath.Join(transfer.SourceDirectory, "fail")
			if mkErr := os.MkdirAll(failDest, 0o755); mkErr != nil {
				slog.Error("Failed to create fallback fail directory", "error", mkErr)
				return errors.New("unable to create fail directory")
			}
		}

		destPath := filepath.Join(failDest, filepath.Base(file))
		if err := os.Rename(file, destPath); err != nil {
			slog.Error("Failed to move file to fail", "file", file, "dest", destPath, "error", err)
			return err
		}

		slog.Info("File archived successfully", "file", file, "dest", destPath)
		return nil

	case "delete":
		if err := os.Remove(file); err != nil {
			slog.Error("Failed to remove file after failed transfer", "file", file, "error", err)
			return err
		}
		slog.Info("File removed after failed transfer", "file", file)
		return nil

	case "", "none":
		// No-op
		slog.Info("No further action after failed transfer", "file", file)
		return nil

	default:
		// Unrecognised action -> treat as no-op but log it
		slog.Warn("Unknown ActionOnFail; no action taken",
			"action", transfer.ActionOnFail, "file", file)
		return nil
	}
}
