#!/bin/bash

# YAPL Parser Test Script
# Runs positive and negative tests

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LEXER="$SCRIPT_DIR/../lex/lexer"
PARSER="$SCRIPT_DIR/parser"
TESTDATA="$SCRIPT_DIR/testdata"

PASS=0
FAIL=0

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================"
echo "YAPL Parser Test Suite"
echo "========================================"
echo ""

# Check that lexer and parser exist
if [ ! -x "$LEXER" ]; then
    echo -e "${RED}Error: Lexer not found at $LEXER${NC}"
    exit 1
fi

if [ ! -x "$PARSER" ]; then
    echo -e "${RED}Error: Parser not found at $PARSER${NC}"
    echo "Building parser..."
    cd "$SCRIPT_DIR" && go build -o parser *.go
    if [ $? -ne 0 ]; then
        echo -e "${RED}Failed to build parser${NC}"
        exit 1
    fi
fi

echo "=== Positive Tests (should pass) ==="
echo ""

for file in "$TESTDATA"/[0-9]*.yapl; do
    if [ -f "$file" ]; then
        basename=$(basename "$file")
        echo -n "Testing $basename... "

        # Run lexer | parser, capture stderr
        output=$("$LEXER" "$basename" < "$file" 2>&1 | "$PARSER" 2>&1)
        status=$?

        if [ $status -eq 0 ]; then
            echo -e "${GREEN}PASS${NC}"
            ((PASS++))
        else
            echo -e "${RED}FAIL${NC}"
            echo "  Error output:"
            echo "$output" | head -5 | sed 's/^/    /'
            ((FAIL++))
        fi
    fi
done

echo ""
echo "=== Negative Tests (should fail) ==="
echo ""

for file in "$TESTDATA"/err_*.yapl; do
    if [ -f "$file" ]; then
        basename=$(basename "$file")
        echo -n "Testing $basename... "

        # Run lexer | parser, capture stderr
        output=$("$LEXER" "$basename" < "$file" 2>&1 | "$PARSER" 2>&1)
        status=$?

        if [ $status -ne 0 ]; then
            echo -e "${GREEN}PASS (correctly rejected)${NC}"
            # Show the error message
            echo "$output" | grep -E "^[^#]" | head -3 | sed 's/^/    /'
            ((PASS++))
        else
            echo -e "${RED}FAIL (should have been rejected)${NC}"
            ((FAIL++))
        fi
    fi
done

echo ""
echo "========================================"
echo -e "Results: ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}"
echo "========================================"

if [ $FAIL -gt 0 ]; then
    exit 1
fi
exit 0
