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

## Telegram Bot

The project includes a Telegram notification system that automatically sends new events.

### Components

- **vga-events-telegram** - CLI tool that reads event JSON and sends to Telegram
- **internal/telegram** - Package with Telegram client and formatting logic
- **.github/workflows/telegram-bot.yml** - GitHub Actions workflow for automation

### Local Development

Test the Telegram notifier locally:

```bash
# Build both tools
make build

# Set Telegram credentials
export TELEGRAM_BOT_TOKEN=your_bot_token
export TELEGRAM_CHAT_ID=your_chat_id

# Test with dry-run (no actual sending)
./vga-events --check-state all --format json | ./vga-events-telegram --dry-run

# Send real notifications
./vga-events --check-state all --format json | ./vga-events-telegram

# Test with a single state
./vga-events --check-state NV --format json | ./vga-events-telegram --dry-run
```

### Getting Telegram Credentials

1. **Bot Token:**
   - Message @BotFather on Telegram
   - Send `/newbot` command
   - Follow instructions to create your bot
   - Copy the bot token

2. **Chat ID:**
   - Start a conversation with your bot
   - Send any message
   - Visit: `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
   - Find `"chat":{"id":123456789}` in the response

### GitHub Actions Setup

The automated workflow requires these repository secrets:
- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_CHAT_ID`

Add secrets via: Repository Settings → Secrets and variables → Actions → New repository secret

### Testing the Workflow

**Manual trigger:**
```bash
# Via GitHub UI: Actions → Telegram Event Notifications → Run workflow

# Via CLI
gh workflow run telegram-bot.yml
```

**View workflow runs:**
```bash
gh run list --workflow=telegram-bot.yml
gh run view <run-id>
```

### Message Format

Messages use HTML formatting for better readability:
- Bold headers and state codes
- Emoji indicators (🏌️ 📍 📅 🏢 🔗)
- Clickable links
- State-specific hashtags
- Up to 4096 characters (Telegram limit)

### Advantages Over Email/Twitter

1. **Native Mobile Notifications** - Instant push to phone
2. **Simple Auth** - Just bot token + chat ID
3. **No Rate Limits** - For personal use
4. **Free** - No API costs
5. **Rich Formatting** - HTML support, emoji, links
6. **Reliable** - No spam filters

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
