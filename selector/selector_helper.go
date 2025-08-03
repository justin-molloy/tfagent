package selector

import (
	"maps"
	"sync"
	"time"
)

type FileSelector struct {
	mu            sync.Mutex
	selectedFiles map[string]time.Time
}

func NewFileSelector() *FileSelector {
	return &FileSelector{
		selectedFiles: make(map[string]time.Time),
	}
}

func (st *FileSelector) AddFile(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.selectedFiles[name] = time.Now()
}

func (st *FileSelector) GetSnapshot() map[string]time.Time {
	st.mu.Lock()
	defer st.mu.Unlock()

	snapshot := make(map[string]time.Time, len(st.selectedFiles))
	maps.Copy(snapshot, st.selectedFiles)

	return snapshot
}

func (st *FileSelector) Delete(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.selectedFiles, name)
}

func (st *FileSelector) AlreadyExists(name string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, exists := st.selectedFiles[name]
	return exists
}
