#!/bin/bash
cd /Users/shrike/projects/commander
echo "Building project..."
go build ./...
echo "Build status: $?"
echo ""
echo "Running go vet..."
go vet ./...
echo "Vet status: $?"
