// selector/selector.go
package selector

import (
	"log/slog"
	"sync"
	"time"

	"github.com/justin-molloy/tfagent/config"
	watcher "github.com/justin-molloy/tfagent/tracker"
	"github.com/justin-molloy/tfagent/utils"
)

// processSnapshot checks all tracked events and queues files that are ready for processing.
// filters and transfer eligibility will be determined here as well, using values from the
// transfer configuration.

func ProcessSnapshot(
	cfg *config.ConfigData,
	tracker *watcher.EventTracker,
	fileQueue chan<- string,
	processingSet *sync.Map,
) {
	now := time.Now()
	snapshot := tracker.GetSnapshot()

	// TODO turn this one on only if snapshot isn't empty
	//	if snapshot {
	//		slog.Debug("Snapshot of lastEvents", "events", snapshot)
	//	}

	for file, t := range snapshot {
		if !hasDelayElapsed(t, now, cfg.Delay) {
			continue
		}

		// utils are OS specific routines(just about everything else should work no matter
		// which operating system it is run on, but these are filesystem operations.)

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
