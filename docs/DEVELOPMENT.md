# Development Guide

This document describes the development workflow for contributors.

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
- **Current test coverage: 82.3%** across core modules
  - internal/calendar: 98.8%
  - internal/course: 96.0%
  - internal/event: 89.0%
  - internal/logger: 89.2%
  - internal/crypto: 86.4%
  - internal/filter: 85.6%
  - internal/scraper: 85.6%

## Code Style

- Follow standard Go conventions
- Use `gofmt` (automatic with most editors)
- Keep functions focused and testable
- Add comments for exported functions

## Quick Debug Commands

- Use `--verbose` flag to see debug output
- Check `~/.local/share/vga-events/` for snapshot files
- Use `--refresh` to reset state for testing

See [DEBUGGING.md](DEBUGGING.md) for detailed debugging techniques.
