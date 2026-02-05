package telegram

import (
	"fmt"
	"strings"

	"github.com/pfrederiksen/vga-events/internal/event"
)

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
