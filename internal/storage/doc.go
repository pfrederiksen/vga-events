// Package storage provides JSON-based persistence for event snapshots.
//
// The storage package manages local snapshot files that track events across runs.
// Snapshots are stored in JSON format, with separate files for each state
// (snapshot_STATE.json) and a combined file for all states (snapshot.json).
// The default storage location is ~/.local/share/vga-events/.
package storage
