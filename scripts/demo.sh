#!/bin/bash

# RangeDB Quick Demo
# Demonstrates the key features of RangeDB

echo "🚀 RangeDB Quick Demo"
echo "===================="
echo

# Check if rangedb binary exists
if [ ! -f "./build/rangedb" ]; then
    echo "⚠️  RangeDB binary not found. Building..."
    make build
fi

echo "1. Start RangeDB server (in background)"
echo "   ./build/rangedb server --cluster-init"
echo
./build/rangedb server --cluster-init --data-dir=/tmp/rangedb-demo &
SERVER_PID=$!

# Wait for server to start
sleep 3

echo "2. Basic key-value operations"
echo "   Store user data..."
./build/rangedb put /users/alice '{"name": "Alice Smith", "email": "alice@example.com", "age": 30}'
./build/rangedb put /users/bob '{"name": "Bob Johnson", "email": "bob@example.com", "age": 25}'
./build/rangedb put /users/charlie '{"name": "Charlie Brown", "email": "charlie@example.com", "age": 35}'

echo
echo "   Retrieve user data..."
echo "   Alice: $(./build/rangedb get /users/alice)"
echo "   Bob: $(./build/rangedb get /users/bob)"

echo
echo "3. Range queries"
echo "   Find all users..."
./build/rangedb range /users/ /users/z

echo
echo "4. Batch operations"
echo "   Add multiple products at once..."
./build/rangedb batch put \
  /products/laptop '{"name": "Laptop", "price": 999.99}' \
  /products/mouse '{"name": "Mouse", "price": 29.99}' \
  /products/keyboard '{"name": "Keyboard", "price": 79.99}'

echo
echo "   View all products..."
./build/rangedb range /products/ /products/z

echo
echo "5. Configuration management"
echo "   Set system configuration..."
./build/rangedb admin config set max_connections 1000
./build/rangedb admin config set cache_size 512MB

echo
echo "   View configuration..."
echo "   Max connections: $(./build/rangedb admin config get max_connections | cut -d'=' -f2 | xargs)"
echo "   Cache size: $(./build/rangedb admin config get cache_size | cut -d'=' -f2 | xargs)"

echo
echo "6. Cluster status"
echo "   Check cluster health..."
./build/rangedb admin cluster status

echo
echo "7. Version information"
./build/rangedb version

echo
echo "8. Cleanup"
echo "   Stopping server..."
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo "   Cleaning up data..."
rm -rf /tmp/rangedb-demo

echo
echo "✅ RangeDB demo completed successfully!"
echo "🎉 Ready for production use!"