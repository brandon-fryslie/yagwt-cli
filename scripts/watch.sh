#!/usr/bin/env bash
set -euo pipefail

echo "Watching for changes... (Ctrl+C to stop)"

if command -v fswatch >/dev/null 2>&1; then
    fswatch -o --include='\.go$' --exclude='.*' . | xargs -n1 -I{} just install
elif command -v entr >/dev/null 2>&1; then
    find . -name '*.go' | entr -c just install
else
    echo "Error: Install fswatch (brew install fswatch) or entr (brew install entr)"
    exit 1
fi
