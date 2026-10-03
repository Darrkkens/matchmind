package football

import (
	"context"
	"sync"
	"time"
)

// ResponseCache keeps raw upstream responses so repeated lookups do not spend API
// quota. A zero TTL means the entry never expires (e.g. a finished match).
type ResponseCache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Put(ctx context.Context, key string, body []byte, ttl time.Duration) error
}

// MemoryCache is the default ResponseCache; it is lost on restart.
type MemoryCache struct {
	mu      sync.Mutex
	entries map[string]memoryEntry
	now     func() time.Time
}

type memoryEntry struct {
	body    []byte
	expires time.Time
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{entries: map[string]memoryEntry{}, now: time.Now}
}

func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || (!e.expires.IsZero() && c.now().After(e.expires)) {
		return nil, false, nil
	}
	return append([]byte(nil), e.body...), true, nil
}

func (c *MemoryCache) Put(_ context.Context, key string, body []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	e := memoryEntry{body: append([]byte(nil), body...)}
	if ttl > 0 {
		e.expires = c.now().Add(ttl)
	}
	c.entries[key] = e
	return nil
}
