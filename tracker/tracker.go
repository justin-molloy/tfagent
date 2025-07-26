package tracker

import (
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/justin-molloy/tfagent/config"
)

func StartWatcher(cfg *config.ConfigData, w *fsnotify.Watcher, et *EventTracker) {
	go func() {
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return
				}

				slog.Debug("Filesystem event", "Op", event.Op, "Name", event.Name)

				// Just a bit of cleanup. If a remove event is received and we've already
				// added it to the processing list, ensure it is removed from the list.
				// this could occur if an incoming file copy is stopped.

				if event.Op&fsnotify.Remove != 0 && et.AlreadyExists(event.Name) {
					et.Delete(event.Name)
					slog.Debug("Cleared tracker after Remove event", "Name", event.Name)
				}

				// if event is not a write/create event, stop processing(continue)
				if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}

				// Match against all configured transfers
				for _, entry := range cfg.Transfers {
					tfMatch, err := FilterMatcher(event.Name, entry)
					if err != nil {
						slog.Warn("Failed to match transfer", "error", err, "Name", event.Name)
						continue
					}

					if et.AlreadyExists(event.Name) {
						slog.Debug("File already queued", "Name", event.Name)
						continue
					}
					if tfMatch {
						slog.Info("Matched and queued for processing", "Name", event.Name)
						et.RecordEvent(event.Name)
						break // Stop after first match
					} else {
						slog.Debug("No match", "Name", entry.Name, "File", event.Name)
					}
				}

			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				slog.Error("Watcher error", "error", err)
			}
		}
	}()
}

func FilterMatcher(eventName string, entry config.ConfigEntry) (bool, error) {
	// Normalise the path to avoid OS-specific path mismatches
	rel, err := filepath.Rel(entry.SourceDirectory, eventName)
	if err != nil || strings.HasPrefix(rel, "..") {
		// eventName is not under SourceDirectory
		return false, err
	}

	// If no filter is set, the path matches
	if entry.Filter == "" {
		return true, nil
	}

	// Compile and evaluate the regex filter
	re, err := regexp.Compile(entry.Filter)
	if err != nil {
		// If the regex is invalid, treat it as non-match (or log the error)
		return false, err
	}

	return re.MatchString(filepath.Base(eventName)), nil
}
