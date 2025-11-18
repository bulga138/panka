#!/bin/bash
# release.sh
# Run this in Git Bash, WSL, or Linux/macOS

# Exit immediately if a command exits with a non-zero status
set -e

if [ -z "$1" ]; then
    echo "Usage: ./release.sh <version>"
    echo "Example: ./release.sh v1.0.0"
    exit 1
fi

VERSION=$1

# 1. Check if git is clean (Optional, good practice)
if [ -n "$(git status --porcelain)" ]; then
    echo "Error: Working directory is not clean. Commit changes first."
    exit 1
fi

# 2. Create git tag
echo "Creating tag $VERSION..."
git tag -a "$VERSION" -m "Release $VERSION"

# 3. Build with the tagged version
# The Makefile uses this TAG variable to inject the version into the binary
echo "Building binary..."
make release TAG="$VERSION"

echo "---------------------------------------"
echo "SUCCESS: Release $VERSION created and built."
echo "Don't forget to push the tag: git push origin $VERSION"