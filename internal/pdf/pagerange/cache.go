package pagerange

import (
	"container/list"
	"fmt"
	"sync"
	"time"
)

// PageObjectCache provides efficient caching of PDF objects with size limits and LRU eviction
type PageObjectCache struct {
	// Core cache data
	objects   map[string]*CachedObject // Key: "objID_generation"
	lruList   *list.List               // LRU list for eviction
	keyToNode map[string]*list.Element // Fast lookup from key to list node

	// Size management
	currentSize int64
	maxSize     int64
	maxObjects  int

	// Statistics
	stats CacheStats
	mutex sync.RWMutex

	// Configuration
	config CacheConfig
}

// CachedObject represents a cached PDF object with metadata
type CachedObject struct {
	Key         string      `json:"key"`
	ObjectRef   ObjectRef   `json:"object_ref"`
	Content     interface{} `json:"content"`
	Size        int64       `json:"size"`
	AccessTime  int64       `json:"access_time"`
	CreateTime  int64       `json:"create_time"`
	AccessCount int         `json:"access_count"`
	ObjectType  string      `json:"object_type"`
}

// CacheConfig configures the cache behavior
type CacheConfig struct {
	MaxSizeBytes   int64         `json:"max_size_bytes"`
	MaxObjects     int           `json:"max_objects"`
	TTL            time.Duration `json:"ttl"`
	EnableTTL      bool          `json:"enable_ttl"`
	EnableStats    bool          `json:"enable_stats"`
	EvictionPolicy string        `json:"eviction_policy"` // "lru", "lfu", "ttl"
}

// CacheStats provides cache performance statistics
type CacheStats struct {
	Hits              int64   `json:"hits"`
	Misses            int64   `json:"misses"`
	Evictions         int64   `json:"evictions"`
	ObjectCount       int     `json:"object_count"`
	TotalSize         int64   `json:"total_size"`
	HitRate           float64 `json:"hit_rate"`
	AverageObjectSize int64   `json:"average_object_size"`
	MemoryEfficiency  float64 `json:"memory_efficiency"`
}

// LRUNode represents a node in the LRU list
type LRUNode struct {
	Key    string
	Object *CachedObject
}

// DefaultCacheConfig returns sensible defaults for the cache
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxSizeBytes:   50 * 1024 * 1024, // 50MB
		MaxObjects:     1000,             // 1000 objects max
		TTL:            30 * time.Minute, // 30 minute TTL
		EnableTTL:      false,            // Disabled by default
		EnableStats:    true,
		EvictionPolicy: "lru",
	}
}

// NewPageObjectCache creates a new page object cache
func NewPageObjectCache(maxSizeBytes int64, config ...CacheConfig) *PageObjectCache {
	cfg := DefaultCacheConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	if maxSizeBytes > 0 {
		cfg.MaxSizeBytes = maxSizeBytes
	}

	return &PageObjectCache{
		objects:    make(map[string]*CachedObject),
		lruList:    list.New(),
		keyToNode:  make(map[string]*list.Element),
		maxSize:    cfg.MaxSizeBytes,
		maxObjects: cfg.MaxObjects,
		config:     cfg,
		stats: CacheStats{
			ObjectCount: 0,
			TotalSize:   0,
		},
	}
}

// Put stores an object in the cache
func (poc *PageObjectCache) Put(objRef ObjectRef, content interface{}) error {
	poc.mutex.Lock()
	defer poc.mutex.Unlock()

	key := poc.makeKey(objRef)

	// Calculate object size
	size := poc.calculateSize(content)

	// Check if object already exists
	if existingObj, exists := poc.objects[key]; exists {
		// Update existing object
		existingObj.Content = content
		existingObj.Size = size
		existingObj.AccessTime = getCurrentTimeMillis()
		existingObj.AccessCount++

		// Move to front of LRU list
		if node, exists := poc.keyToNode[key]; exists {
			poc.lruList.MoveToFront(node)
		}

		return nil
	}

	// Create new cached object
	cachedObj := &CachedObject{
		Key:         key,
		ObjectRef:   objRef,
		Content:     content,
		Size:        size,
		AccessTime:  getCurrentTimeMillis(),
		CreateTime:  getCurrentTimeMillis(),
		AccessCount: 1,
		ObjectType:  poc.detectObjectType(content),
	}

	// Check size limits and evict if necessary
	if err := poc.ensureSpace(size); err != nil {
		return fmt.Errorf("failed to ensure space: %w", err)
	}

	// Add to cache
	poc.objects[key] = cachedObj
	node := poc.lruList.PushFront(&LRUNode{
		Key:    key,
		Object: cachedObj,
	})
	poc.keyToNode[key] = node

	// Update statistics
	poc.currentSize += size
	poc.stats.ObjectCount++
	poc.stats.TotalSize = poc.currentSize

	return nil
}

