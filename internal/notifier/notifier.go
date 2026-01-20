package notifier

import (
	"github.com/pfrederiksen/vga-events/internal/event"
)

// Notifier defines the interface for posting event notifications
type Notifier interface {
	// Notify posts notifications for the given events
	Notify(events []*event.Event) error
}
