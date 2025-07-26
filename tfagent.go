package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"github.com/justin-molloy/tfagent/config"
	"github.com/justin-molloy/tfagent/selector"
	"github.com/justin-molloy/tfagent/static"
	"github.com/justin-molloy/tfagent/watcher"
)

func main() {

	// Parse commandline flags

	flags := config.ParseFlags()

	// Load config from config.yaml (default) or whatever file passed by flags.

	cfg, err := config.LoadConfig(flags.ConfigFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	if flags.PrtConf {
		config.PrintConfig(cfg)
		os.Exit(0)
	}
	// Configure logging

	logFile, err := config.SetupLogger(cfg, flags)
	if err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	// 1. Create new filesystem watcher

	w, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Failed to create watcher", "error", err)
		os.Exit(1)
	}
	defer w.Close()

	// Set up our tracking maps so that events can be safely handled.
	// tracker is a map that records all filesystem events that are generated
	// from the fsnotify module. This map is passed to the selector below.
	tracker := watcher.NewEventTracker()

	// watcher is the filesystem event watcher
	watcher.StartWatcher(w, tracker)

	// Add source directories (or files) to watch from config

	for _, entry := range cfg.Transfers {
		err = w.Add(entry.SourceDirectory)
		if err != nil {
			slog.Error(err.Error())
		} else {
			slog.Info("Added Source Dir", "source", entry.SourceDirectory)
		}
	}

	// queue for files to be processed
	fileQueue := make(chan string, 100) // buffered to avoid blocking

	// Selector views events that have been added to the tracker map
	processingSet := sync.Map{}
	selector.StartSelector(cfg.Delay, tracker, fileQueue, &processingSet)

	// Processing loop - read from the queue and process file.

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

func FindMatchingTransfer(file string, transfers []config.ConfigEntry) (*config.ConfigEntry, error) {
	for _, entry := range transfers {
		if strings.HasPrefix(file, entry.SourceDirectory) {
			return &entry, nil
		}
	}
	return nil, fmt.Errorf("no matching config entry found for file: %s", file)
}
