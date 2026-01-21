package course

import (
	"testing"
	"time"
)

func TestCourseCache(t *testing.T) {
	cache := NewCourseCache()

	t.Run("new cache is empty", func(t *testing.T) {
		if cache.Size() != 0 {
			t.Errorf("new cache size = %d, want 0", cache.Size())
		}
	})

	t.Run("set and get", func(t *testing.T) {
		info := &CourseInfo{
			Name:    "Test Course",
			City:    "Test City",
			State:   "CA",
			Yardage: 7000,
			Par:     72,
			Source:  "test",
		}

		cache.Set("Test Course", "CA", info)

		got := cache.Get("Test Course", "CA")
		if got == nil {
			t.Fatal("Get returned nil, expected course info")
		}

		if got.Name != info.Name {
			t.Errorf("Get().Name = %q, want %q", got.Name, info.Name)
		}
	})

	t.Run("get non-existent returns nil", func(t *testing.T) {
		got := cache.Get("Unknown Course", "NY")
		if got != nil {
			t.Errorf("Get(unknown) = %v, want nil", got)
		}
	})

	t.Run("expired entries return nil", func(t *testing.T) {
		cache := NewCourseCache()
		cache.TTL = 1 * time.Millisecond // Very short TTL for testing

		info := &CourseInfo{
			Name:   "Expiring Course",
			Source: "test",
		}

		cache.Set("Expiring Course", "NV", info)

		// Should exist immediately
		got := cache.Get("Expiring Course", "NV")
		if got == nil {
			t.Fatal("Get immediately after Set returned nil")
		}

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// Should be expired
		got = cache.Get("Expiring Course", "NV")
		if got != nil {
			t.Errorf("Get after expiration = %v, want nil", got)
		}
	})

	t.Run("CleanExpired removes expired entries", func(t *testing.T) {
		cache := NewCourseCache()
		cache.TTL = 1 * time.Millisecond

		// Add entries
		for i := 0; i < 5; i++ {
			info := &CourseInfo{Name: "Course", Source: "test"}
			cache.Set("Course", string(rune('A'+i)), info)
		}

		if cache.Size() != 5 {
			t.Errorf("cache size after adds = %d, want 5", cache.Size())
		}

		// Wait for expiration
		time.Sleep(10 * time.Millisecond)

		// Clean expired
		removed := cache.CleanExpired()
		if removed != 5 {
			t.Errorf("CleanExpired removed %d, want 5", removed)
		}

		if cache.Size() != 0 {
			t.Errorf("cache size after clean = %d, want 0", cache.Size())
		}
	})
}

func TestCacheKey(t *testing.T) {
	tests := []struct {
		title1, state1 string
		title2, state2 string
		shouldMatch    bool
	}{
		{
			title1:      "Pebble Beach",
			state1:      "CA",
			title2:      "Pebble Beach",
			state2:      "CA",
			shouldMatch: true,
		},
		{
			title1:      "Pebble Beach",
			state1:      "CA",
			title2:      "Pebble Beach",
			state2:      "NV",
			shouldMatch: false,
		},
		{
			title1:      "Pebble Beach Golf Links",
			state1:      "CA",
			title2:      "Pebble Beach",
			state2:      "CA",
			shouldMatch: true, // Normalization should make these the same
		},
	}

	for _, tt := range tests {
		key1 := cacheKey(tt.title1, tt.state1)
		key2 := cacheKey(tt.title2, tt.state2)

		match := key1 == key2
		if match != tt.shouldMatch {
			t.Errorf("cacheKey(%q, %q) vs cacheKey(%q, %q): match=%v, want %v",
				tt.title1, tt.state1, tt.title2, tt.state2, match, tt.shouldMatch)
		}
	}
}
