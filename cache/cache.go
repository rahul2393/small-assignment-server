// Package cache is an in-memory key value store.
// This package should be used when data doesn't
// needed to be cached across VMs.  cache is thread
// safe so it can be used from multiple goroutines.
package cache

import (
	"sync"
	"time"
)

// Set sets the item in the namespace for the given key.
// If a value already exists for the namespace and key combination then
// it is overwritten.
func Set(namespace, key string, i Item) {
	key = formKey(namespace, key)
	m.set(key, i)
}

// Get retrieves an item from the cache from the namespace
// for the given key. src will be equal to the Item.Src
// supplied to the Set function for the same key.  If the
// item isn't in the cache then src will be nil.  in will
// be whether an item was in the cache.
func Get(namespace, key string) (src interface{}, in bool) {
	key = formKey(namespace, key)
	return m.get(key)
}

func GetAll() map[string]Item {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resources
}

// Delete deletes an item from the cache from the namespace
// for the given key.  Delete may be called even if an item
// was never set for the namespace and key combination.
func Delete(namespace, key string) {
	key = formKey(namespace, key)
	m.delete(key)
}

func DeleteByKey(key string) {
	m.delete(key)
}

func Clear() {
	for key, _ := range m.resources {
		m.delete(key)
	}
}

func formKey(namespace, key string) string {
	return namespace + ":" + key
}

var (
	m = &cacheMap{
		resources: map[string]Item{},
		mu:        &sync.RWMutex{},
	}
)

// An Item is an entry in the cache.
type Item struct {
	// Src is object to be stored
	Src interface{}
	// Duration is the duration the item should be able
	// to be retrieved.
	Duration   time.Duration
	insertTime time.Time
}

func (i Item) isExpired() bool {
	if i.insertTime.IsZero() {
		return true
	} else if i.Duration == 0 {
		return false
	}
	return i.insertTime.Add(i.Duration).Before(time.Now())
}

type cacheMap struct {
	resources map[string]Item
	mu        *sync.RWMutex
}

func (m *cacheMap) set(key string, i Item) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i.insertTime = time.Now()
	m.resources[key] = i
}

func (m *cacheMap) get(key string) (interface{}, bool) {
	m.mu.RLock()
	i, in := m.resources[key]
	m.mu.RUnlock()
	if in && i.isExpired() {
		m.delete(key)
		return nil, false
	}
	return i.Src, in
}

func (m *cacheMap) delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.resources, key)
}
