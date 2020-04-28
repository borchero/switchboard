package utils

import (
	"sync"

	"github.com/borchero/switchboard/backends"
)

// BackendCache provides cache-safe access to backends indexed by name.
type BackendCache interface {
	// Get returns the backend zone with the specified name if it can be found.
	Get(name string) (backends.DNSZone, bool)
	// Update adds/replaces the given backend zone in the cache with the given name.
	Update(name string, zone backends.DNSZone)
	// Remove removes the backend zone with the specified name from the cache.
	Remove(name string)
}

type backendCacheImpl struct {
	backends map[string]backends.DNSZone
	mutex    sync.RWMutex
}

// NewBackendCache returns a freshly initialized backend cache.
func NewBackendCache() BackendCache {
	return &backendCacheImpl{backends: make(map[string]backends.DNSZone)}
}

func (cache *backendCacheImpl) Get(name string) (backends.DNSZone, bool) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	zone, ok := cache.backends[name]
	return zone, ok
}

func (cache *backendCacheImpl) Update(name string, zone backends.DNSZone) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.backends[name] = zone
}

func (cache *backendCacheImpl) Remove(name string) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	delete(cache.backends, name)
}
