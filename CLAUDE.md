# Development Workflow

This document describes the development workflow for contributors working with Claude Code.

## Branch Protection

The `main` branch is protected. All changes must go through pull requests.

## Development Process

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make changes and test**
   ```bash
   go test ./...
   go build
   ```

3. **Lint your code**
   ```bash
   golangci-lint run
   ```

4. **Commit with descriptive messages**
   ```bash
   git add .
   git commit -m "Add: descriptive message about your changes"
   ```

5. **Push and create PR**
   ```bash
   git push -u origin feature/your-feature-name
   gh pr create --title "Your PR title" --body "Description of changes"
   ```

6. **Wait for CI checks** - GitHub Actions will run linting and tests

7. **Merge after approval** - Once CI passes and PR is approved

## Testing

- Add tests for new features in `*_test.go` files
- Use `testdata/fixtures/` for test HTML fixtures
- Run tests: `go test -v ./...`
- Check coverage: `go test -cover ./...`
- **Current test coverage: 82.3%** across core modules
  - internal/calendar: 98.8%
  - internal/course: 96.0%
  - internal/event: 89.0%
  - internal/logger: 89.2%
  - internal/crypto: 86.4%
  - internal/filter: 85.6%
  - internal/scraper: 85.6%

## Code Style

- Follow standard Go conventions
- Use `gofmt` (automatic with most editors)
- Keep functions focused and testable
- Add comments for exported functions

## Debugging

- Use `--verbose` flag to see debug output
- Check `~/.local/share/vga-events/` for snapshot files
- Use `--refresh` to reset state for testing

### Structured Logging

The bot uses the `internal/logger` package for structured JSON logging:

```go
import "github.com/pfrederiksen/vga-events/internal/logger"

// Log messages with structured fields
logger.Info("Processing command", logger.Fields{
    "command": "/subscribe",
    "user_id": chatID,
})

logger.Error("Failed to fetch events", logger.Fields{
    "state": "NV",
    "retry_count": 3,
}, err)

// Track metrics
logger.IncrCounter("commands.subscribe")
logger.SetGauge("active_users", 42.0)
logger.RecordTiming("api.fetch", duration)

// Get metrics snapshot
snapshot := logger.GetMetricsSnapshot()
```

### Event Filtering System

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

### Bulk Operations

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

## Telegram Bot (Interactive + Multi-User)

The project includes an interactive Telegram bot with personalized notifications.

### Architecture

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

### Components

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

### Security Features

**Rate Limiting:**
- Per-user rate limiting: 10 commands/minute using sliding window algorithm
- Prevents command spam and DoS attempts
- Automatic cleanup to prevent memory growth

**Data Protection:**
- Local snapshot files use 0600 permissions (owner-only access)
- Optional AES-256-GCM encryption for sensitive Gist data
- PBKDF2 key derivation with 100,000 iterations
- Encrypts: event notes, event statuses, invite codes
- Backward compatible with unencrypted data

**Input Validation:**
- Length limits: notes (500 chars), city names (100 chars), search keywords (100 chars)
- Control character sanitization
- Validation before processing

**Error Handling:**
- Sanitized error messages (no sensitive API responses exposed)
- Proper error wrapping for debugging

### Current Version: v0.7.0

**v0.7.0 (Latest):**
- **Advanced Event Filtering** - Create custom filters for precise event discovery
  - Filter by date ranges (e.g., "Mar 1-15", "March", "Apr 1 - May 15")
  - Filter by course names (substring match, case-insensitive)
  - Filter by cities (substring match, case-insensitive)
  - Filter by states (within your subscriptions)
  - Weekends-only filtering (Saturday/Sunday events)
  - Save and load filter presets with names
  - Commands: `/filter`, `/filter date/course/city/state/weekends`, `/filter save/load/clear`, `/filters`
- **Event Deduplication** - Events appearing in multiple states show "Also in:" notation
  - Reduces notification spam for cross-state events
  - Appears in notifications, `/events`, `/my-events`, and CLI output
  - Uses normalized title matching to detect duplicates
- **Bulk Operations** - Manage multiple events at once
  - `/bulk register <id1> <id2>` - Mark multiple events as registered
  - `/bulk note <ids> <text>` - Add same note to multiple events
  - `/bulk status <status> <ids>` - Set status for multiple events at once
  - Interactive keyboard menu for easier bulk actions
  - Supports space-separated and comma-separated event IDs
- **Enhanced Help System** - Contextual help for every command
  - `/help <command>` - Get detailed help with examples for any command
  - Shows description, usage, examples, and related commands
  - Comprehensive documentation for all 40+ bot commands
- **Structured Logging and Metrics** - Production-ready observability
  - JSON-formatted structured logging with levels (DEBUG, INFO, WARN, ERROR)
  - Operation metrics tracking (counters, gauges, timings)
  - Sanitized error messages that never expose sensitive data
  - New package: `internal/logger` with `Logger` and `Metrics` types
