# Debugging Guide

## Quick Debug Commands

- Use `--verbose` flag to see debug output
- Check `~/.local/share/vga-events/` for snapshot files
- Use `--refresh` to reset state for testing

## Structured Logging

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

**Log Levels:**
- DEBUG - Detailed debugging information
- INFO - General informational messages
- WARN - Warning messages for potential issues
- ERROR - Error conditions

## Debugging Workflows

### Check Gist contents

```bash
curl -H "Authorization: token $TELEGRAM_GITHUB_TOKEN" \
  https://api.github.com/gists/$TELEGRAM_GIST_ID | \
  jq -r '.files["preferences.json"].content'
```

### Test preference filtering

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

### Debug command processing

```bash
# Run bot in dry-run mode to see what would happen
./vga-events-bot --dry-run

# Run with verbose logging
./vga-events-bot --verbose
```

### Debug notifications

```bash
# Test notification formatting without sending
./vga-events --check-state NV --format json | jq

# Send test notification to yourself
./vga-events --check-state NV --format json | \
  ./vga-events-telegram --chat-id YOUR_CHAT_ID --dry-run
```

## Common Issues

### Bot not responding
1. Check `TELEGRAM_BOT_TOKEN` is set correctly
2. Verify bot is not blocked by user
3. Check rate limiting logs

### Events not showing up
1. Verify state subscription: `/list`
2. Check filters: `/filter` (use `/filter clear` to reset)
3. Check event date is not in the past (use `/settings` to adjust)

### Gist not updating
1. Verify `TELEGRAM_GIST_ID` and `TELEGRAM_GITHUB_TOKEN` are set
2. Check token has 'gist' scope
3. Verify Gist is accessible (not deleted)

### Encryption issues
1. Ensure `TELEGRAM_ENCRYPTION_KEY` is consistent across runs
2. Check for encryption/decryption errors in logs
3. Verify key is properly base64-encoded if using raw keys
