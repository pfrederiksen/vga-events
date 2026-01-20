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

## Twitter Bot

The project includes a Twitter notification system that automatically posts new events.

### Components

- **vga-events-twitter** - CLI tool that reads event JSON and posts to Twitter
- **internal/notifier** - Package with Twitter client and formatting logic
- **.github/workflows/twitter-bot.yml** - GitHub Actions workflow for automation

### Local Development

Test the Twitter notifier locally:

```bash
# Build both tools
make build
go build -o vga-events-twitter ./cmd/vga-events-twitter

# Set Twitter credentials
export TWITTER_API_KEY=xxx
export TWITTER_API_SECRET=xxx
export TWITTER_ACCESS_TOKEN=xxx
export TWITTER_ACCESS_SECRET=xxx

# Test with dry-run (no actual posting)
./vga-events --check-state NV --format json | ./vga-events-twitter --dry-run

# Post to Twitter
./vga-events --check-state all --format json | ./vga-events-twitter
```

### GitHub Actions Setup

The automated workflow requires these repository secrets:
- `TWITTER_API_KEY`
- `TWITTER_API_SECRET`
- `TWITTER_ACCESS_TOKEN`
- `TWITTER_ACCESS_SECRET`

Add secrets via: Repository Settings → Secrets and variables → Actions → New repository secret

### Testing the Workflow

**Manual trigger:**
```bash
# Via GitHub UI: Actions → Twitter Event Notifications → Run workflow

# Via CLI
gh workflow run twitter-bot.yml
```

**View workflow runs:**
```bash
gh run list --workflow=twitter-bot.yml
gh run view <run-id>
```

### Snapshot Storage

The GitHub Actions workflow uses GitHub Actions cache to persist snapshots between runs:
- Cache key: `vga-events-snapshots-*`
- Location: `.snapshots/` directory
- Restored before each run, saved after completion

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
