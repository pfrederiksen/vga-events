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

## Twitter Notifications

Automatically post new VGA events to Twitter using the companion `vga-events-twitter` tool.

### Setup

1. **Get Twitter API credentials:**
   - Create a Twitter Developer account at https://developer.twitter.com
   - Create an app and get API keys
   - You need: API Key, API Secret, Access Token, Access Secret

2. **Set environment variables:**
   ```bash
   export TWITTER_API_KEY=your_api_key
   export TWITTER_API_SECRET=your_api_secret
   export TWITTER_ACCESS_TOKEN=your_access_token
   export TWITTER_ACCESS_SECRET=your_access_secret
   ```

3. **Test with dry-run:**
   ```bash
   vga-events --check-state all --format json | vga-events-twitter --dry-run
   ```

4. **Post to Twitter:**
   ```bash
   vga-events --check-state all --format json | vga-events-twitter
   ```

### Options

- `--events-file <path>` - Read from file instead of stdin
- `--dry-run` - Print tweets without posting
- `--max-tweets <n>` - Limit number of tweets (default: 10)
- `--state <STATE>` - Only tweet events for specific state

### Automated Posting (GitHub Actions)

This repository includes a GitHub Actions workflow that:
- Runs daily at 8 AM UTC
- Checks for new events
- Posts to Twitter automatically
- Can be manually triggered

To enable:
1. Fork the repository
2. Add Twitter credentials as repository secrets:
   - `TWITTER_API_KEY`
   - `TWITTER_API_SECRET`
   - `TWITTER_ACCESS_TOKEN`
   - `TWITTER_ACCESS_SECRET`
3. Enable GitHub Actions in your fork

The workflow is defined in `.github/workflows/twitter-bot.yml`

## How It Works

1. Fetches the public state events page from vgagolf.org
2. Parses event listings (state code, course, date, city)
3. Generates deterministic IDs for each event
4. Compares with previous snapshot
5. Reports new events and saves updated snapshot
6. Optionally posts to Twitter via `vga-events-twitter`

## Development

See [CLAUDE.md](CLAUDE.md) for development workflow.

## License

MIT License - see [LICENSE](LICENSE)
