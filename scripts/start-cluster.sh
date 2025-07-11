#!/bin/bash

# start-cluster.sh - Start a 3-node RangeDB cluster for testing

set -e

# Configuration
BINARY="./build/rangedb"
DATA_DIR="./data"
LOG_DIR="./logs"

# Node configurations
NODE1_PEER="localhost:8080"
NODE1_CLIENT="localhost:8081"
NODE1_DATA="$DATA_DIR/node1"
NODE1_LOG="$LOG_DIR/node1.log"

NODE2_PEER="localhost:8082"
NODE2_CLIENT="localhost:8083"
NODE2_DATA="$DATA_DIR/node2"
NODE2_LOG="$LOG_DIR/node2.log"

NODE3_PEER="localhost:8084"
NODE3_CLIENT="localhost:8085"
NODE3_DATA="$DATA_DIR/node3"
NODE3_LOG="$LOG_DIR/node3.log"

PEERS="$NODE1_PEER,$NODE2_PEER,$NODE3_PEER"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting RangeDB 3-node cluster...${NC}"

# Check if binary exists
if [ ! -f "$BINARY" ]; then
    echo -e "${RED}Error: Binary not found at $BINARY${NC}"
    echo "Please run 'make build' first"
    exit 1
fi

# Create directories
mkdir -p "$DATA_DIR" "$LOG_DIR"
mkdir -p "$NODE1_DATA" "$NODE2_DATA" "$NODE3_DATA"

# Clean up any existing processes
echo -e "${YELLOW}Cleaning up existing processes...${NC}"
pkill -f "rangedb server" || true
sleep 2

# Start Node 1 (cluster init)
echo -e "${GREEN}Starting Node 1 (cluster init)...${NC}"
$BINARY server \
    --peer-address="$NODE1_PEER" \
    --client-address="$NODE1_CLIENT" \
    --data-dir="$NODE1_DATA" \
    --peers="$PEERS" \
    --cluster-init \
    --log-level=info \
    > "$NODE1_LOG" 2>&1 &

NODE1_PID=$!
echo "Node 1 PID: $NODE1_PID"

# Wait for Node 1 to start
sleep 3

# Start Node 2 (join cluster)
echo -e "${GREEN}Starting Node 2 (join cluster)...${NC}"
$BINARY server \
    --peer-address="$NODE2_PEER" \
    --client-address="$NODE2_CLIENT" \
    --data-dir="$NODE2_DATA" \
    --join="$NODE1_PEER,$NODE2_PEER,$NODE3_PEER" \
    --log-level=info \
    > "$NODE2_LOG" 2>&1 &

NODE2_PID=$!
echo "Node 2 PID: $NODE2_PID"

# Wait for Node 2 to start
sleep 3

# Start Node 3 (join cluster)
echo -e "${GREEN}Starting Node 3 (join cluster)...${NC}"
$BINARY server \
    --peer-address="$NODE3_PEER" \
    --client-address="$NODE3_CLIENT" \
    --data-dir="$NODE3_DATA" \
    --join="$NODE1_PEER,$NODE2_PEER,$NODE3_PEER" \
    --log-level=info \
    > "$NODE3_LOG" 2>&1 &

NODE3_PID=$!
echo "Node 3 PID: $NODE3_PID"

# Save PIDs for later cleanup
echo "$NODE1_PID" > "$DATA_DIR/node1.pid"
echo "$NODE2_PID" > "$DATA_DIR/node2.pid"
echo "$NODE3_PID" > "$DATA_DIR/node3.pid"

# Wait for all nodes to start
echo -e "${YELLOW}Waiting for cluster to initialize...${NC}"
sleep 5

# Check if all nodes are running
if kill -0 "$NODE1_PID" 2>/dev/null && kill -0 "$NODE2_PID" 2>/dev/null && kill -0 "$NODE3_PID" 2>/dev/null; then
    echo -e "${GREEN}✓ All nodes started successfully!${NC}"
    echo ""
    echo "Cluster Information:"
    echo "  Node 1: $NODE1_CLIENT (peer: $NODE1_PEER)"
    echo "  Node 2: $NODE2_CLIENT (peer: $NODE2_PEER)"
    echo "  Node 3: $NODE3_CLIENT (peer: $NODE3_PEER)"
    echo ""
    echo "Example usage:"
    echo "  $BINARY --endpoints=$NODE1_CLIENT put /test/key 'hello world'"
    echo "  $BINARY --endpoints=$NODE1_CLIENT get /test/key"
    echo "  $BINARY --endpoints=$NODE1_CLIENT admin cluster status"
    echo ""
    echo "Logs are available at:"
    echo "  $NODE1_LOG"
    echo "  $NODE2_LOG"
    echo "  $NODE3_LOG"
    echo ""
    echo "To stop the cluster: ./scripts/stop-cluster.sh"
else
    echo -e "${RED}✗ Some nodes failed to start. Check logs for details.${NC}"
    exit 1
fi
