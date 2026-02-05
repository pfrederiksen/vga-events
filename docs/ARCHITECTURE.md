# System Architecture

## Overview

The project consists of three binaries working together:

**Three binaries:**
- **vga-events** - Scrapes and checks for new events
- **vga-events-telegram** - Sends notifications to Telegram
- **vga-events-bot** - Processes user commands (/subscribe, /unsubscribe, etc.)

**Seven workflows:**
- **telegram-bot-commands.yml** - Processes commands every 15 minutes
- **telegram-bot.yml** - Checks for events hourly, sends personalized notifications
- **telegram-daily-digest.yml** - Sends daily digest at 9 AM UTC for digest mode users
- **telegram-weekly-digest.yml** - Sends weekly digest on Mondays at 9 AM UTC
- **telegram-reminders.yml** - Sends event reminders daily at 9 AM UTC
- **telegram-weekly-stats.yml** - Archives weekly stats every Sunday at 11:59 PM UTC
- **ci.yml** - Runs tests and builds on PRs

**Storage:**
- User preferences stored in private GitHub Gist (JSON)
- Optional AES-256-GCM encryption for sensitive data
- Event snapshots stored in GitHub Actions cache (secure, owner-only permissions)
- Local snapshots use 0600 permissions (owner read/write only)
- No database needed!

## Components

1. **internal/telegram** - Telegram API client (send messages)
2. **internal/preferences** - User preference management + Gist storage with encryption
3. **internal/crypto** - AES-256-GCM encryption for sensitive data
4. **internal/filter** - Event filtering system with preset support
5. **internal/logger** - Structured JSON logging and metrics tracking
6. **cmd/vga-events-bot** - Command processor (handles /subscribe, /filter, /bulk, etc.) with rate limiting
7. **cmd/vga-events-bot/bulk_helpers.go** - Bulk operation utilities
8. **cmd/vga-events-telegram** - Notification sender
9. **.github/workflows/telegram-bot-commands.yml** - Command processing
10. **.github/workflows/telegram-bot.yml** - Personalized notifications
11. **.github/workflows/telegram-daily-digest.yml** - Daily digest delivery
12. **.github/workflows/telegram-weekly-digest.yml** - Weekly digest delivery
13. **.github/workflows/telegram-reminders.yml** - Event reminder delivery

## Dispatcher Architecture

The bot uses dispatcher functions to reduce complexity in the main command processor:

- `cmd/vga-events-bot/filter_dispatcher.go` - Handles all `/filter` subcommands
- `cmd/vga-events-bot/bulk_dispatcher.go` - Handles all `/bulk` subcommands
- `cmd/vga-events-bot/param_helpers.go` - Parameter extraction and validation
- `cmd/vga-events-bot/status_helpers.go` - Status emoji and text display
- `cmd/vga-events-bot/stats_helpers.go` - Statistics formatting
- `internal/telegram/formatter_helpers.go` - Event message formatting helpers

**Benefits:**
- Reduced complexity: processCommand from 66 to ~40-45
- Better testability: Each dispatcher independently testable
- Clear separation: One dispatcher per command family

## Event Filtering System

The filtering system allows users to create complex event filters:

**Filter Structure:**
- Date ranges (from/to)
- Course names (substring match, case-insensitive)
- Cities (substring match, case-insensitive)
- States (within user subscriptions)
- Weekends-only flag
- Maximum price (placeholder for future use)

**Filter Presets:**
Users can save filters with names and load them later:

```go
import "github.com/pfrederiksen/vga-events/internal/filter"

// Create a filter
f := filter.NewFilter()
f.WeekendsOnly = true
f.Courses = []string{"Pebble Beach"}

// Save as preset
preset := filter.NewFilterPreset("weekend-pebble", f)
user.Filters["weekend-pebble"] = preset

// Apply filter to events
filtered := f.Apply(events)
```

**Date Range Parsing:**
The `internal/filter/parser.go` supports multiple date formats:
- `Mar 1-15` - Same month range
- `March 1 - April 15` - Cross-month range
- `March` - Entire month
- Automatically handles year inference (current or next year)

## Bulk Operations

Bulk operation helpers in `cmd/vga-events-bot/bulk_helpers.go`:

```go
// Parse event IDs from user input (space or comma-separated)
eventIDs := parseBulkEventIDs(parts)

// Mark multiple events as registered
handleBulkRegister(prefs, chatID, eventIDs, &modified)

// Add same note to multiple events
handleBulkNote(prefs, chatID, eventIDs, "Great course!", &modified)

// Set status for multiple events
handleBulkStatus(prefs, chatID, "interested", eventIDs, &modified)
```

## How Personalization Works

1. User subscribes via `/subscribe NV`
2. Command processor (runs every 15min) updates Gist
3. Event checker (runs hourly) loads preferences from Gist
4. For each user: filter events by their subscribed states
5. Send only matching events to each user
