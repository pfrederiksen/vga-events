// Package logger provides structured JSON logging and metrics tracking for the VGA Events bot.
//
// The logger supports multiple log levels (DEBUG, INFO, WARN, ERROR) and outputs
// structured JSON for easy parsing and analysis. All logs include timestamps and
// can include arbitrary structured fields.
//
// Metrics tracking includes counters (incrementing values), gauges (point-in-time values),
// and timings (duration measurements) with automatic statistical aggregation.
//
// Example usage:
//
//	logger.Info("User subscribed", logger.Fields{
//	    "user_id": "123456",
//	    "state": "NV",
//	})
//
//	logger.Error("API request failed", logger.Fields{
//	    "endpoint": "/events",
//	    "retry_count": 3,
//	}, err)
//
//	logger.IncrCounter("commands.subscribe")
//	logger.RecordTiming("api.fetch", duration)
package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Level represents log severity
type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// Logger provides structured logging
type Logger struct {
	minLevel Level
	output   *os.File
}

// Fields represents structured log fields
type Fields map[string]interface{}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
	Fields    Fields `json:"fields,omitempty"`
	Error     string `json:"error,omitempty"`
}

var defaultLogger *Logger

func init() {
	defaultLogger = New(LevelInfo, os.Stdout)
}

// New creates a new logger with the specified minimum log level and output destination.
// Messages below the minimum level will be discarded.
func New(level Level, output *os.File) *Logger {
	return &Logger{
		minLevel: level,
		output:   output,
	}
}

// SetDefault sets the default package-level logger used by the convenience functions
// (Debug, Info, Warn, Error). This allows centralizing logger configuration.
func SetDefault(logger *Logger) {
	defaultLogger = logger
}

// log writes a structured log entry
func (l *Logger) log(level Level, message string, fields Fields, err error) {
	// Check if we should log this level
	if !l.shouldLog(level) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     string(level),
		Message:   message,
		Fields:    fields,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	// Marshal to JSON
	data, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		// Fallback to plain text if JSON marshal fails
		fmt.Fprintf(l.output, "[%s] %s: %s (marshal error: %v)\n",
			entry.Timestamp, entry.Level, entry.Message, marshalErr)
		return
	}

	fmt.Fprintln(l.output, string(data))
}

// shouldLog determines if a message should be logged based on level
func (l *Logger) shouldLog(level Level) bool {
	levels := map[Level]int{
		LevelDebug: 0,
		LevelInfo:  1,
		LevelWarn:  2,
		LevelError: 3,
	}
	return levels[level] >= levels[l.minLevel]
}

// Debug logs a debug message with optional structured fields.
// Debug messages are typically used for detailed diagnostic information.
func (l *Logger) Debug(message string, fields Fields) {
	l.log(LevelDebug, message, fields, nil)
}

// Info logs an informational message with optional structured fields.
// Info messages are used for general operational information.
func (l *Logger) Info(message string, fields Fields) {
	l.log(LevelInfo, message, fields, nil)
}

// Warn logs a warning message with optional structured fields.
// Warning messages indicate potential issues that don't prevent operation.
func (l *Logger) Warn(message string, fields Fields) {
	l.log(LevelWarn, message, fields, nil)
}

// Error logs an error message with optional structured fields and an error object.
// Error messages indicate failures that prevent normal operation.
func (l *Logger) Error(message string, fields Fields, err error) {
	l.log(LevelError, message, fields, err)
}

// Package-level convenience functions using default logger

// Debug logs a debug message with the default logger
func Debug(message string, fields Fields) {
	defaultLogger.Debug(message, fields)
}

// Info logs an info message with the default logger
func Info(message string, fields Fields) {
	defaultLogger.Info(message, fields)
}

// Warn logs a warning message with the default logger
func Warn(message string, fields Fields) {
	defaultLogger.Warn(message, fields)
}

// Error logs an error message with the default logger
func Error(message string, fields Fields, err error) {
	defaultLogger.Error(message, fields, err)
}

// Metrics tracks operational metrics including counters, gauges, and timings.
// All operations are thread-safe.
//
// Counters track incrementing values (e.g., number of commands processed).
// Gauges track point-in-time values (e.g., number of active users).
// Timings track durations and automatically compute min/max/average statistics.
type Metrics struct {
	mu       sync.Mutex
	counters map[string]int64
	gauges   map[string]float64
	timings  map[string][]time.Duration
}

var defaultMetrics *Metrics

func init() {
	defaultMetrics = NewMetrics()
}

// NewMetrics creates a new metrics tracker with empty counters, gauges, and timings.
func NewMetrics() *Metrics {
	return &Metrics{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
		timings:  make(map[string][]time.Duration),
	}
}

// IncrCounter increments a counter by 1. If the counter doesn't exist, it is initialized to 1.
// Thread-safe.
func (m *Metrics) IncrCounter(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name]++
}

// SetGauge sets a gauge to the specified value, overwriting any previous value.
// Thread-safe.
func (m *Metrics) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

// RecordTiming records a duration measurement. Multiple measurements are tracked
// and statistics (count, total, average, min, max) are computed in GetSnapshot.
// Thread-safe.
func (m *Metrics) RecordTiming(name string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timings[name] = append(m.timings[name], duration)
}

// GetSnapshot returns a snapshot of all metrics as a map containing:
//   - "counters": map of counter names to values
//   - "gauges": map of gauge names to values
//   - "timings": map of timing names to statistics (count, total, average, min, max)
//
// The snapshot is a deep copy, safe to use concurrently with metric updates.
func (m *Metrics) GetSnapshot() map[string]interface{} {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot := make(map[string]interface{})

	// Copy counters
	counters := make(map[string]int64)
	for k, v := range m.counters {
		counters[k] = v
	}
	snapshot["counters"] = counters

	// Copy gauges
	gauges := make(map[string]float64)
	for k, v := range m.gauges {
		gauges[k] = v
	}
	snapshot["gauges"] = gauges

	// Calculate timing statistics
	timings := make(map[string]map[string]interface{})
	for name, durations := range m.timings {
		if len(durations) == 0 {
			continue
		}

		var total time.Duration
		min := durations[0]
		max := durations[0]

		for _, d := range durations {
			total += d
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
		}

		timings[name] = map[string]interface{}{
			"count":   len(durations),
			"total":   total.String(),
			"average": (total / time.Duration(len(durations))).String(),
			"min":     min.String(),
			"max":     max.String(),
		}
	}
	snapshot["timings"] = timings

	return snapshot
}

// Package-level metrics functions using the default metrics tracker

// IncrCounter increments a counter on the default metrics tracker.
// Convenience function equivalent to calling defaultMetrics.IncrCounter(name).
func IncrCounter(name string) {
	defaultMetrics.IncrCounter(name)
}

// SetGauge sets a gauge on the default metrics tracker.
// Convenience function equivalent to calling defaultMetrics.SetGauge(name, value).
func SetGauge(name string, value float64) {
	defaultMetrics.SetGauge(name, value)
}

// RecordTiming records a timing on the default metrics tracker.
// Convenience function equivalent to calling defaultMetrics.RecordTiming(name, duration).
func RecordTiming(name string, duration time.Duration) {
	defaultMetrics.RecordTiming(name, duration)
}

// GetMetricsSnapshot returns a snapshot of all metrics from the default tracker.
// Convenience function equivalent to calling defaultMetrics.GetSnapshot().
func GetMetricsSnapshot() map[string]interface{} {
	return defaultMetrics.GetSnapshot()
}
