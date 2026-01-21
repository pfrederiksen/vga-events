package course

import (
	"time"
)

// CourseCache manages cached course information with TTL
type CourseCache struct {
	Courses  map[string]*CourseInfo `json:"courses"`   // normalized_title+state → info
	CachedAt map[string]time.Time   `json:"cached_at"` // key → cache time
	TTL      time.Duration          `json:"-"`         // Cache TTL (not serialized)
}

// NewCourseCache creates a new course cache with default 7-day TTL
func NewCourseCache() *CourseCache {
	return &CourseCache{
		Courses:  make(map[string]*CourseInfo),
		CachedAt: make(map[string]time.Time),
		TTL:      7 * 24 * time.Hour, // 7 days
	}
}

// Get retrieves course info from cache if not expired
// Returns nil if not found or expired
func (c *CourseCache) Get(title, state string) *CourseInfo {
	key := cacheKey(title, state)

	info, exists := c.Courses[key]
	if !exists {
		return nil
	}

	// Check if expired
	cachedTime, hasTime := c.CachedAt[key]
	if !hasTime || time.Since(cachedTime) > c.TTL {
		// Expired, remove from cache
		delete(c.Courses, key)
		delete(c.CachedAt, key)
		return nil
	}

	return info
}

// Set stores course info in cache
func (c *CourseCache) Set(title, state string, info *CourseInfo) {
	key := cacheKey(title, state)
	c.Courses[key] = info
	c.CachedAt[key] = time.Now()
}

// cacheKey generates a cache key from title and state
func cacheKey(title, state string) string {
	normalized := normalizeTitle(title)
	return normalized + "|" + state
}

// CleanExpired removes expired entries from cache
func (c *CourseCache) CleanExpired() int {
	removed := 0
	now := time.Now()

	for key, cachedTime := range c.CachedAt {
		if now.Sub(cachedTime) > c.TTL {
			delete(c.Courses, key)
			delete(c.CachedAt, key)
			removed++
		}
	}

	return removed
}

// Size returns the number of cached entries
func (c *CourseCache) Size() int {
	return len(c.Courses)
}