// Get retrieves an object from the cache
func (poc *PageObjectCache) Get(objRef ObjectRef) interface{} {
	poc.mutex.Lock()
	defer poc.mutex.Unlock()

	key := poc.makeKey(objRef)

	if obj, exists := poc.objects[key]; exists {
		// Update access statistics
		obj.AccessTime = getCurrentTimeMillis()
		obj.AccessCount++

		// Move to front of LRU list
		if node, exists := poc.keyToNode[key]; exists {
			poc.lruList.MoveToFront(node)
		}

		if poc.config.EnableStats {
			poc.stats.Hits++
		}

		return obj.Content
	}

	if poc.config.EnableStats {
		poc.stats.Misses++
	}

	return nil
}

// Contains checks if an object exists in the cache without updating access time
func (poc *PageObjectCache) Contains(objRef ObjectRef) bool {
	poc.mutex.RLock()
	defer poc.mutex.RUnlock()

	key := poc.makeKey(objRef)
	_, exists := poc.objects[key]
	return exists
}

// Remove removes an object from the cache
func (poc *PageObjectCache) Remove(objRef ObjectRef) bool {
	poc.mutex.Lock()
	defer poc.mutex.Unlock()

	key := poc.makeKey(objRef)
	return poc.removeByKey(key)
}

// Clear removes all objects from the cache
func (poc *PageObjectCache) Clear() {
	poc.mutex.Lock()
	defer poc.mutex.Unlock()

	poc.objects = make(map[string]*CachedObject)
	poc.lruList = list.New()
	poc.keyToNode = make(map[string]*list.Element)
	poc.currentSize = 0
	poc.stats = CacheStats{}
}

// GetStats returns current cache statistics
func (poc *PageObjectCache) GetStats() CacheStats {
	poc.mutex.RLock()
	defer poc.mutex.RUnlock()

	stats := poc.stats

	// Calculate derived statistics
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total) * 100
	}

	if stats.ObjectCount > 0 {
		stats.AverageObjectSize = stats.TotalSize / int64(stats.ObjectCount)
	}

	if poc.maxSize > 0 {
		stats.MemoryEfficiency = float64(stats.TotalSize) / float64(poc.maxSize) * 100
	}

	return stats
}

// GetSize returns the current cache size in bytes
func (poc *PageObjectCache) GetSize() int64 {
	poc.mutex.RLock()
	defer poc.mutex.RUnlock()
	return poc.currentSize
}

// GetObjectCount returns the number of objects in the cache
func (poc *PageObjectCache) GetObjectCount() int {
	poc.mutex.RLock()
	defer poc.mutex.RUnlock()
	return len(poc.objects)
}

// GetCapacity returns the maximum cache size
func (poc *PageObjectCache) GetCapacity() int64 {
	return poc.maxSize
}

// ListObjects returns a list of all cached objects (for debugging)
func (poc *PageObjectCache) ListObjects() []CachedObject {
	poc.mutex.RLock()
	defer poc.mutex.RUnlock()

	objects := make([]CachedObject, 0, len(poc.objects))
	for _, obj := range poc.objects {
		objects = append(objects, *obj)
	}

	return objects
}

// GetMostAccessed returns the most frequently accessed objects
func (poc *PageObjectCache) GetMostAccessed(limit int) []CachedObject {
	poc.mutex.RLock()
	defer poc.mutex.RUnlock()

	objects := make([]CachedObject, 0, len(poc.objects))
	for _, obj := range poc.objects {
		objects = append(objects, *obj)
	}

	// Sort by access count (descending)
	for i := 0; i < len(objects)-1; i++ {
		for j := i + 1; j < len(objects); j++ {
			if objects[i].AccessCount < objects[j].AccessCount {
				objects[i], objects[j] = objects[j], objects[i]
			}
		}
	}

	if limit > 0 && limit < len(objects) {
		objects = objects[:limit]
	}

	return objects
}

// EvictExpired removes expired objects (if TTL is enabled)
func (poc *PageObjectCache) EvictExpired() int {
	if !poc.config.EnableTTL {
		return 0
	}

	poc.mutex.Lock()
	defer poc.mutex.Unlock()

	currentTime := getCurrentTimeMillis()
	ttlMs := poc.config.TTL.Milliseconds()
	evicted := 0

	for key, obj := range poc.objects {
		if currentTime-obj.AccessTime > ttlMs {
			poc.removeByKey(key)
			evicted++
		}
	}

	return evicted
}

// Internal helper methods

// makeKey creates a cache key from an object reference
func (poc *PageObjectCache) makeKey(objRef ObjectRef) string {
	return fmt.Sprintf("%d_%d", objRef.ObjectID, objRef.Generation)
}

// calculateSize estimates the memory size of an object
func (poc *PageObjectCache) calculateSize(content interface{}) int64 {
	switch v := content.(type) {
	case string:
		return int64(len(v))
	case []byte:
		return int64(len(v))
	case map[string]interface{}:
		// Rough estimation for complex objects
		return int64(len(fmt.Sprintf("%+v", v)))
	default:
		// Default estimation
		return int64(len(fmt.Sprintf("%+v", v)))
	}
}

