#!/bin/bash

# Script to view AMG-QAD test logs in a readable format

RESULTS_DIR="./mnt/weka/amg-qad/results"
RESULTS_FILE="$RESULTS_DIR/results.jsonl"

if [ ! -f "$RESULTS_FILE" ]; then
    echo "Results file not found at: $RESULTS_FILE"
    echo "Trying local results file..."
    RESULTS_FILE="./results/results.jsonl"
    if [ ! -f "$RESULTS_FILE" ]; then
        echo "No results file found!"
        exit 1
    fi
fi

echo "Reading results from: $RESULTS_FILE"
echo "==============================================="

# Get the last test run (most recent entry)
LAST_ENTRY=$(tail -n 1 "$RESULTS_FILE")

# Check if jq is available for pretty formatting
if command -v jq >/dev/null 2>&1; then
    echo "📊 LATEST TEST RUN SUMMARY:"
    echo "$LAST_ENTRY" | jq -r '
        "Timestamp: " + .timestamp +
        "\nOverall Status: " + (if .passed then "✅ PASSED" else "❌ FAILED" end) +
        "\nDuration: " + .duration +
        "\nTests: " + (.parameters | gsub("test_suite_"; "") | gsub("_tests"; " tests"))
    '
    
    echo ""
    echo "📋 INDIVIDUAL TEST DETAILS:"
    echo "$LAST_ENTRY" | jq -r '
        if .tests then
            .tests[] | 
            "\n🧪 Test: " + .name +
            "\n   Status: " + (if .passed then "✅ PASSED" else "❌ FAILED" end) +
            "\n   Duration: " + .duration +
            "\n   Logs:\n" + .logs +
            "\n" + ("-" * 50)
        else
            "\n📝 FULL LOGS:\n" + .logs
        end
    '
else
    echo "📊 LATEST TEST RUN (Raw JSON):"
    echo "$LAST_ENTRY" | python3 -m json.tool 2>/dev/null || echo "$LAST_ENTRY"
fi

echo ""
echo "💡 TIP: Install 'jq' for better formatted output: sudo apt install jq"
