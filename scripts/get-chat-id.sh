#!/bin/bash
#
# Helper script to retrieve Telegram chat ID
# Usage: ./scripts/get-chat-id.sh <BOT_TOKEN>
#

if [ -z "$1" ]; then
    echo "Usage: $0 <BOT_TOKEN>"
    echo ""
    echo "Steps:"
    echo "1. Create a bot via @BotFather on Telegram"
    echo "2. Start a conversation with your bot and send it a message"
    echo "3. Run this script with your bot token"
    echo ""
    echo "Example:"
    echo "  $0 1234567890:ABCdefGHIjklMNOpqrsTUVwxyz"
    exit 1
fi

BOT_TOKEN="$1"

echo "Fetching updates from Telegram API..."
echo ""

RESPONSE=$(curl -s "https://api.telegram.org/bot${BOT_TOKEN}/getUpdates")

# Check if request was successful
if echo "$RESPONSE" | grep -q '"ok":true'; then
    echo "✅ Successfully connected to Telegram API"
    echo ""

    # Extract chat IDs
    CHAT_IDS=$(echo "$RESPONSE" | grep -o '"chat":{"id":[0-9-]*' | grep -o '[0-9-]*$' | sort -u)

    if [ -z "$CHAT_IDS" ]; then
        echo "⚠️  No chat IDs found!"
        echo ""
        echo "Make sure you:"
        echo "1. Started a conversation with your bot"
        echo "2. Sent at least one message to it"
        echo ""
        echo "Then run this script again."
    else
        echo "📱 Found chat ID(s):"
        echo ""
        for CHAT_ID in $CHAT_IDS; do
            echo "  Chat ID: $CHAT_ID"
        done
        echo ""
        echo "To use with vga-events-telegram:"
        echo ""
        echo "  export TELEGRAM_BOT_TOKEN=$BOT_TOKEN"
        echo "  export TELEGRAM_CHAT_ID=$(echo $CHAT_IDS | head -1)"
    fi
else
    echo "❌ Error connecting to Telegram API"
    echo ""
    echo "Response:"
    echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
    echo ""
    echo "Check that your bot token is correct."
fi
