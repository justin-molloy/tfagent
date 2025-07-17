package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/justin-molloy/tfagent/config"
	"github.com/justin-molloy/tfagent/static"
)

func main() {

	// Parse commandline flags

	flags := config.ParseFlags()

	// Load config

	cfg, err := config.LoadConfig(flags.ConfigFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Override config with flags if different from defaults

	if flags.LogFile != "logs/app.log" {
		cfg.LogFile = flags.LogFile
	}
	if flags.LogLevel != "info" {
		cfg.LogLevel = flags.LogLevel
	}
	if flags.LogToConsole {
		cfg.LogToConsole = true
	}

	// Configure logging

	logFile, err := config.SetupLogger(cfg.LogFile, cfg.LogLevel, cfg.LogToConsole)
	if err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	// Add source directories to watch from config

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error(err.Error())
	}

	defer watcher.Close()

	for _, entry := range cfg.Transfers {
		err = watcher.Add(entry.SourceDirectory)
		if err != nil {
			slog.Error(err.Error())
		} else {
			slog.Info("Added Source Dir", "source", entry.SourceDirectory)
		}
	}

	var processingSet sync.Map // safely tracks in-progress files
	lastEvent := make(map[string]time.Time)
	fileQueue := make(chan string, 100) // buffered to avoid blocking

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				slog.Info("Filesystem event", "Op", event.Op, "Name", event.Name)

				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					lastEvent[event.Name] = time.Now()
				}

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Error(err.Error())
			}
		}
	}()

	// Polling loop to check for stable files

	go func() {
		for {
			now := time.Now()
			for file, t := range lastEvent {
				delaySec := GetDelayForFile(file, cfg.Transfers, 2)

				if now.Sub(t) > time.Duration(delaySec)*time.Second {
					// Avoid enqueueing if already processing
					if _, alreadyProcessing := processingSet.Load(file); !alreadyProcessing {
						processingSet.Store(file, true)
						slog.Info("Queued file after delay", "file", file)
						fileQueue <- file
					} else {
						slog.Debug("Skipped enqueue; already processing", "file", file)
					}
					delete(lastEvent, file)
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	go func() {
		for file := range fileQueue {
			slog.Info("Processing file from queue", "file", file)

			transfer, err := FindMatchingTransfer(file, cfg.Transfers)
			if err != nil {
				slog.Warn("No matching transfer config found", "file", file, "error", err)
				processingSet.Delete(file)
				continue
			}

			var result string

			switch transfer.TransferType {
			case "sftp":
				result, err = static.UploadSFTP(file, *transfer)
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
	}()

	// Block and create a channel to receive OS signals for interrupts
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	slog.Info("Program terminated by user", "signal", sig.String())
	os.Exit(0)

}

func GetDelayForFile(filePath string, transfers []config.ConfigEntry, defaultDelay int) int {
	fileAbs, err := filepath.Abs(filePath)
	if err != nil {
		slog.Warn(fmt.Sprintf("Could not resolve absolute path for: %s", filePath))
		return defaultDelay
	}

	for _, entry := range transfers {
		dirAbs, err := filepath.Abs(entry.SourceDirectory)
		if err != nil {
			slog.Warn(fmt.Sprintf("Could not resolve absolute path for config entry: %s", entry.SourceDirectory))
			continue
		}

		if strings.HasPrefix(fileAbs, dirAbs+string(os.PathSeparator)) || fileAbs == dirAbs {
			return *entry.Delay
		}
	}

	slog.Warn(fmt.Sprintf("No matching SourceDirectory found for file: %s. Using default delay.", filePath))
	return defaultDelay
}

func FindMatchingTransfer(file string, transfers []config.ConfigEntry) (*config.ConfigEntry, error) {
	for _, entry := range transfers {
		if strings.HasPrefix(file, entry.SourceDirectory) {
			return &entry, nil
		}
	}
	return nil, fmt.Errorf("no matching config entry found for file: %s", file)
}
