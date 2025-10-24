#!/bin/bash

# Test MCP server by running it for 2 seconds
# The server should start up, sync documents, and wait for MCP messages

echo "Starting MCP server test..."
echo ""

# Run the server in background
./markdown-vector-mcp &
PID=$!

# Wait 2 seconds
sleep 2

# Kill the server
kill $PID 2>/dev/null || true

echo ""
echo "Server test complete. Check stderr output above for:"
echo "  - [INFO] markdown-vector-mcp starting..."
echo "  - [INFO] Configuration loaded successfully"
echo "  - [INFO] Using device: cpu"
echo "  - [INFO] Syncing documents..."
echo "  - [INFO] Sync complete: +X, ~X, -X"
echo "  - [INFO] Starting MCP server..."
echo "  - [INFO] Registered 5 MCP tools"
