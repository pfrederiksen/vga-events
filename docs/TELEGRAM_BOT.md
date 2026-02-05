# Telegram Bot Setup & Usage

## Local Development

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

## GitHub Gist Setup

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

## Bot Commands

Users send these to the bot:

### Essential

- `/start` - Start the bot
- `/menu` - Quick actions menu
- `/help` - Show help message
- `/help <command>` - Detailed help for a command
- `/subscribe <STATE>` - Subscribe to a state (e.g., `/subscribe NV`)
- `/unsubscribe <STATE>` - Unsubscribe from a state (e.g., `/unsubscribe CA`)
- `/unsubscribe all` - Unsubscribe from all states
- `/manage` - Manage subscriptions with buttons
- `/list` - Show subscriptions
- `/check` - Check for new events now

### Event Discovery

- `/events` - View all events
- `/my-events` - View tracked events
- `/search <keyword>` - Search events
- `/near <city>` - Find events near a city
- `/export-calendar` - Download .ics calendar file

### Event Tracking

- `/note <event_id> <text>` - Add note to event
- `/note <event_id> clear` - Remove note
- `/notes` - List events with notes
- Use status buttons: ⭐ Interested, ✅ Registered, 🤔 Maybe, ❌ Skip

### Event Filtering

- `/filter` - Show filter menu
- `/filter date <range>` - Filter by date
- `/filter course <name>` - Filter by course
- `/filter city <name>` - Filter by city
- `/filter weekends` - Weekend events only
- `/filter save <name>` - Save filter preset
- `/filter load <name>` - Load filter preset
- `/filter clear` - Clear filters
- `/filters` - List saved filters

### Bulk Operations

- `/bulk` - Bulk operations menu
- `/bulk register <ids>` - Mark multiple as registered
- `/bulk note <ids> <text>` - Add note to multiple
- `/bulk status <status> <ids>` - Set status for multiple

### Notifications

- `/settings` - Configure notification mode (immediate/daily/weekly)
- `/reminders` - Configure event reminders
- `/notify-removals on|off` - Toggle removal notifications

### Statistics & Social

- `/stats` - View activity statistics
- `/invite` - Generate invite code
- `/join <code>` - Join using invite code
- `/friends` - View friends list

## Testing Workflows

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
