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

## Code Style

- Follow standard Go conventions
- Use `gofmt` (automatic with most editors)
- Keep functions focused and testable
- Add comments for exported functions

## Debugging

- Use `--verbose` flag to see debug output
- Check `~/.local/share/vga-events/` for snapshot files
- Use `--refresh` to reset state for testing

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
- Event snapshots stored in GitHub Actions cache
- No database needed!

### Components

1. **internal/telegram** - Telegram API client (send messages)
2. **internal/preferences** - User preference management + Gist storage
3. **cmd/vga-events-bot** - Command processor (handles /subscribe, etc.)
4. **cmd/vga-events-telegram** - Notification sender
5. **.github/workflows/telegram-bot-commands.yml** - Command processing
6. **.github/workflows/telegram-bot.yml** - Personalized notifications
7. **.github/workflows/telegram-daily-digest.yml** - Daily digest delivery
8. **.github/workflows/telegram-weekly-digest.yml** - Weekly digest delivery
9. **.github/workflows/telegram-reminders.yml** - Event reminder delivery

### v0.5.0 Features (Current)

**New Commands:**
- `/note <event_id> <text>` - Add personal notes to events
- `/note <event_id> clear` - Remove notes
- `/notes` - List all events with notes
- `/near <city>` - Find events near a specific city (e.g., `/near Las Vegas`)
- `/unsubscribe all` - Unsubscribe from all states with confirmation

**New Infrastructure:**
- **Event Change Detection** - Events tracked with StableKey (SHA1 of state + normalized title)
- **Change Notifications** - Detects when event dates/titles/cities change (infrastructure complete)
- **Event Notes** - Personal notes stored in EventNotes map in UserPreferences

**Data Model Changes:**
- `Event.StableKey` - New field for tracking events across detail changes
- `UserPreferences.EventNotes` - Map of event ID → note text
- `UserPreferences.NotifyOnChanges` - Flag for change notifications (default: true)
- `Snapshot.StableIndex` - Map of StableKey → Event ID
- `Snapshot.ChangeLog` - Array of recent EventChange objects

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
