# Release Process

Releases are automated using GoReleaser and published to the Homebrew tap.

## Creating a Release

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

## Local Release Testing

Test the release process locally (without publishing):

```bash
goreleaser release --snapshot --clean
```

This creates builds in `./dist/` without pushing anything.

## Homebrew Tap Setup

The `.goreleaser.yml` configuration automatically:
- Publishes to `github.com/pfrederiksen/homebrew-tap`
- Uses `HOMEBREW_TAP_GITHUB_TOKEN` secret for authentication
- Updates the formula in `Formula/vga-events.rb`

Users can install via:
```bash
brew install pfrederiksen/tap/vga-events
```

## Version Information

Version info is injected at build time via ldflags:
- `version` - Git tag (e.g., "v0.1.0")
- `commit` - Git commit SHA
- `date` - Build timestamp

View with: `vga-events --version`

## Release Checklist

- [ ] All tests pass (`go test ./...`)
- [ ] Linting passes (`golangci-lint run`)
- [ ] CHANGELOG.md updated
- [ ] VERSION_HISTORY.md updated
- [ ] Version tag follows semantic versioning
- [ ] Tag pushed to remote
- [ ] GitHub Actions workflow succeeds
- [ ] Homebrew tap updated automatically
- [ ] Test installation via Homebrew
