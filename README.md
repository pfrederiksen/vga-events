# vga-events

A simple, reliable CLI tool to check for newly-added VGA Golf state events without logging in.

## Features

- Check for new events by state (e.g., `NV`) or all states
- Tracks events across runs using local snapshots
- Reports only new events since last check
- Extracts event dates from the website
- Sort events by date, state, or title
- View all tracked events with `--show-all`
- JSON or text output formats
- Exit codes for easy scripting

## Installation

### Homebrew (macOS/Linux)

```bash
brew install pfrederiksen/tap/vga-events
```

### Go Install

```bash
go install github.com/pfrederiksen/vga-events@latest
```

### Build from Source

```bash
git clone https://github.com/pfrederiksen/vga-events.git
cd vga-events
go build
```

## Usage

Check for new Nevada events:
```bash
vga-events --check-state NV
```

Check all states:
```bash
vga-events --check-state all
```

Get JSON output:
```bash
vga-events --check-state all --format json
```

Refresh/reset snapshot:
```bash
vga-events --check-state all --refresh
```

Show all tracked events (not just new ones):
```bash
vga-events --check-state NV --show-all
vga-events --check-state all --show-all --format json
```

Sort events by title:
```bash
vga-events --check-state all --sort title
```

Check version:
```bash
vga-events --version
```

### Flags

- `--check-state <STATE|all>` - Required. Check specific state or all states
- `--format <text|json>` - Output format (default: text)
- `--sort <date|state|title>` - Sort order (default: date)
- `--data-dir <path>` - Data directory (default: ~/.local/share/vga-events)
- `--refresh` - Recreate snapshot without showing new events
- `--show-all` - Show all tracked events, not just new ones
- `--verbose` - Show debug logs
- `--version, -v` - Show version information

### Exit Codes

- `0` - No new events (or --refresh/--show-all mode)
- `2` - New events found
- `1` - Error occurred

## Cron Usage

Check for Nevada events daily at 8 AM:
```cron
0 8 * * * /usr/local/bin/vga-events --check-state NV && echo "New NV events!" | mail -s "VGA Events" you@example.com
```

Check all states and get notified if new events exist:
```bash
#!/bin/bash
if vga-events --check-state all; then
    # Exit code 2 means new events
    if [ $? -eq 2 ]; then
        # Send notification
        vga-events --check-state all | mail -s "New VGA Events" you@example.com
    fi
fi
```

## Telegram Bot (Interactive + Personalized Notifications)

Get personalized VGA event notifications via Telegram! The bot supports multiple users, each with their own state subscriptions.

### Quick Start

1. **Create your bot:**
   - Message @BotFather on Telegram
   - Send `/newbot` and follow instructions
   - Save your bot token

2. **Start chatting with your bot:**
   - Click the link BotFather provides
   - Send `/help` to see available commands
   - Use `/subscribe NV` to subscribe to Nevada events
   - Use `/list` to see your subscriptions

### Bot Commands

Send these commands to your bot in Telegram:

**Essential Commands:**
- `/menu` - Quick actions menu with buttons
- `/help` - Show help message with all commands
- `/subscribe <STATE>` - Subscribe to a state's events (e.g., `/subscribe NV`)
- `/manage` - Manage your subscriptions with buttons
- `/list` - Show your current subscriptions

**Event Discovery:**
- `/search <keyword>` - Search for events (e.g., `/search "Pine Valley"`)
- `/my-events` - View events you've marked as interested/registered
- `/export-calendar` - Download all events as .ics calendar file

**Event Tracking:**
When you receive event notifications, use the status buttons to track them:
- ⭐ **Interested** - Events you want to attend
- ✅ **Registered** - Events you've signed up for
- 🤔 **Maybe** - Events you're considering
- ❌ **Skip** - Events you're not interested in

**Reminders:**
- `/reminders` - Configure event reminders (1 day, 3 days, 1 week, or 2 weeks before)
- Get reminded about events you've marked as ⭐ Interested or ✅ Registered

**Notification Settings:**
- `/settings` - Configure notification mode:
  - Immediate (default) - Get notified right away
  - Daily digest - Receive a daily summary at 9 AM UTC
  - Weekly digest - Receive a weekly summary on Mondays

**Multi-User Support:** Each person gets their own subscriptions, event tracking, and reminder preferences!

### GitHub Actions Setup (Automated Notifications)

The bot runs on GitHub Actions - no server needed! It:
- Checks for commands every 15 minutes
- Checks for new events every hour
- Sends personalized notifications to each user based on their subscriptions

**Required Secrets:**

1. **Create a GitHub Gist** to store user preferences:

   **Option A: Using GitHub CLI (recommended)**
   ```bash
   echo '{}' | gh gist create --filename "vga-events-preferences.json" --desc "VGA Events Bot Preferences" -
   ```

   **Option B: Using the helper script**
   ```bash
   # Get a GitHub token with 'gist' scope from https://github.com/settings/tokens
   ./scripts/create-gist.sh YOUR_GITHUB_TOKEN
   ```

   Both methods will output a Gist ID.

2. **Add repository secrets** (Settings → Secrets and variables → Actions):
   - `TELEGRAM_BOT_TOKEN` - Your bot token from @BotFather
   - `TELEGRAM_GIST_ID` - The Gist ID from step 1
   - `TELEGRAM_GITHUB_TOKEN` - GitHub token with 'gist' scope

3. The workflows will start running automatically:
   - Commands processed every 15 minutes
   - Notifications sent hourly

### Local Testing

For development or testing locally:

```bash
# Set environment variables
export TELEGRAM_BOT_TOKEN=your_bot_token
export TELEGRAM_GIST_ID=your_gist_id
export TELEGRAM_GITHUB_TOKEN=your_github_token

# Process bot commands manually
./vga-events-bot

# Send notifications manually
./vga-events --check-state all --format json | ./vga-events-telegram --chat-id YOUR_CHAT_ID
```

### How It Works

1. **User subscribes** via `/subscribe NV` command
2. **Bot processes command** (runs every 15 minutes via GitHub Actions)
3. **Preferences stored** in private GitHub Gist
4. **Event checking** runs hourly via GitHub Actions
5. **Personalized notifications** sent only for subscribed states
6. **Each user** receives only their relevant events

## How It Works

1. Fetches the public state events page from vgagolf.org
2. Parses event listings (state code, course, date, city)
3. Generates deterministic IDs for each event
4. Compares with previous snapshot
5. Reports new events and saves updated snapshot

## Development

See [CLAUDE.md](CLAUDE.md) for development workflow.

## License

MIT License - see [LICENSE](LICENSE)
