#!/bin/bash

# stop-cluster.sh - Stop the RangeDB cluster

set -e

# Configuration
DATA_DIR="./data"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Stopping RangeDB cluster...${NC}"

# Stop nodes by PID if available
if [ -f "$DATA_DIR/node1.pid" ]; then
    NODE1_PID=$(cat "$DATA_DIR/node1.pid")
    if kill -0 "$NODE1_PID" 2>/dev/null; then
        echo "Stopping Node 1 (PID: $NODE1_PID)..."
        kill -TERM "$NODE1_PID" || true
    fi
    rm -f "$DATA_DIR/node1.pid"
fi

if [ -f "$DATA_DIR/node2.pid" ]; then
    NODE2_PID=$(cat "$DATA_DIR/node2.pid")
    if kill -0 "$NODE2_PID" 2>/dev/null; then
        echo "Stopping Node 2 (PID: $NODE2_PID)..."
        kill -TERM "$NODE2_PID" || true
    fi
    rm -f "$DATA_DIR/node2.pid"
fi

if [ -f "$DATA_DIR/node3.pid" ]; then
    NODE3_PID=$(cat "$DATA_DIR/node3.pid")
    if kill -0 "$NODE3_PID" 2>/dev/null; then
        echo "Stopping Node 3 (PID: $NODE3_PID)..."
        kill -TERM "$NODE3_PID" || true
    fi
    rm -f "$DATA_DIR/node3.pid"
fi

# Fallback: kill all rangedb processes
echo "Cleaning up any remaining rangedb processes..."
pkill -f "rangedb server" || true

# Wait for processes to stop
sleep 2

# Force kill if still running
pkill -9 -f "rangedb server" || true

echo -e "${GREEN}✓ Cluster stopped successfully!${NC}"
echo ""
echo "To clean up data directories:"
echo "  rm -rf $DATA_DIR"
echo "  rm -rf ./logs"