- **Improved Test Coverage** - 82.3% overall test coverage
  - New tests for filter parsing, bulk operations, and logging
  - Integration tests for filtering system
  - 98.8% coverage in calendar package
  - 96.0% coverage in course package

**New Packages and Files (v0.7.0):**
- `internal/filter/filter.go` - Filter types and matching logic
- `internal/filter/parser.go` - Date range parsing utilities
- `internal/filter/filter_test.go` - Filter unit tests
- `internal/filter/parser_test.go` - Parser unit tests
- `internal/filter/integration_test.go` - End-to-end filter tests
- `internal/logger/logger.go` - Structured logging and metrics
- `internal/logger/logger_test.go` - Logger tests
- `cmd/vga-events-bot/bulk_helpers.go` - Bulk operation utilities
- `cmd/vga-events-bot/bulk_helpers_test.go` - Bulk operation tests

**Data Model Changes (v0.7.0):**
- `UserPreferences.Filters` - Map of filter preset name → FilterPreset
- `UserPreferences.ActiveFilter` - Currently active filter (nil if none)
- `Event.AlsoIn` - Array of additional state codes where event appears
- `Filter` struct - Filtering criteria (dates, courses, cities, states, weekends, price)
- `FilterPreset` struct - Named filter with creation/update timestamps

**v0.6.0:**
- **Event Removal Notifications** - Get notified when events are removed or cancelled from the VGA website
  - High-priority notifications (⚠️) for events you're registered for or tracking
  - Low-priority notifications (ℹ️) for events in your subscribed states
  - Toggle with `/notify-removals on/off` command
  - Includes your personal notes if you had any for the event
  - Removed events stored for 30 days in snapshot
- **Removal Detection Infrastructure:**
  - Added `RemovedEvents` field to DiffResult
  - CompareSnapshots now detects removed events via StableKey tracking
  - Snapshot stores removed events separately with 30-day cleanup
  - "removed" ChangeType added to EventChange

**v0.5.2:**
- `/check` command for manual event checking - Instantly check for new events without waiting for hourly cycle

**v0.5.1:**
- Fixed Golf Course API integration - API key now correctly passed to workflows
- Deduplicated tee listings - No more duplicate male/female tees with same names

**v0.5.0:**
- **Golf Course API Integration** - Enriches event notifications with detailed course information
  - Shows ALL unique tee options with par, yardage, slope, and rating
  - 30-day caching reduces API usage (300 requests/day limit)
  - Appears in new event notifications AND `/my-events` command
  - Uses golfcourseapi.com (~30,000 courses worldwide)
- **New Commands:**
  - `/note <event_id> <text>` - Add personal notes to events
  - `/note <event_id> clear` - Remove notes
  - `/notes` - List all events with notes
  - `/near <city>` - Find events near a specific city (e.g., `/near Las Vegas`)
  - `/unsubscribe all` - Unsubscribe from all states with confirmation
  - `/notify-removals on/off` - Toggle removal notifications
- **Event Change Detection** - Events tracked with StableKey (SHA1 of state + normalized title)
- **Change Notifications** - Detects when event dates/titles/cities change (infrastructure complete)
- **Event Notes** - Personal notes stored in EventNotes map in UserPreferences

**Data Model Changes:**
- `Event.StableKey` - New field for tracking events across detail changes
- `UserPreferences.EventNotes` - Map of event ID → note text
- `UserPreferences.NotifyOnChanges` - Flag for change notifications (default: true)
- `UserPreferences.NotifyOnRemoval` - Flag for removal notifications (default: true)
- `Snapshot.StableIndex` - Map of StableKey → Event ID
- `Snapshot.ChangeLog` - Array of recent EventChange objects
- `Snapshot.RemovedEvents` - Map of recently removed events (kept for 30 days)
- `Snapshot.CourseCache` - 30-day cache for golf course information
- `DiffResult.RemovedEvents` - Array of events removed since last check
- `EventChange.ChangeType` - Now includes "removed" type

### Local Development

**Setup:**
```bash
# Build all tools
make build

# Create Gist for testing (using gh CLI)
echo '{}' | gh gist create --filename "vga-events-preferences.json" --desc "VGA Events Bot Preferences" -

# Alternative: use the helper script
# ./scripts/create-gist.sh YOUR_GITHUB_TOKEN

# Set environment variables
export TELEGRAM_BOT_TOKEN=your_bot_token
export TELEGRAM_GIST_ID=your_gist_id
export TELEGRAM_GITHUB_TOKEN=your_github_token
export GOLF_COURSE_API_KEY=your_golf_api_key        # Optional: enables course info
export TELEGRAM_ENCRYPTION_KEY=your_encryption_key  # Optional: enables data encryption
```

