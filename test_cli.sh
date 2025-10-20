#!/bin/bash

echo "Testing School Finder CLI Commands"
echo "===================================="
echo ""

echo "1. Testing search command (limit 2):"
echo "   ./schoolfinder search --limit 2 'Lincoln High'"
./schoolfinder search --limit 2 "Lincoln High" | jq -r '.[0].name, .[0].ncessch'
echo ""

echo "2. Testing search with state filter (CA, limit 1):"
echo "   ./schoolfinder search --state CA --limit 1 'High School'"
./schoolfinder search --state CA --limit 1 "High School" | jq -r '.[0].name, .[0].state'
echo ""

echo "3. Testing details command:"
echo "   ./schoolfinder details 291867001016"
./schoolfinder details 291867001016 | jq -r '.name, .city, .state, .enrollment'
echo ""

echo "4. Testing scrape command (without API key - should fail):"
echo "   ANTHROPIC_API_KEY='' ./schoolfinder scrape 291867001016"
ANTHROPIC_API_KEY="" ./schoolfinder scrape 291867001016 2>&1 | head -1
echo ""

echo "All CLI tests completed!"