// detectObjectType attempts to detect the type of PDF object
func (poc *PageObjectCache) detectObjectType(content interface{}) string {
	if str, ok := content.(string); ok {
		if contains(str, "/Type /Page") {
			return "Page"
		} else if contains(str, "/Type /Pages") {
			return "Pages"
		} else if contains(str, "/Subtype /Image") {
			return "Image"
		} else if contains(str, "/FT /") {
			return "Form"
		} else if contains(str, "stream") {
			return "Stream"
		}
	}
	return "Unknown"
}

// ensureSpace ensures there's enough space for a new object
func (poc *PageObjectCache) ensureSpace(neededSize int64) error {
	// Check size limit
	for poc.currentSize+neededSize > poc.maxSize && poc.lruList.Len() > 0 {
		if !poc.evictLRU() {
			return fmt.Errorf("failed to evict object for space")
		}
	}

	// Check object count limit
	for len(poc.objects) >= poc.maxObjects && poc.lruList.Len() > 0 {
		if !poc.evictLRU() {
			return fmt.Errorf("failed to evict object for count limit")
		}
	}

	return nil
}

// evictLRU evicts the least recently used object
func (poc *PageObjectCache) evictLRU() bool {
	if poc.lruList.Len() == 0 {
		return false
	}

	// Get least recently used item (back of list)
	element := poc.lruList.Back()
	if element == nil {
		return false
	}

	lruNode := element.Value.(*LRUNode)
	return poc.removeByKey(lruNode.Key)
}

// removeByKey removes an object by its key
func (poc *PageObjectCache) removeByKey(key string) bool {
	obj, exists := poc.objects[key]
	if !exists {
		return false
	}

	// Remove from maps
	delete(poc.objects, key)

	// Remove from LRU list
	if node, exists := poc.keyToNode[key]; exists {
		poc.lruList.Remove(node)
		delete(poc.keyToNode, key)
	}

	// Update statistics
	poc.currentSize -= obj.Size
	poc.stats.ObjectCount--
	poc.stats.TotalSize = poc.currentSize
	if poc.config.EnableStats {
		poc.stats.Evictions++
	}

	return true
}

// Utility function for string containment check
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					someContains(s, substr)))
}

func someContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Background maintenance methods

// StartMaintenance starts background maintenance routines
func (poc *PageObjectCache) StartMaintenance() {
	if poc.config.EnableTTL {
		go poc.maintenanceLoop()
	}
}

// maintenanceLoop runs periodic maintenance tasks
func (poc *PageObjectCache) maintenanceLoop() {
	ticker := time.NewTicker(time.Minute) // Run every minute
	defer ticker.Stop()

	for range ticker.C {
		poc.EvictExpired()
	}
}

// CacheMetrics provides detailed cache metrics
type CacheMetrics struct {
	Stats        CacheStats           `json:"stats"`
	ObjectTypes  map[string]int       `json:"object_types"`
	SizeByType   map[string]int64     `json:"size_by_type"`
	AccessCounts map[string]int       `json:"access_counts"`
	MemoryUsage  MemoryUsageBreakdown `json:"memory_usage"`
}

// MemoryUsageBreakdown provides detailed memory usage information
type MemoryUsageBreakdown struct {
	TotalBytes      int64   `json:"total_bytes"`
	ObjectBytes     int64   `json:"object_bytes"`
	MetadataBytes   int64   `json:"metadata_bytes"`
	OverheadBytes   int64   `json:"overhead_bytes"`
	UtilizationRate float64 `json:"utilization_rate"`
}

// GetDetailedMetrics returns comprehensive cache metrics
func (poc *PageObjectCache) GetDetailedMetrics() CacheMetrics {
	poc.mutex.RLock()
	defer poc.mutex.RUnlock()

	metrics := CacheMetrics{
		Stats:        poc.GetStats(),
		ObjectTypes:  make(map[string]int),
		SizeByType:   make(map[string]int64),
		AccessCounts: make(map[string]int),
	}

	// Analyze objects by type
	for _, obj := range poc.objects {
		metrics.ObjectTypes[obj.ObjectType]++
		metrics.SizeByType[obj.ObjectType] += obj.Size

		// Categorize by access frequency
		var accessCategory string
		if obj.AccessCount > 10 {
			accessCategory = "high"
		} else if obj.AccessCount > 3 {
			accessCategory = "medium"
		} else {
			accessCategory = "low"
		}
		metrics.AccessCounts[accessCategory]++
	}

	// Calculate memory usage breakdown
	objectBytes := poc.currentSize
	metadataBytes := int64(len(poc.objects)) * 200  // Rough estimate for metadata
	overheadBytes := int64(len(poc.keyToNode)) * 50 // Rough estimate for LRU overhead
	totalBytes := objectBytes + metadataBytes + overheadBytes

	metrics.MemoryUsage = MemoryUsageBreakdown{
		TotalBytes:      totalBytes,
		ObjectBytes:     objectBytes,
		MetadataBytes:   metadataBytes,
		OverheadBytes:   overheadBytes,
		UtilizationRate: float64(objectBytes) / float64(totalBytes) * 100,
	}

	return metrics
}
