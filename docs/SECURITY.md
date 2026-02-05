# Security Features

## Rate Limiting

- Per-user rate limiting: 10 commands/minute using sliding window algorithm
- Prevents command spam and DoS attempts
- Automatic cleanup to prevent memory growth

## Data Protection

**File Permissions:**
- Local snapshot files use 0600 permissions (owner-only access)

**Encryption:**
- Optional AES-256-GCM encryption for sensitive Gist data
- PBKDF2 key derivation with 100,000 iterations
- Encrypts: event notes, event statuses, invite codes
- Backward compatible with unencrypted data

**Setup:**
```bash
export TELEGRAM_ENCRYPTION_KEY="your-strong-passphrase-here"
```

## Input Validation

**Length Limits:**
- Notes: 500 characters
- City names: 100 characters
- Search keywords: 100 characters

**Sanitization:**
- Control character sanitization
- Validation before processing
- Parameter extraction with validation helpers

## Error Handling

- Sanitized error messages (no sensitive API responses exposed)
- Proper error wrapping for debugging
- Never expose internal details to users

## Best Practices

1. **Never commit secrets** - Use environment variables
2. **Use encryption key** - Enable `TELEGRAM_ENCRYPTION_KEY` for production
3. **Review permissions** - GitHub Secrets should be restricted to workflows
4. **Monitor rate limits** - Check logs for suspicious activity
