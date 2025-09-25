#!/bin/bash

# Test script for DeepSeek function calling

echo "Testing DeepSeek Function Calling"
echo "================================="
echo ""
echo "Set debug mode to see tool call details"
export DEECLI_DEBUG=1

# Start the chat with a prompt that should trigger tool usage
echo "Testing with: 'List the files in this project'"
echo ""

# Use a here document to send input
./deecli chat << 'EOF'
List the files in this project folder
/quit
EOF