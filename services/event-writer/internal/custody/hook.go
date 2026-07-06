// Package custody — hook hash-chain перед записью в ClickHouse (S6-3).
package custody

import (
	"log"
	"os"
	"sync"

	platformcustody "era/services/platform/custody"
	"google.golang.org/protobuf/proto"
)

var (
	mu    sync.Mutex
	chain = platformcustody.NewChain()
)

// SealEnvelope вычисляет звено hash-chain для protobuf Envelope.
func SealEnvelope(env proto.Message) platformcustody.Entry {
	payload, err := proto.Marshal(env)
	if err != nil {
		return platformcustody.Entry{}
	}
	mu.Lock()
	defer mu.Unlock()
	entry := chain.Seal(payload)
	if path := os.Getenv("ERA_CUSTODY_LOG"); path != "" {
		log.Printf("custody: head=%s prev=%s bytes=%d log=%s", entry.Hash, entry.PrevHash, len(payload), path)
	}
	return entry
}
