#!/bin/bash

# Script to send digest notifications to users
# Usage: ./send-digest.sh <frequency>
# frequency: "daily" or "weekly"

set -e

FREQUENCY=$1

if [ -z "$FREQUENCY" ]; then
  echo "Usage: $0 <frequency>"
  echo "frequency: daily or weekly"
  exit 1
fi

if [ "$FREQUENCY" != "daily" ] && [ "$FREQUENCY" != "weekly" ]; then
  echo "Error: frequency must be 'daily' or 'weekly'"
  exit 1
fi

echo "Sending $FREQUENCY digests..."

# Load preferences
PREFS_JSON=$(cat preferences.json)

# Find users with matching digest frequency
USERS=$(echo "$PREFS_JSON" | jq -r --arg freq "$FREQUENCY" '
  to_entries[] |
  select(.value.active == true and .value.digest_frequency == $freq and (.value.pending_events // [] | length) > 0) |
  .key
')

if [ -z "$USERS" ]; then
  echo "No users with pending $FREQUENCY digest events"
  exit 0
fi

USER_COUNT=$(echo "$USERS" | wc -l | tr -d ' ')
echo "Found $USER_COUNT user(s) with pending $FREQUENCY digest events"

# Send digest to each user
while IFS= read -r CHAT_ID; do
  [ -z "$CHAT_ID" ] && continue

  echo ""
  echo "Processing digest for user $CHAT_ID..."

  # Get pending events count
  PENDING_COUNT=$(jq -r --arg chat "$CHAT_ID" '.[$chat].pending_events // [] | length' preferences.json)

  if [ "$PENDING_COUNT" -eq 0 ]; then
    echo "  No pending events, skipping"
    continue
  fi

  echo "  Pending events: $PENDING_COUNT"

  # Extract pending events to temporary file
  jq --arg chat "$CHAT_ID" '{new_events: .[$chat].pending_events}' preferences.json > "digest_${CHAT_ID}.json"

  # Format digest message (we'll use a Go helper for this)
  # For now, send individual events
  ./vga-events-telegram --chat-id "$CHAT_ID" --events-file "digest_${CHAT_ID}.json" --max-messages 20

  if [ $? -eq 0 ]; then
    echo "  ✅ Successfully sent $FREQUENCY digest"

    # Clear pending events
    jq --arg chat "$CHAT_ID" '.[$chat].pending_events = []' preferences.json > preferences.tmp && mv preferences.tmp preferences.json

    echo "  📝 Cleared pending events"
  else
    echo "  ❌ Failed to send digest"
  fi

  # Rate limiting between users
  sleep 1

done <<< "$USERS"

# Save updated preferences
echo ""
echo "Saving updated preferences..."
GIST_CONTENT=$(jq -Rs . preferences.json)
curl -s -X PATCH \
  -H "Authorization: token $TELEGRAM_GITHUB_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  "https://api.github.com/gists/$TELEGRAM_GIST_ID" \
  -d "{\"files\":{\"preferences.json\":{\"content\":$GIST_CONTENT}}}" > /dev/null

echo "✅ Digest delivery complete"
