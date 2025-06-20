package streaming

import (
	"sync"
)

// LRUCache implements a thread-safe Least Recently Used cache
type LRUCache struct {
	mutex    sync.RWMutex
	capacity int
	items    map[string]*cacheNode
	head     *cacheNode // Most recently used
	tail     *cacheNode // Least recently used
	hits     int64
	misses   int64
}

// cacheNode represents a node in the doubly-linked list
type cacheNode struct {
	key   string
	value interface{}
	prev  *cacheNode
	next  *cacheNode
}

// NewLRUCache creates a new LRU cache with the specified capacity
func NewLRUCache(capacity int) *LRUCache {
	if capacity <= 0 {
		capacity = 100 // Default capacity
	}

	cache := &LRUCache{
		capacity: capacity,
		items:    make(map[string]*cacheNode),
	}

	// Initialize dummy head and tail nodes
	cache.head = &cacheNode{}
	cache.tail = &cacheNode{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head

	return cache
}

// Get retrieves a value from the cache and marks it as recently used
func (c *LRUCache) Get(key string) (interface{}, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if node, exists := c.items[key]; exists {
		// Move to front (most recently used)
		c.moveToFront(node)
		c.hits++
		return node.value, true
	}

	c.misses++
	return nil, false
}

// Put adds or updates a key-value pair in the cache
func (c *LRUCache) Put(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if node, exists := c.items[key]; exists {
		// Update existing node
		node.value = value
		c.moveToFront(node)
		return
	}

	// Create new node
	newNode := &cacheNode{
		key:   key,
		value: value,
	}

	// Add to front
	c.addToFront(newNode)
	c.items[key] = newNode

	// Check capacity and evict if necessary
	if len(c.items) > c.capacity {
		c.evictLRU()
	}
}

// Remove removes a key from the cache
func (c *LRUCache) Remove(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if node, exists := c.items[key]; exists {
		c.removeNode(node)
		delete(c.items, key)
		return true
	}

	return false
}

// Clear removes all items from the cache
func (c *LRUCache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]*cacheNode)
	c.head.next = c.tail
	c.tail.prev = c.head
	c.hits = 0
	c.misses = 0
}

// Len returns the current number of items in the cache
func (c *LRUCache) Len() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.items)
}

// Capacity returns the maximum capacity of the cache
func (c *LRUCache) Capacity() int {
	return c.capacity
}

// Stats returns cache statistics
func (c *LRUCache) Stats() CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	total := c.hits + c.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(c.hits) / float64(total) * 100
	}

	return CacheStats{
		Hits:     c.hits,
		Misses:   c.misses,
		HitRate:  hitRate,
		Size:     len(c.items),
		Capacity: c.capacity,
	}
}

// Keys returns all keys in the cache (from most to least recently used)
func (c *LRUCache) Keys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	keys := make([]string, 0, len(c.items))
	current := c.head.next

	for current != c.tail {
		keys = append(keys, current.key)
		current = current.next
	}

	return keys
}

// Contains checks if a key exists in the cache without updating its position
func (c *LRUCache) Contains(key string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	_, exists := c.items[key]
	return exists
}

// Peek gets a value without marking it as recently used
func (c *LRUCache) Peek(key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if node, exists := c.items[key]; exists {
		return node.value, true
	}

	return nil, false
}

// Internal helper methods

// moveToFront moves a node to the front of the list (most recently used)
func (c *LRUCache) moveToFront(node *cacheNode) {
	c.removeNode(node)
	c.addToFront(node)
}

// addToFront adds a node right after the head (most recently used position)
func (c *LRUCache) addToFront(node *cacheNode) {
	node.prev = c.head
	node.next = c.head.next
	c.head.next.prev = node
	c.head.next = node
}

// removeNode removes a node from the doubly-linked list
func (c *LRUCache) removeNode(node *cacheNode) {
	node.prev.next = node.next
	node.next.prev = node.prev
}

// evictLRU removes the least recently used item
func (c *LRUCache) evictLRU() {
	lru := c.tail.prev
	if lru != c.head {
		c.removeNode(lru)
		delete(c.items, lru.key)
	}
}

// CacheStats provides statistics about cache performance
type CacheStats struct {
	Hits     int64   `json:"hits"`
	Misses   int64   `json:"misses"`
	HitRate  float64 `json:"hit_rate_percent"`
	Size     int     `json:"current_size"`
	Capacity int     `json:"max_capacity"`
}
