package keycache

import "sync"

// Cache is a mutex protected cache for quick storage and retrieval of keys by KeyID
// This implements an in-memory store only, and any data is effectively lost on application exit
type Cache struct {
	keys map[string][]byte
	mtx  *sync.Mutex
}

// NewCache creates a new instance of a key cache
// It returns a pointer to the Cache struct
func NewCache() *Cache {
	return &Cache{
		keys: make(map[string][]byte),
		mtx:  &sync.Mutex{},
	}
}

// Get retrieves a key by its keyID
// It returns a byte slice containing the key data, as well as a bool that indicates
// if the key exists in the cache
func (c *Cache) Get(keyID string) (key []byte, exists bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	key, exists = c.keys[keyID]
	return
}

// Store persists a key in the cache by its keyID
func (c *Cache) Store(keyID string, key []byte) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.keys[keyID] = key
}

var global *Cache

func init() {
	global = NewCache()
}

// Get retrieves a key by its keyID from the default global keycache
func Get(keyID string) (key []byte, exists bool) {
	return global.Get(keyID)
}

// Store persists a key by its keyID from the default global keycache
func Store(keyID string, key []byte) {
	global.Store(keyID, key)
}