**Test command processing:**
```bash
# Dry run (shows what would happen)
./vga-events-bot --dry-run

# Real processing
./vga-events-bot
```

**Test notifications:**
```bash
# Send to specific user
./vga-events --check-state all --format json | \
  ./vga-events-telegram --chat-id YOUR_CHAT_ID

# Test with dry-run
./vga-events --check-state NV --format json | \
  ./vga-events-telegram --chat-id YOUR_CHAT_ID --dry-run
```

**Test with Telegram:**
1. Send `/subscribe NV` to your bot
2. Wait for command processor to run (or run `./vga-events-bot` manually)
3. Check that Gist updated with your subscription
4. Run notification workflow to test personalized filtering

### GitHub Gist Setup

**Create Gist:**

Using GitHub CLI (recommended):
```bash
echo '{}' | gh gist create --filename "vga-events-preferences.json" --desc "VGA Events Bot Preferences" -
```

Or using the helper script:
```bash
# Get GitHub token with 'gist' scope from https://github.com/settings/tokens
./scripts/create-gist.sh ghp_yourTokenHere
```

Both methods create a private Gist containing:
```json
{}
```

As users subscribe, it will look like:
```json
{
  "1745308556": {
    "states": ["NV", "CA"],
    "active": true
  },
  "9876543210": {
    "states": ["TX"],
    "active": true
  }
}
```

**Required GitHub Secrets:**
- `TELEGRAM_BOT_TOKEN` - From @BotFather
- `TELEGRAM_GIST_ID` - From create-gist.sh output
- `TELEGRAM_GITHUB_TOKEN` - GitHub token with 'gist' scope
- `GOLF_COURSE_API_KEY` - From golfcourseapi.com (optional, enables course info)
- `TELEGRAM_ENCRYPTION_KEY` - Strong passphrase for data encryption (optional but recommended, enables AES-256 encryption)

### Bot Commands

Users send these to the bot:
- `/subscribe NV` - Subscribe to Nevada
- `/unsubscribe CA` - Unsubscribe from California
- `/list` - Show subscriptions
- `/help` - Show help message

### How Personalization Works

1. User subscribes via `/subscribe NV`
2. Command processor (runs every 15min) updates Gist
3. Event checker (runs hourly) loads preferences from Gist
4. For each user: filter events by their subscribed states
5. Send only matching events to each user

### Testing Workflows

**Command processor:**
```bash
gh workflow run telegram-bot-commands.yml
gh run list --workflow=telegram-bot-commands.yml
```

**Notifications:**
```bash
gh workflow run telegram-bot.yml
gh run list --workflow=telegram-bot.yml
```

**View logs:**
```bash
gh run view <run-id> --log
```

### Debugging

**Check Gist contents:**
```bash
curl -H "Authorization: token $TELEGRAM_GITHUB_TOKEN" \
  https://api.github.com/gists/$TELEGRAM_GIST_ID | \
  jq -r '.files["preferences.json"].content'
```

**Test preference filtering:**
```bash
# Simulate filtering for a user subscribed to NV, CA
jq --arg states "NV,CA" '
  .new_events as $events |
  ($states | split(",")) as $subscribed |
  {
    new_events: ($events | map(select(.state as $s | $subscribed | index($s)))),
    event_count: ($events | map(select(.state as $s | $subscribed | index($s))) | length)
  }
' events.json
```

## Releases

Releases are automated using GoReleaser and published to the Homebrew tap.

### Creating a Release

1. **Merge all changes to main**
   ```bash
   # Ensure your PR is merged to main
   git checkout main
   git pull origin main
   ```

2. **Create and push a version tag**
   ```bash
   # Use semantic versioning (e.g., v0.1.0, v1.0.0)
   git tag v0.1.0
   git push origin v0.1.0
   ```

3. **GitHub Actions automatically**:
   - Builds binaries for Linux, macOS, Windows (amd64 and arm64)
   - Creates a GitHub release with archives and checksums
   - Updates the Homebrew formula in `pfrederiksen/homebrew-tap`

### Local Release Testing

Test the release process locally (without publishing):

```bash
goreleaser release --snapshot --clean
```

This creates builds in `./dist/` without pushing anything.

### Homebrew Tap Setup

The `.goreleaser.yml` configuration automatically:
- Publishes to `github.com/pfrederiksen/homebrew-tap`
- Uses `HOMEBREW_TAP_GITHUB_TOKEN` secret for authentication
- Updates the formula in `Formula/vga-events.rb`

Users can install via:
```bash
brew install pfrederiksen/tap/vga-events
```

### Version Information

Version info is injected at build time via ldflags:
- `version` - Git tag (e.g., "v0.1.0")
- `commit` - Git commit SHA
- `date` - Build timestamp

View with: `vga-events --version`
