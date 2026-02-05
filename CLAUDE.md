# VGA Events Bot - Documentation

Welcome to the VGA Events Bot project documentation. This interactive Telegram bot helps users track VGA Golf events across multiple states with personalized notifications.

## Quick Start

**For Contributors:**
1. Read [Development Guide](docs/DEVELOPMENT.md) - Workflow, testing, code style
2. Understand [System Architecture](docs/ARCHITECTURE.md) - Components and design patterns
3. Review [Security Features](docs/SECURITY.md) - Best practices

**For Bot Operators:**
1. Follow [Telegram Bot Setup](docs/TELEGRAM_BOT.md) - Local development and deployment
2. Configure [GitHub Secrets](docs/TELEGRAM_BOT.md#github-gist-setup) - Required tokens and keys
3. Review [Bot Commands](docs/TELEGRAM_BOT.md#bot-commands) - Available user commands

**For Troubleshooting:**
1. Check [Debugging Guide](docs/DEBUGGING.md) - Common issues and solutions
2. View [Version History](docs/VERSION_HISTORY.md) - Feature changelog

## Project Overview

**Current Version:** v0.7.0

**Key Features:**
- ✅ Multi-user Telegram bot with personalized notifications
- 🔍 Advanced event filtering (date, course, city, weekends)
- 📝 Event tracking with notes and status management
- 🔄 Bulk operations for managing multiple events
- 🔒 AES-256-GCM encryption for sensitive data
- 📊 Structured logging and metrics
- 🏌️ Golf Course API integration with tee details
- ⚠️ Event removal notifications

**Architecture:**
- 3 binaries: scraper, notifier, command processor
- 7 GitHub Actions workflows
- GitHub Gist for user preferences (no database!)
- 82.3% test coverage

## Documentation Index

### For Developers

- **[DEVELOPMENT.md](docs/DEVELOPMENT.md)** - Development workflow, testing, code style
- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** - System design, components, patterns
- **[DEBUGGING.md](docs/DEBUGGING.md)** - Debugging techniques and troubleshooting
- **[SECURITY.md](docs/SECURITY.md)** - Security features and best practices

### For Operations

- **[TELEGRAM_BOT.md](docs/TELEGRAM_BOT.md)** - Bot setup, configuration, commands
- **[RELEASES.md](docs/RELEASES.md)** - Release process and versioning

### Reference

- **[VERSION_HISTORY.md](docs/VERSION_HISTORY.md)** - Detailed changelog

## Quick Reference

### Common Commands

```bash
# Development
go test ./...                    # Run all tests
golangci-lint run                # Lint code
make build                       # Build all binaries

# Bot Testing
./vga-events-bot --dry-run      # Test command processing
./vga-events-bot --verbose      # Run with debug logging

# Workflows
gh workflow run telegram-bot-commands.yml    # Test command processor
gh workflow run telegram-bot.yml             # Test notifications
```

### Environment Variables

```bash
# Required
export TELEGRAM_BOT_TOKEN=...        # From @BotFather
export TELEGRAM_GIST_ID=...          # GitHub Gist ID
export TELEGRAM_GITHUB_TOKEN=...     # GitHub token with gist scope

# Optional
export GOLF_COURSE_API_KEY=...       # Golf course details
export TELEGRAM_ENCRYPTION_KEY=...   # Data encryption (recommended)
```

## Project Structure

```
vga-events/
├── cmd/
│   ├── vga-events/              # Event scraper
│   ├── vga-events-telegram/     # Notification sender
│   └── vga-events-bot/          # Command processor
├── internal/
│   ├── telegram/                # Telegram API client
│   ├── preferences/             # User management + Gist storage
│   ├── filter/                  # Event filtering system
│   ├── logger/                  # Structured logging
│   ├── crypto/                  # AES-256-GCM encryption
│   ├── event/                   # Event data models
│   ├── course/                  # Golf course API
│   └── scraper/                 # VGA website scraper
├── .github/workflows/           # CI/CD workflows
└── docs/                        # Documentation
```

## Getting Help

- **Issues:** Report bugs at [GitHub Issues](https://github.com/pfrederiksen/vga-events/issues)
- **Pull Requests:** See [DEVELOPMENT.md](docs/DEVELOPMENT.md) for contribution guide
- **Security:** Review [SECURITY.md](docs/SECURITY.md) for security considerations

## License

See LICENSE file for details.
