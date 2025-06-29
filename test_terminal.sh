#!/bin/bash

# Simple test script to verify terminal integration
echo "=== Terminal Integration Test ==="
echo "Current directory: $(pwd)"
echo "Date: $(date)"
echo "Go version: $(go version 2>/dev/null || echo 'Go not found')"
echo "Shell: $SHELL"
echo "Terminal: $TERM"
echo "=== Test Complete ==="
