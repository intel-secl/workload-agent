package keycache

import "sync"

// Key
type Key struct {
	ID    string
	Bytes []byte
}

// Cache is a mutex protected cache for quick storage and retrieval of keys by ImageUUID
// This implements an in-memory store only, and any data is effectively lost on application exit
type Cache struct {
	keys map[string]Key
	mtx  *sync.Mutex
}

// NewCache creates a new instance of a key cache
// It returns a pointer to the Cache struct
func NewCache() *Cache {
	return &Cache{
		keys: make(map[string]Key),
		mtx:  &sync.Mutex{},
	}
}

// Get retrieves a key by its keyID
// It returns a byte slice containing the key data, as well as a bool that indicates
// if the key exists in the cache
func (c *Cache) Get(imageUUID string) (key Key, exists bool) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	key, exists = c.keys[imageUUID]
	return
}

// Store persists a key in the cache by its keyID
func (c *Cache) Store(imageID string, key Key) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.keys[imageID] = key
}

var global *Cache

func init() {
	global = NewCache()
}

// Get retrieves a key by its imageID from the default global keycache
func Get(imageID string) (key Key, exists bool) {
	return global.Get(imageID)
}

// Store persists a key by its keyID from the default global keycache
func Store(imageID string, key Key) {
	global.Store(imageID, key)
}
