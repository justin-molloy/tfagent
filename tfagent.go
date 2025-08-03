package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	//	"os/signal"
	"strings"
	//	"sync"
	//	"syscall"
	//	"time"

	"golang.org/x/sys/windows/svc"

	"github.com/fsnotify/fsnotify"
	"github.com/justin-molloy/tfagent/config"

	//	"github.com/justin-molloy/tfagent/selector"
	"github.com/justin-molloy/tfagent/service"
	//	"github.com/justin-molloy/tfagent/static"
	//	"github.com/justin-molloy/tfagent/tracker"
)

func main() {

	// Application name used for Windows service call, and for defining
	// where the config file should be(if installed using installer).

	const AppName = "TFAgent"

	flags := config.ParseFlags()

	// Get and read config file

	configFile, err := config.GetConfigFile(AppName)
	if err != nil {
		log.Fatalf("Can't find where the config file lives: %v", err)
		os.Exit(1)
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
		os.Exit(1)
	}

	// set up logging immediately once we've determined where we should
	// log to and exit if we can't set logging up(if we can't set up
	// logging, there's a bigger problem that needs to be resolved).

	logFile, err := config.SetupLogger(cfg, flags)
	if err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
		os.Exit(1)
	}
	if logFile != nil {
		defer logFile.Close()
	}

	// Run the service if called from Windows Service

	isService, err := svc.IsWindowsService()
	if err != nil {
		slog.Error("failed to determine session type", "error", err)
		os.Exit(1)
	}

	if isService {
		slog.Info("Running as Windows Service", "isService", isService)
		svc.Run(AppName, &service.TFAgentService{})
		return
	} else {
		slog.Info("Running as standalone app outside of Windows Service Control Manager")
	}

	if flags.PrtConf {
		config.PrintConfig(cfg)
		os.Exit(0)
	}

	// Create new filesystem event watcher

	w, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("Failed to create watcher", "error", err)
		os.Exit(1)
	}
	defer w.Close()

	// Set up our tracking maps so that events can be safely handled.
	// tracker is a map that records all filesystem events that are generated
	// from watcher(fsnotify). This map is passed to the selector below.
	// The tracker function also does some basic tests to ensure that the event
	// is one we're interested in - eg. create/notify events, and that the file
	// meets the filter criteria specified in the config.

	/*

		trackerMap := tracker.NewEventTracker()

		// source directories from config are added to watcher

		tracker.StartWatcher(&cfg, w, trackerMap)
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

		// Selector views events that have been added to the tracker map, and
		// moves them to the fileQueue when eligible.

		processingSet := sync.Map{}

		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()

			for range ticker.C {
				selector.ProcessSnapshot(&cfg, trackerMap, fileQueue, &processingSet)
			}
		}()

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
	*/

}

func FindMatchingTransfer(file string, transfers []config.ConfigEntry) (*config.ConfigEntry, error) {
	for _, entry := range transfers {
		if strings.HasPrefix(file, entry.SourceDirectory) {
			return &entry, nil
		}
	}
	return nil, fmt.Errorf("no matching config entry found for file: %s", file)
}
