#!/bin/bash
# Iteratively compile examples/showcase.rb, reporting errors for Claude to fix.
# Usage: ./scripts/iterate_showcase.sh [max_attempts]

set -euo pipefail
cd "$(dirname "$0")/.."

MAX=${1:-20}
SHOWCASE=examples/showcase.rb

echo "=== Building thanos ==="
go build -o thanos .

for i in $(seq 1 "$MAX"); do
    echo ""
    echo "=== Attempt $i/$MAX ==="

    # Check Ruby syntax first
    if ! ruby -c "$SHOWCASE" 2>/dev/null; then
        echo "RUBY_SYNTAX_ERROR"
        ruby -c "$SHOWCASE" 2>&1 || true
        exit 1
    fi

    # Try thanos compile
    OUTPUT=$(./thanos compile -s "$SHOWCASE" 2>&1 | sed 's/\x1b\[[0-9;]*m//g')

    if echo "$OUTPUT" | grep -q "^syntax error\|^line \|Error\|error\|panic"; then
        echo "COMPILE_ERROR:"
        echo "$OUTPUT" | grep -E "^syntax error|^line |Error|error|panic" | head -5
        exit 1
    fi

    # Try thanos exec (compares Ruby vs Go output)
    EXEC_OUTPUT=$(./thanos exec -s "$SHOWCASE" 2>&1 | sed 's/\x1b\[[0-9;]*m//g')

    if echo "$EXEC_OUTPUT" | grep -q "Error\|error\|panic\|FAIL\|failed"; then
        echo "EXEC_ERROR:"
        echo "$EXEC_OUTPUT" | head -30
        exit 1
    fi

    if [ -z "$EXEC_OUTPUT" ]; then
        echo "SUCCESS: Ruby and Go output match!"
        echo ""
        echo "=== Ruby output preview ==="
        ruby "$SHOWCASE" 2>&1 | head -20
        echo "..."
        exit 0
    else
        echo "OUTPUT_MISMATCH:"
        echo "$EXEC_OUTPUT" | head -20
        exit 1
    fi
done

echo "EXHAUSTED: $MAX attempts"
exit 1
