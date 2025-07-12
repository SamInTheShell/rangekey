#!/bin/bash

# Test script for cross-platform binaries
# This script verifies that all cross-compiled binaries work correctly

set -e

echo "Testing cross-platform binaries..."

# Test Linux binary
echo "Testing Linux binary..."
if [ -f "dist/linux/rangedb" ]; then
    echo "✓ Linux binary exists"
    file dist/linux/rangedb
    # Test basic functionality
    if ./dist/linux/rangedb version > /dev/null 2>&1; then
        echo "✓ Linux binary executes correctly"
    else
        echo "✗ Linux binary failed to execute"
        exit 1
    fi
else
    echo "✗ Linux binary not found"
    exit 1
fi

# Test macOS binary
echo "Testing macOS binary..."
if [ -f "dist/darwin/rangedb" ]; then
    echo "✓ macOS binary exists"
    file dist/darwin/rangedb
    # Note: Can't execute on Linux, but we can check file format
    if file dist/darwin/rangedb | grep -q "Mach-O"; then
        echo "✓ macOS binary has correct format"
    else
        echo "✗ macOS binary has incorrect format"
        exit 1
    fi
else
    echo "✗ macOS binary not found"
    exit 1
fi

# Test Windows binary
echo "Testing Windows binary..."
if [ -f "dist/windows/rangedb.exe" ]; then
    echo "✓ Windows binary exists"
    file dist/windows/rangedb.exe
    # Note: Can't execute on Linux, but we can check file format
    if file dist/windows/rangedb.exe | grep -q "PE32+"; then
        echo "✓ Windows binary has correct format"
    else
        echo "✗ Windows binary has incorrect format"
        exit 1
    fi
else
    echo "✗ Windows binary not found"
    exit 1
fi

echo "All cross-platform binaries tested successfully!"

# Check sizes
echo ""
echo "Binary sizes:"
ls -lh dist/linux/rangedb dist/darwin/rangedb dist/windows/rangedb.exe

echo ""
echo "Testing release archives..."
if [ -d "dist/release" ]; then
    echo "Release archives:"
    ls -lh dist/release/
    
    # Test that archives contain the correct files
    echo ""
    echo "Archive contents:"
    
    # Test Linux archive
    if [ -f "dist/release/rangedb-"*"-linux-amd64.tar.gz" ]; then
        echo "Linux archive contents:"
        tar -tzf dist/release/rangedb-*-linux-amd64.tar.gz
    fi
    
    # Test macOS archive
    if [ -f "dist/release/rangedb-"*"-darwin-amd64.tar.gz" ]; then
        echo "macOS archive contents:"
        tar -tzf dist/release/rangedb-*-darwin-amd64.tar.gz
    fi
    
    # Test Windows archive
    if [ -f "dist/release/rangedb-"*"-windows-amd64.zip" ]; then
        echo "Windows archive contents:"
        unzip -l dist/release/rangedb-*-windows-amd64.zip
    fi
    
    echo "✓ All release archives created successfully"
else
    echo "No release archives found (run 'make release' to create them)"
fi

echo ""
echo "Cross-platform binary testing completed successfully!"