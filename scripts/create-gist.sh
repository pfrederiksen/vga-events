#!/bin/bash
#
# Helper script to create a GitHub Gist for storing Telegram bot preferences
# Usage: ./scripts/create-gist.sh <GITHUB_TOKEN>
#

if [ -z "$1" ]; then
    echo "Usage: $0 <GITHUB_TOKEN>"
    echo ""
    echo "Steps:"
    echo "1. Go to https://github.com/settings/tokens"
    echo "2. Click 'Generate new token' → 'Generate new token (classic)'"
    echo "3. Give it a name like 'VGA Events Bot'"
    echo "4. Check the 'gist' scope"
    echo "5. Click 'Generate token' and copy it"
    echo ""
    echo "Then run:"
    echo "  $0 ghp_yourTokenHere"
    exit 1
fi

GITHUB_TOKEN="$1"

echo "Creating private Gist for VGA Events Bot preferences..."
echo ""

# Create the Gist
RESPONSE=$(curl -s -X POST https://api.github.com/gists \
  -H "Authorization: token $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  -d '{
    "description": "VGA Events Telegram Bot Preferences",
    "public": false,
    "files": {
      "preferences.json": {
        "content": "{}"
      }
    }
  }')

# Check if successful
if echo "$RESPONSE" | grep -q '"id"'; then
    GIST_ID=$(echo "$RESPONSE" | grep -o '"id": *"[^"]*"' | head -1 | sed 's/"id": *"\([^"]*\)"/\1/')

    echo "✅ Gist created successfully!"
    echo ""
    echo "Gist ID: $GIST_ID"
    echo "URL: https://gist.github.com/$GIST_ID"
    echo ""
    echo "Add these to your GitHub repository secrets:"
    echo ""
    echo "  TELEGRAM_GIST_ID=$GIST_ID"
    echo "  TELEGRAM_GITHUB_TOKEN=$GITHUB_TOKEN"
    echo ""
    echo "Go to: https://github.com/YOUR_USERNAME/vga-events/settings/secrets/actions"
else
    echo "❌ Error creating Gist"
    echo ""
    echo "Response:"
    echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
    echo ""
    echo "Check that your GitHub token has the 'gist' scope."
    exit 1
fi
