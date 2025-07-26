package watcher

import (
	"maps"
	"sync"
	"time"
)

type EventTracker struct {
	mu         sync.Mutex
	lastEvents map[string]time.Time
}

func NewEventTracker() *EventTracker {
	return &EventTracker{
		lastEvents: make(map[string]time.Time),
	}
}

func (et *EventTracker) RecordEvent(name string) {
	et.mu.Lock()
	defer et.mu.Unlock()
	et.lastEvents[name] = time.Now()
}

func (et *EventTracker) GetSnapshot() map[string]time.Time {
	et.mu.Lock()
	defer et.mu.Unlock()

	snapshot := make(map[string]time.Time, len(et.lastEvents))
	maps.Copy(snapshot, et.lastEvents)
	return snapshot
}

func (et *EventTracker) Delete(name string) {
	et.mu.Lock()
	defer et.mu.Unlock()
	delete(et.lastEvents, name)
}
