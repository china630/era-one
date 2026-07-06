// Package custody — hash-chain для chain of custody (S6-3, ADR-0006).
package custody

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// Chain — последовательная цепочка SHA-256 от предыдущего звена.
type Chain struct {
	mu       sync.Mutex
	lastHash string
}

// NewChain создаёт цепочку; genesis — нулевой хеш.
func NewChain() *Chain {
	return &Chain{lastHash: genesis}
}

const genesis = "0000000000000000000000000000000000000000000000000000000000000000"

// Entry — одно звено цепочки.
type Entry struct {
	PrevHash string
	Hash     string
}

// Seal вычисляет хеш payload с привязкой к предыдущему звену.
func (c *Chain) Seal(payload []byte) Entry {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := sha256.Sum256(append([]byte(c.lastHash), payload...))
	entry := Entry{PrevHash: c.lastHash, Hash: hex.EncodeToString(h[:])}
	c.lastHash = entry.Hash
	return entry
}

// Verify проверяет звено относительно prevHash и payload.
func Verify(prevHash string, hash string, payload []byte) bool {
	if prevHash == "" {
		prevHash = genesis
	}
	h := sha256.Sum256(append([]byte(prevHash), payload...))
	return hash == hex.EncodeToString(h[:])
}

// Head возвращает текущий хеш головы цепочки.
func (c *Chain) Head() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastHash
}
