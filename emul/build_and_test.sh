#!/bin/bash
# Build and run all tests for WUT-4 emulator

set -e  # Exit on error

echo "WUT-4 Emulator Test Suite"
echo "=========================="
echo ""

# Build the assembler if needed
echo "Building assembler..."
cd ../yasm
go build -o yasm .
if [ ! -f yasm ]; then
    echo "ERROR: Failed to build assembler"
    exit 1
fi
cd ../emul

# Assemble all test programs
echo ""
echo "Assembling test programs..."

# Function to assemble a file
assemble_test() {
    local asm_file=$1
    local out_file="${asm_file%.w4a}.out"
    echo "  Assembling $(basename $asm_file)..."
    ../yasm/yasm "$asm_file" "$out_file"
    if [ $? -ne 0 ]; then
        echo "  ERROR: Failed to assemble $asm_file"
        return 1
    fi
}

# Assemble all test programs
for category in testdata/*/; do
    if [ -d "$category" ]; then
        for asm_file in "$category"*.w4a; do
            if [ -f "$asm_file" ]; then
                assemble_test "$asm_file" || exit 1
            fi
        done
    fi
done

echo ""
echo "Building emulator..."
go build -o emul .

echo ""
echo "Running unit tests..."
go test -v -run "^Test(Decode|MMU|LoadStore|Alignment|Context|Register0)" .

echo ""
echo "Running integration tests..."
go test -v -run "^Test(Integration|Arithmetic|Memory|Branch)" .

echo ""
echo "=========================="
echo "All tests completed successfully!"
