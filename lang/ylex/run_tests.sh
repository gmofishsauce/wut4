#!/bin/bash
# Run lexer tests
# Usage: ./run_tests.sh

cd "$(dirname "$0")"

# Build ylex if needed
if [ ! -f ylex ] || [ lexer.go -nt ylex ]; then
    echo "Building ylex..."
    go build -o ylex lexer.go
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

    actual=$(./ylex "${name}.yapl" < "$input")
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
