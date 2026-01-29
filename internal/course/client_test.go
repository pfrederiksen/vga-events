package course

import (
	"testing"
	"time"
)

func TestCleanCourseName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "course with date suffix",
			input:    "Stallion Mountain 1.25.26",
			expected: "Stallion Mountain",
		},
		{
			name:     "course with slash date",
			input:    "Pebble Beach 12/15/25",
			expected: "Pebble Beach",
		},
		{
			name:     "course without date",
			input:    "Augusta National",
			expected: "Augusta National",
		},
		{
			name:     "course with extra spaces",
			input:    "  TPC Sawgrass  ",
			expected: "TPC Sawgrass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanCourseName(tt.input)
			if result != tt.expected {
				t.Errorf("CleanCourseName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCache(t *testing.T) {
	t.Run("NewCache creates cache with correct TTL", func(t *testing.T) {
		cache := NewCache()

		if cache == nil {
			t.Fatal("NewCache returned nil")
		}
		expectedTTL := 30 * 24 * time.Hour
		if cache.TTL != expectedTTL {
			t.Errorf("cache.TTL = %v, want %v", cache.TTL, expectedTTL)
		}
		if cache.Size() != 0 {
			t.Errorf("new cache size = %d, want 0", cache.Size())
		}
	})

	t.Run("Set and Get course", func(t *testing.T) {
		cache := NewCache()
		course := &CourseInfo{
			ID:         123,
			ClubName:   "Test Club",
			CourseName: "Test Course",
		}

		// Set course
		cache.Set("Test Course", "Las Vegas", "NV", course)

		// Get course
		retrieved := cache.Get("Test Course", "Las Vegas", "NV")
		if retrieved == nil {
			t.Fatal("Get returned nil")
		}
		if retrieved.ID != course.ID {
			t.Errorf("retrieved.ID = %d, want %d", retrieved.ID, course.ID)
		}
		if retrieved.ClubName != course.ClubName {
			t.Errorf("retrieved.ClubName = %q, want %q", retrieved.ClubName, course.ClubName)
		}
	})

	t.Run("Get non-existent course returns nil", func(t *testing.T) {
		cache := NewCache()
		result := cache.Get("Nonexistent", "City", "ST")
		if result != nil {
			t.Errorf("Get for non-existent course returned %v, want nil", result)
		}
	})

	t.Run("Expired entries are not returned", func(t *testing.T) {
		cache := NewCache()
		course := &CourseInfo{
			ID:         456,
			CourseName: "Expired Course",
		}

		// Manually create an expired entry
		cache.mu.Lock()
		cache.Courses["Expired Course|City|ST"] = &CachedCourse{
			Info:      course,
			CachedAt:  time.Now().Unix() - 3600,
			ExpiresAt: time.Now().Unix() - 1, // Expired 1 second ago
		}
		cache.mu.Unlock()

		result := cache.Get("Expired Course", "City", "ST")
		if result != nil {
			t.Errorf("Get for expired course returned %v, want nil", result)
		}
	})

	t.Run("Cleanup removes expired entries", func(t *testing.T) {
		cache := NewCache()

		// Manually add an expired course
		course := &CourseInfo{ID: 789, CourseName: "Test"}
		cache.mu.Lock()
		cache.Courses["Test|City|ST"] = &CachedCourse{
			Info:      course,
			CachedAt:  time.Now().Unix() - 3600,
			ExpiresAt: time.Now().Unix() - 1, // Expired
		}
		cache.mu.Unlock()

		if cache.Size() != 1 {
			t.Errorf("cache.Size() = %d, want 1", cache.Size())
		}

		// Run cleanup
		removed := cache.Cleanup()

		if removed != 1 {
			t.Errorf("Cleanup removed %d entries, want 1", removed)
		}
		if cache.Size() != 0 {
			t.Errorf("cache.Size() after cleanup = %d, want 0", cache.Size())
		}
	})

	t.Run("Cache with nil course info", func(t *testing.T) {
		cache := NewCache()

		// Set nil course (for negative caching)
		cache.Set("", "City", "ST", nil)

		if cache.Size() != 1 {
			t.Errorf("cache.Size() = %d, want 1", cache.Size())
		}

		// Should return nil but from cache
		result := cache.Get("", "City", "ST")
		if result != nil {
			t.Errorf("Get for nil course returned %v, want nil", result)
		}
	})
}

func TestCourseInfo(t *testing.T) {
	t.Run("GetDisplayName with both club and course", func(t *testing.T) {
		info := &CourseInfo{
			ClubName:   "Pebble Beach",
			CourseName: "Pebble Beach Golf Links",
		}

		expected := "Pebble Beach - Pebble Beach Golf Links"
		if info.GetDisplayName() != expected {
			t.Errorf("GetDisplayName() = %q, want %q", info.GetDisplayName(), expected)
		}
	})

	t.Run("GetDisplayName with only course name", func(t *testing.T) {
		info := &CourseInfo{
			CourseName: "Augusta National",
		}

		// When ClubName is empty, it formats as " - CourseName"
		// This is the current behavior
		expected := " - Augusta National"
		if info.GetDisplayName() != expected {
			t.Errorf("GetDisplayName() = %q, want %q", info.GetDisplayName(), expected)
		}
	})

	t.Run("GetDisplayName with only club name", func(t *testing.T) {
		info := &CourseInfo{
			ClubName: "Torrey Pines",
		}

		expected := "Torrey Pines"
		if info.GetDisplayName() != expected {
			t.Errorf("GetDisplayName() = %q, want %q", info.GetDisplayName(), expected)
		}
	})
}

func TestNewClient(t *testing.T) {
	t.Run("NewClient creates client with cache", func(t *testing.T) {
		client := NewClient("test-api-key")

		if client == nil {
			t.Fatal("NewClient returned nil")
		}
		if client.apiKey != "test-api-key" {
			t.Errorf("client.apiKey = %q, want %q", client.apiKey, "test-api-key")
		}
		if client.cache == nil {
			t.Error("client.cache is nil, want non-nil")
		}
	})

	t.Run("NewClientWithCache uses provided cache", func(t *testing.T) {
		customCache := NewCache()
		client := NewClientWithCache("test-key", customCache)

		if client == nil {
			t.Fatal("NewClientWithCache returned nil")
		}
		if client.cache != customCache {
			t.Error("client.cache is not the provided cache")
		}
	})

	t.Run("GetCache returns client cache", func(t *testing.T) {
		client := NewClient("test-key")
		cache := client.GetCache()

		if cache == nil {
			t.Error("GetCache returned nil")
		}
		if cache != client.cache {
			t.Error("GetCache returned different cache than client.cache")
		}
	})
}
