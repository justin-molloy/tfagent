package tracker

import (
	"testing"
	"time"
)

func TestRecordAndGetSnapshot(t *testing.T) {
	tracker := NewEventTracker()

	tracker.RecordEvent("file1.txt")
	time.Sleep(10 * time.Millisecond)
	tracker.RecordEvent("file2.txt")

	snapshot := tracker.GetSnapshot()

	if len(snapshot) != 2 {
		t.Fatalf("Expected 2 entries in snapshot, got %d", len(snapshot))
	}

	if _, ok := snapshot["file1.txt"]; !ok {
		t.Error("file1.txt not found in snapshot")
	}
	if _, ok := snapshot["file2.txt"]; !ok {
		t.Error("file2.txt not found in snapshot")
	}
}

func TestDelete(t *testing.T) {
	tracker := NewEventTracker()

	tracker.RecordEvent("file1.txt")
	tracker.RecordEvent("file2.txt")

	tracker.Delete("file1.txt")
	snapshot := tracker.GetSnapshot()

	if _, ok := snapshot["file1.txt"]; ok {
		t.Error("file1.txt was not deleted")
	}
	if _, ok := snapshot["file2.txt"]; !ok {
		t.Error("file2.txt should still exist")
	}
}

func TestSnapshotIsCopy(t *testing.T) {
	tracker := NewEventTracker()
	tracker.RecordEvent("file.txt")

	snapshot := tracker.GetSnapshot()
	snapshot["file.txt"] = time.Time{} // try modifying the copy

	// Get a fresh snapshot and verify original wasn't modified
	newSnapshot := tracker.GetSnapshot()
	if newSnapshot["file.txt"].IsZero() {
		t.Error("Modifying snapshot should not affect original map")
	}
}

func TestHasDuplicate(t *testing.T) {
	tracker := NewEventTracker()
	testFile := "file.txt"

	if tracker.HasDuplicate(testFile) {
		t.Errorf("Expected no duplicate, but got true")
	}

	tracker.RecordEvent(testFile)

	if !tracker.HasDuplicate(testFile) {
		t.Errorf("Expected duplicate, but got false")
	}

	tracker.Delete(testFile)

	if tracker.HasDuplicate(testFile) {
		t.Errorf("Expected no duplicate after delete, but got true")
	}
}
