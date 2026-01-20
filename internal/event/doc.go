// Package event provides types and functions for managing VGA Golf state events.
//
// The event package handles event representation, identification, and change detection
// through snapshot-based diffing. Each event is assigned a deterministic SHA1-based ID
// generated from its state code and raw text, enabling reliable tracking across runs.
package event
