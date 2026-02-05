package telegram

import (
	"fmt"
	"strings"

	"github.com/pfrederiksen/vga-events/internal/event"
)

// formatChangeValue formats old/new value display for event changes
func formatChangeValue(msg *strings.Builder, oldValue, newValue, fieldName string) {
	// Show old value
	if oldValue != "" {
		msg.WriteString(fmt.Sprintf("  ❌ <s>%s</s>\n", oldValue))
	} else {
		msg.WriteString(fmt.Sprintf("  ❌ <s>No %s</s>\n", fieldName))
	}

	// Show new value
	if newValue != "" {
		msg.WriteString(fmt.Sprintf("  ✅ %s\n", newValue))
	} else {
		msg.WriteString(fmt.Sprintf("  ✅ No %s\n", fieldName))
	}
}

// formatEventHeader writes common event header fields to a message builder
func formatEventHeader(msg *strings.Builder, evt *event.Event, hasNote bool) {
	if hasNote {
		msg.WriteString("🏌️ 📝 <b>New VGA Golf Event!</b>\n\n")
	} else {
		msg.WriteString("🏌️ <b>New VGA Golf Event!</b>\n\n")
	}

	msg.WriteString(fmt.Sprintf("📍 <b>%s</b> - %s\n", evt.State, evt.Title))

	if len(evt.AlsoIn) > 0 {
		msg.WriteString(fmt.Sprintf("   <i>Also in: %s</i>\n", strings.Join(evt.AlsoIn, ", ")))
	}

	if evt.DateText != "" {
		niceDate := event.FormatDateNice(evt.DateText)
		msg.WriteString(fmt.Sprintf("📅 %s\n", niceDate))
	}

	if evt.City != "" {
		msg.WriteString(fmt.Sprintf("🏢 %s\n", evt.City))
	}
}
