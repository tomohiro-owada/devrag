#!/bin/bash

# Test MCP protocol by sending an initialize request
# MCP servers should respond to JSON-RPC requests over stdio

echo "Testing MCP protocol..."
echo ""

# Send an initialize request (MCP protocol)
# The server should respond with a JSON-RPC response
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "test-client",
      "version": "1.0.0"
    }
  }
}' | ./devrag 2>&1 &

# Wait for server to start
sleep 3

# Kill any remaining process
pkill -f devrag 2>/dev/null || true

echo ""
echo "Test complete. If you see JSON-RPC responses above, the MCP protocol is working."
