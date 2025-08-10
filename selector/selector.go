// selector/selector.go
package selector

import (
	"log/slog"
	"time"

	"github.com/justin-molloy/tfagent/tracker"
	"github.com/justin-molloy/tfagent/utils"
)

// processSnapshot checks all tracked events and queues files that are ready for processing.
// filters and transfer eligibility will be determined here as well, using values from the
// transfer configuration.

func StartSelector(
	trackerMap *tracker.EventTracker,
	fileQueue chan<- string,
	processingSet *FileSelector,
) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		snapshot := trackerMap.GetSnapshot()

		if len(snapshot) > 0 {
			slog.Debug("Snapshot of lastEvents", "events", snapshot)
		}

		// hasDelayElapsed - currently hardcoded to 1 second (may fix this later)
		for file, t := range snapshot {
			if !hasDelayElapsed(t, now, 1) {
				continue
			}

			if !utils.CheckReadyForProcessing(file) {
				continue
			}

			if processingSet.AlreadyExists(file) {
				slog.Debug("Skipped enqueue; already processing", "file", file)
				trackerMap.Delete(file)
				continue
			}

			processingSet.AddFile(file)
			slog.Info("Queued file after delay", "file", file)
			fileQueue <- file
			trackerMap.Delete(file)
		}
	}
}

// hasDelayElapsed checks if the required delay has passed since the event time.
func hasDelayElapsed(t time.Time, now time.Time, delaySec int) bool {
	return now.Sub(t) > time.Duration(delaySec)*time.Second
}
