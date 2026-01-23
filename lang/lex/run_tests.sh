#!/bin/bash
# Run lexer tests
# Usage: ./run_tests.sh

cd "$(dirname "$0")"

# Build lexer if needed
if [ ! -f lexer ] || [ lexer.go -nt lexer ]; then
    echo "Building lexer..."
    go build -o lexer lexer.go
fi

PASS=0
FAIL=0

for input in testdata/*.yapl; do
    name=$(basename "$input" .yapl)
    expected="testdata/${name}.expected"

    if [ ! -f "$expected" ]; then
        echo "SKIP: $name (no expected output)"
        continue
    fi

    actual=$(./lexer "${name}.yapl" < "$input")
    expected_content=$(cat "$expected")

    if [ "$actual" = "$expected_content" ]; then
        echo "PASS: $name"
        ((PASS++))
    else
        echo "FAIL: $name"
        echo "--- Expected ---"
        cat "$expected"
        echo "--- Actual ---"
        echo "$actual"
        echo "---"
        ((FAIL++))
    fi
done

echo ""
echo "Results: $PASS passed, $FAIL failed"
exit $FAIL
