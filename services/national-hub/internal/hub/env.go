package hub

import (
	"fmt"
	"os"
)

// NewFromEnv — ERA_STORE_PATH → SQLite, иначе in-memory (S7-4).
func NewFromEnv(path string) (ObjectStore, func(), error) {
	if path == "" {
		path = os.Getenv("ERA_STORE_PATH")
	}
	if path == "" {
		return NewStore(), func() {}, nil
	}
	st, err := OpenSQLiteStore(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite store: %w", err)
	}
	return st, func() { _ = st.Close() }, nil
}
