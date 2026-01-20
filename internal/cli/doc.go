// Package cli implements the command-line interface for vga-events.
//
// The cli package provides the Cobra-based CLI with support for checking state events,
// formatting output (text/JSON), sorting (by date/state/title), and managing snapshots.
// It coordinates the scraper, storage, and event packages to fetch, persist, and report
// on newly-added VGA Golf state events.
package cli
