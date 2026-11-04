#!/bin/sh
set -eu

TESTS_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ $# -gt 0 ]; then
    TEST_SCRIPT="$1"
    case "$TEST_SCRIPT" in
        /*) ;;
        *) TEST_SCRIPT="$TESTS_DIR/$TEST_SCRIPT" ;;
    esac
    echo "Running single test: $TEST_SCRIPT"
    "$TEST_SCRIPT"
    exit 0
fi

echo "Running all docker tests..."
COUNT=0
for test in "$TESTS_DIR"/*.sh; do
    case "$(basename "$test")" in
        run-all.sh|lib.sh)
            continue
            ;;
    esac
    echo "----------------------------------------"
    echo "Executing: $(basename "$test")"
    echo "----------------------------------------"
    "$test"
    COUNT=$((COUNT + 1))
done

echo "========================================"
echo "All $COUNT docker tests passed successfully!"
echo "========================================"

