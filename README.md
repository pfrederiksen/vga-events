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

## Telegram Notifications

Automatically receive new VGA events via Telegram using the companion `vga-events-telegram` tool.

### Setup

1. **Create a Telegram bot:**
   - Message @BotFather on Telegram
   - Send `/newbot` and follow instructions
   - Save your bot token (looks like `1234567890:ABCdefGHIjklMNOpqrsTUVwxyz`)

2. **Get your Chat ID:**
   - Start a conversation with your bot
   - Send any message to it
   - Visit `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
   - Find your chat ID in the response (looks like `123456789`)

3. **Set environment variables:**
   ```bash
   export TELEGRAM_BOT_TOKEN=your_bot_token_here
   export TELEGRAM_CHAT_ID=your_chat_id_here
   ```

4. **Test with dry-run:**
   ```bash
   vga-events --check-state all --format json | vga-events-telegram --dry-run
   ```

5. **Send real notifications:**
   ```bash
   vga-events --check-state all --format json | vga-events-telegram
   ```

### Command Options

- `--bot-token <token>` - Telegram bot token (or env: TELEGRAM_BOT_TOKEN)
- `--chat-id <id>` - Telegram chat ID (or env: TELEGRAM_CHAT_ID)
- `--events-file <path>` - Read from file instead of stdin
- `--dry-run` - Print messages without sending
- `--max-messages <n>` - Limit number of messages (default: 10)
- `--state <STATE>` - Only send messages for specific state

### Automated Notifications (GitHub Actions)

This repository includes a GitHub Actions workflow that:
- Runs every hour (at the top of the hour)
- Checks all states for new events
- Sends Telegram notifications automatically (max 10 per run)
- Can be manually triggered

To enable:
1. Add repository secrets (Settings → Secrets and variables → Actions):
   - `TELEGRAM_BOT_TOKEN`
   - `TELEGRAM_CHAT_ID`
2. The workflow will automatically run every hour

The workflow is defined in `.github/workflows/telegram-bot.yml`

### Interactive Usage

You can also run checks manually anytime:
```bash
# Check Nevada events and notify
vga-events --check-state NV --format json | vga-events-telegram

# Check all states
vga-events --check-state all --format json | vga-events-telegram

# Dry run to see what would be sent
vga-events --check-state all --format json | vga-events-telegram --dry-run
```

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
