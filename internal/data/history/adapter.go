package history

import (
	"time"
)

// Adapter bridges Store to the core HistoryStore port.
type Adapter struct {
	store *Store
}

func NewAdapter(store *Store) *Adapter {
	return &Adapter{store: store}
}

func (a *Adapter) SaveSnapshot(projectKey string, snapshot Snapshot) error {
	return a.store.SaveSnapshot(projectKey, snapshot)
}

func (a *Adapter) LoadSnapshots(projectKey string, since time.Time) ([]Snapshot, error) {
	return a.store.LoadSnapshots(projectKey, since)
}
