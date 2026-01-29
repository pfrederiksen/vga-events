package course

import (
	"sync"
	"time"
)

// CachedCourse represents a cached course with expiration
type CachedCourse struct {
	Info      *CourseInfo `json:"info"`
	CachedAt  int64       `json:"cached_at"`  // Unix timestamp
	ExpiresAt int64       `json:"expires_at"` // Unix timestamp
}

// Cache stores course information with TTL
type Cache struct {
	mu      sync.RWMutex
	Courses map[string]*CachedCourse `json:"Courses"` // Key: "courseName|city|state"
	TTL     time.Duration            `json:"ttl"`     // How long to cache
}

// NewCache creates a new course cache with 30-day TTL
func NewCache() *Cache {
	return &Cache{
		Courses: make(map[string]*CachedCourse),
		TTL:     30 * 24 * time.Hour, // 30 days
	}
}

// cacheKey generates a cache key from course search parameters
func cacheKey(courseName, city, state string) string {
	return courseName + "|" + city + "|" + state
}

// Get retrieves a course from cache if it exists and hasn't expired
func (c *Cache) Get(courseName, city, state string) *CourseInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := cacheKey(courseName, city, state)
	cached, exists := c.Courses[key]
	if !exists {
		return nil
	}

	// Check if expired
	now := time.Now().Unix()
	if cached.ExpiresAt < now {
		return nil
	}

	return cached.Info
}

// Set stores a course in the cache
func (c *Cache) Set(courseName, city, state string, info *CourseInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().Unix()
	key := cacheKey(courseName, city, state)

	c.Courses[key] = &CachedCourse{
		Info:      info,
		CachedAt:  now,
		ExpiresAt: now + int64(c.TTL.Seconds()),
	}
}

// Cleanup removes expired entries from the cache
func (c *Cache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().Unix()
	removed := 0

	for key, cached := range c.Courses {
		if cached.ExpiresAt < now {
			delete(c.Courses, key)
			removed++
		}
	}

	return removed
}

// Size returns the number of cached Courses
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.Courses)
}
