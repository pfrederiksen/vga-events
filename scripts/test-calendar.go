package main

import (
	"fmt"
	"os"
	"time"

	"github.com/pfrederiksen/vga-events/internal/calendar"
	"github.com/pfrederiksen/vga-events/internal/event"
)

func main() {
	// Create a sample event
	evt := &event.Event{
		ID:        "test-event-123",
		State:     "NV",
		Title:     "Spring Championship at TPC Las Vegas",
		DateText:  "Mar 15 2026",
		City:      "Las Vegas",
		SourceURL: "https://vgagolf.org/state-events",
		FirstSeen: time.Now(),
	}

	// Generate .ics file
	icsContent := calendar.GenerateICS(evt)

	// Write to file (owner read/write only for security)
	filename := "test-vga-event.ics"
	if err := os.WriteFile(filename, []byte(icsContent), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Generated calendar file: %s\n\n", filename)
	fmt.Println("Test it by:")
	fmt.Println("1. Open the .ics file with your calendar app (double-click)")
	fmt.Println("2. Or import it into Google Calendar, Apple Calendar, or Outlook")
	fmt.Println("\nFile contents preview:")
	fmt.Println("---")
	fmt.Println(icsContent)
}
