package tracker

import (
	"log/slog"
	"maps"
	"sync"
	"time"
)

type EventTracker struct {
	mu         sync.Mutex
	lastEvents map[string]time.Time
}

func NewEventTracker() *EventTracker {
	slog.Debug("New Event Tracker")
	return &EventTracker{
		lastEvents: make(map[string]time.Time),
	}
}

func (et *EventTracker) RecordEvent(name string) {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.lastEvents[name] = time.Now()
	slog.Debug("RecordEvent", "name", name, "event", et.lastEvents[name])
}

func (et *EventTracker) GetSnapshot() map[string]time.Time {
	et.mu.Lock()
	defer et.mu.Unlock()

	snapshot := make(map[string]time.Time, len(et.lastEvents))
	maps.Copy(snapshot, et.lastEvents)

	return snapshot
}

func (et *EventTracker) Delete(name string) {
	slog.Debug("DeleteEvent", "name", name, "event", et.lastEvents[name])
	et.mu.Lock()
	defer et.mu.Unlock()
	delete(et.lastEvents, name)
}

func (et *EventTracker) AlreadyExists(name string) bool {
	slog.Debug("AlreadyExistsEvent", "name", name, "event", et.lastEvents[name])
	et.mu.Lock()
	defer et.mu.Unlock()
	_, exists := et.lastEvents[name]
	return exists
}
