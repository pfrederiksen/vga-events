# Version History

## Current Version: v0.8.5

### v0.8.5 (Latest)

**Gist Storage Reliability**
- Added revision-aware conditional writes so concurrent preference changes fail safely instead of being silently overwritten.
- Long-running bot sessions now refresh preferences immediately before applying command batches.
- Added support for GitHub's truncated Gist responses, a 10 MiB safety limit, and trusted-host validation for raw content downloads.
- Updated GitHub API authentication and version headers and expanded HTTP-level and race-test coverage.

**Operational Safety**
- Empty event scrapes now signal a likely upstream format change instead of resembling removal of every event.
- Notification checks fail visibly after retries are exhausted so health monitoring can detect outages.
- Encryption keys are passed consistently to preference-writing bot workflows.
- Corrected the documented cron exit-code handling and removed a generated test binary from source control.

### v0.8.4

**Nevada Event Coverage**
- Fixed parsing for state championship entries like `2026 NV State Championship` that omit the usual `NV -` event prefix on the VGA public state-events page.

**Go Toolchain Update**
- Updated the project and GitHub Actions workflows to use Go `1.26.4`.

### v0.8.3

**Go Toolchain Update**
- Updated the project and GitHub Actions workflows to use Go `1.26.2`.

**Automation Reliability**
- Dependency and release automation now support a dedicated `AUTOMATION_PR_TOKEN` fallback and degrade cleanly when Actions is not allowed to create pull requests.

**Refactoring and Maintainability**
- Reduced complexity in the Telegram bot command and callback dispatchers by extracting dedicated handler helpers.
- Split `UserPreferences` into embedded focused preference groups while preserving the existing API and JSON layout.
- Consolidated repeated GitHub API request setup, scraper event parsing branches, CLI run-check orchestration, and Telegram HTTP test scaffolding.
- Simplified repeated `/filter` command argument parsing and shared event-list delivery in bot flows.

### v0.8.2

**Workflow Reliability Fixes**
- Dependency automation now limits itself to Go-compatible patch updates instead of attempting upgrades that require a newer toolchain than the project supports.
- Release-supporting automation now creates required labels before opening issues or PRs, preventing scheduled workflow failures due to missing labels.
- The User Feedback Agent workflow was rewritten into valid, parseable YAML and aligned with the labels it uses at runtime.

**CI Stability**
- Fixed a date-sensitive `internal/filter` integration test that failed once March moved into the past because the date parser intentionally rolls ambiguous month-only ranges into the next year.

### v0.7.0

**Advanced Event Filtering** - Create custom filters for precise event discovery
- Filter by date ranges (e.g., "Mar 1-15", "March", "Apr 1 - May 15")
- Filter by course names (substring match, case-insensitive)
- Filter by cities (substring match, case-insensitive)
- Filter by states (within your subscriptions)
- Weekends-only filtering (Saturday/Sunday events)
- Save and load filter presets with names
- Commands: `/filter`, `/filter date/course/city/state/weekends`, `/filter save/load/clear`, `/filters`

**Event Deduplication** - Events appearing in multiple states show "Also in:" notation
- Reduces notification spam for cross-state events
- Appears in notifications, `/events`, `/my-events`, and CLI output
- Uses normalized title matching to detect duplicates

**Bulk Operations** - Manage multiple events at once
- `/bulk register <id1> <id2>` - Mark multiple events as registered
- `/bulk note <ids> <text>` - Add same note to multiple events
- `/bulk status <status> <ids>` - Set status for multiple events at once
- Interactive keyboard menu for easier bulk actions
- Supports space-separated and comma-separated event IDs

**Enhanced Help System** - Contextual help for every command
- `/help <command>` - Get detailed help with examples for any command
- Shows description, usage, examples, and related commands
- Comprehensive documentation for all 40+ bot commands

**Structured Logging and Metrics** - Production-ready observability
- JSON-formatted structured logging with levels (DEBUG, INFO, WARN, ERROR)
- Operation metrics tracking (counters, gauges, timings)
- Sanitized error messages that never expose sensitive data
- New package: `internal/logger` with `Logger` and `Metrics` types

**Improved Test Coverage** - 82.3% overall test coverage
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

### v0.6.0

**Event Removal Notifications** - Get notified when events are removed or cancelled from the VGA website
- High-priority notifications (⚠️) for events you're registered for or tracking
- Low-priority notifications (ℹ️) for events in your subscribed states
- Toggle with `/notify-removals on/off` command
- Includes your personal notes if you had any for the event
- Removed events stored for 30 days in snapshot

**Removal Detection Infrastructure:**
- Added `RemovedEvents` field to DiffResult
- CompareSnapshots now detects removed events via StableKey tracking
- Snapshot stores removed events separately with 30-day cleanup
- "removed" ChangeType added to EventChange

### v0.5.2

- `/check` command for manual event checking - Instantly check for new events without waiting for hourly cycle

### v0.5.1

- Fixed Golf Course API integration - API key now correctly passed to workflows
- Deduplicated tee listings - No more duplicate male/female tees with same names

### v0.5.0

**Golf Course API Integration** - Enriches event notifications with detailed course information
- Shows ALL unique tee options with par, yardage, slope, and rating
- 30-day caching reduces API usage (300 requests/day limit)
- Appears in new event notifications AND `/my-events` command
- Uses golfcourseapi.com (~30,000 courses worldwide)

**New Commands:**
- `/note <event_id> <text>` - Add personal notes to events
- `/note <event_id> clear` - Remove notes
- `/notes` - List all events with notes
- `/near <city>` - Find events near a specific city (e.g., `/near Las Vegas`)
- `/unsubscribe all` - Unsubscribe from all states with confirmation
- `/notify-removals on/off` - Toggle removal notifications

**Event Change Detection** - Events tracked with StableKey (SHA1 of state + normalized title)

**Change Notifications** - Detects when event dates/titles/cities change (infrastructure complete)

**Event Notes** - Personal notes stored in EventNotes map in UserPreferences

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
