package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/justin-molloy/tfagent/config"
	"github.com/justin-molloy/tfagent/logdata"
	"github.com/justin-molloy/tfagent/static"
)

var configFile string = "config.yaml"

func main() {

	// read config

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Configure logging

	logFile, err := logdata.SetupLogger(filepath.Join(cfg.LogDest, "logfile.log"), slog.LevelInfo)
	if err != nil {
		panic(err)
	}

	defer logFile.Close()

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
			slog.Info("Added Source Dir: ", "source", entry.SourceDirectory)
		}
	}

	// Main program loop - run function in background
	lastEvent := make(map[string]time.Time)

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
					slog.Info("Processed after delay", "file", file)

					err := static.UploadFile(file, cfg.Transfers)
					if err != nil {
						slog.Error(err.Error())
					} else {
						delete(lastEvent, file)
					}
				}
			}
			time.Sleep(500 * time.Millisecond)
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
