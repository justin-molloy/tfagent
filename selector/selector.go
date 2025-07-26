// selector/selector.go
package selector

import (
	"log/slog"
	"sync"
	"time"

	"github.com/justin-molloy/tfagent/utils"
	"github.com/justin-molloy/tfagent/watcher"
)

// StartSelector runs a background goroutine that periodically checks the event tracker
// and queues files that have been stable for a specified delay.
func StartSelector(
	cfgDelay int,
	tracker *watcher.EventTracker,
	fileQueue chan<- string,
	processingSet *sync.Map,
) {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			processSnapshot(cfgDelay, tracker, fileQueue, processingSet)
		}
	}()
}

// processSnapshot checks all tracked events and queues files that are ready for processing.
// filters and transfer eligibility will be determined here as well, using values from the
// transfer configuration.

func processSnapshot(
	cfgDelay int,
	tracker *watcher.EventTracker,
	fileQueue chan<- string,
	processingSet *sync.Map,
) {
	now := time.Now()
	snapshot := tracker.GetSnapshot()
	slog.Debug("Snapshot of lastEvents", "events", snapshot)

	for file, t := range snapshot {
		if !hasDelayElapsed(t, now, cfgDelay) {
			continue
		}

		// utils are OS specific routines.

		if !utils.CheckReadyForProcessing(file) {
			continue
		}

		if _, alreadyProcessing := processingSet.Load(file); alreadyProcessing {
			slog.Debug("Skipped enqueue; already processing", "file", file)
			tracker.Delete(file)
			continue
		}

		processingSet.Store(file, true)
		slog.Info("Queued file after delay", "file", file)
		fileQueue <- file
		tracker.Delete(file)
	}
}

// hasDelayElapsed checks if the required delay has passed since the event time.
func hasDelayElapsed(t time.Time, now time.Time, delaySec int) bool {
	return now.Sub(t) > time.Duration(delaySec)*time.Second
}
