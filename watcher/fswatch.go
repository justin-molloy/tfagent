package watcher

import (
	"log/slog"

	"github.com/fsnotify/fsnotify"
)

func StartWatcher(w *fsnotify.Watcher, et *EventTracker) {
	go func() {
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return
				}

				slog.Debug("Filesystem event", "Op", event.Op, "Name", event.Name)

				if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
					et.RecordEvent(event.Name)
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
