package logger

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

func TestLogger_Log(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "log-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())   // nolint:errcheck
	defer tmpFile.Close()              // nolint:errcheck

	logger := New(LevelInfo, tmpFile)

	tests := []struct {
		name    string
		level   Level
		message string
		fields  Fields
		err     error
		want    bool // should log
	}{
		{
			name:    "info message",
			level:   LevelInfo,
			message: "test message",
			fields:  Fields{"key": "value"},
			want:    true,
		},
		{
			name:    "debug below threshold",
			level:   LevelDebug,
			message: "debug message",
			want:    false, // won't log (below INFO)
		},
		{
			name:    "error with err",
			level:   LevelError,
			message: "error occurred",
			err:     errors.New("test error"),
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, _ := tmpFile.Seek(0, 2) // Get current position

			logger.log(tt.level, tt.message, tt.fields, tt.err)

			after, _ := tmpFile.Seek(0, 2) // Get new position
			logged := after > before

			if logged != tt.want {
				t.Errorf("log() logged = %v, want %v", logged, tt.want)
			}
		})
	}
}

func TestLogEntry_JSON(t *testing.T) {
	entry := LogEntry{
		Timestamp: "2026-01-01T00:00:00Z",
		Level:     "INFO",
		Message:   "test message",
		Fields: Fields{
			"user_id": "123",
			"action":  "subscribe",
		},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded LogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.Message != entry.Message {
		t.Errorf("Message = %v, want %v", decoded.Message, entry.Message)
	}
}

func TestMetrics_Counter(t *testing.T) {
	m := NewMetrics()

	m.IncrCounter("test_counter")
	m.IncrCounter("test_counter")
	m.IncrCounter("test_counter")

	snapshot := m.GetSnapshot()
	counters := snapshot["counters"].(map[string]int64)

	if counters["test_counter"] != 3 {
		t.Errorf("Counter = %v, want 3", counters["test_counter"])
	}
}

func TestMetrics_Gauge(t *testing.T) {
	m := NewMetrics()

	m.SetGauge("memory_mb", 512.5)
	m.SetGauge("memory_mb", 1024.0)

	snapshot := m.GetSnapshot()
	gauges := snapshot["gauges"].(map[string]float64)

	if gauges["memory_mb"] != 1024.0 {
		t.Errorf("Gauge = %v, want 1024.0", gauges["memory_mb"])
	}
}

func TestMetrics_Timing(t *testing.T) {
	m := NewMetrics()

	m.RecordTiming("api_call", 100*time.Millisecond)
	m.RecordTiming("api_call", 200*time.Millisecond)
	m.RecordTiming("api_call", 150*time.Millisecond)

	snapshot := m.GetSnapshot()
	timings := snapshot["timings"].(map[string]map[string]interface{})

	apiTiming := timings["api_call"]
	if apiTiming["count"].(int) != 3 {
		t.Errorf("Timing count = %v, want 3", apiTiming["count"])
	}

	if apiTiming["min"].(string) != "100ms" {
		t.Errorf("Min timing = %v, want 100ms", apiTiming["min"])
	}

	if apiTiming["max"].(string) != "200ms" {
		t.Errorf("Max timing = %v, want 200ms", apiTiming["max"])
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	// Test that package-level functions don't panic
	Info("test info", Fields{"key": "value"})
	Warn("test warning", nil)
	Error("test error", Fields{"component": "test"}, errors.New("test"))

	IncrCounter("test")
	SetGauge("test", 42.0)
	RecordTiming("test", time.Second)

	snapshot := GetMetricsSnapshot()
	if snapshot == nil {
		t.Error("GetMetricsSnapshot() returned nil")
	}
}

func TestLogger_Levels(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "log-test-*")
	defer os.Remove(tmpFile.Name())   // nolint:errcheck
	defer tmpFile.Close()              // nolint:errcheck

	tests := []struct {
		name      string
		minLevel  Level
		logLevel  Level
		shouldLog bool
	}{
		{"debug logs at debug", LevelDebug, LevelDebug, true},
		{"info logs at debug", LevelDebug, LevelInfo, true},
		{"debug doesn't log at info", LevelInfo, LevelDebug, false},
		{"error always logs", LevelDebug, LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.minLevel, tmpFile)
			before, _ := tmpFile.Seek(0, 2)

			logger.log(tt.logLevel, "test", nil, nil)

			after, _ := tmpFile.Seek(0, 2)
			logged := after > before

			if logged != tt.shouldLog {
				t.Errorf("shouldLog = %v, want %v", logged, tt.shouldLog)
			}
		})
	}
}
