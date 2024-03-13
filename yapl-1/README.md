# YAPL-1: Yet Another Programming Language, version 1

## Overview

YAPL-1 is second in a sequence of language designs targeted at the
WUT-4, a 16-bit hobbiest computer design. YAPL-0 was abandoned. The
primary long-term goal of this YAPL language project is _self-hosting_
the compiler, that is, rewriting it in YAPL and compiling the language
on the WUT-4. The entire project is built around this goal. This is a
difficult goal, one that unlikely to be met without relentless focus.

YAPL-1 is an attempt to make a minimal programming language. No
potential simplification is too minimal for YAPL-1. The short-term goal
is write an _entire_ compiler, from source code input to the generation
of assembler source code for an executable.

The Wholly Unnecessary Technologies model 4 (WUT-4) exists a design
document and a functional emulator. The emulator is found in this repo.
Some details about the WUT-4 appear below.

To meet the self-hosting goal, the compiler must have minimal system
dependencies. Every dependency must be implemented by hand in assembly
language for the WUT-4 and debugged. Dependencies will be introduced
grudgingly. Currently, there are three: the ability to get a byte of
input from an externally-defined standard input, to put a byte of output
on an externally-defined standard output, and exit with an exit code.

The assembler is contained in this `../asm` in this repo. It will not
be self-hosted for now. The Go language project has shown that the
design of an assembler intended for machine use in a compiler pipeline
may be quite different than the design of a traditional assembler
intended for programmers, and keeping the existing assembler on the
host computer retains design flexibility. There is no linker; all
executables must be self-contained.

## Constraints

The WUT-4 is a 16-bit computer with both byte and word memory
addressing.  Each running program has 64kb of data space plus 64kw
(16-bit words) of program space. The WUT-4 is a traditional RISC with
low code density, but the most important factor is data space. Every
aspect of the implementation must be oriented toward efficient use of
space. Popular modern techniques, like loading the entire source
program into memory and using pointers or indexes as references,
are not candidates for this implementation.

Everything must stream, and anything to be stored in memory must be
designed with attention to space reduction.

The WUT-4 kernel will eventually offer memory sharing with very
lightweight switching between process-like entities in a predefined
group of processes. The compiler will eventually be implemented as such
a predefined group, enabling a lightweight multiple-pass structure. I
hope this will allow moderately sophisticated modern compiler techniques
like a control-flow graph and register allocation by linear scan. My
belief is that dataflow techniques (like SSA, sea of nodes, etc.) are
probably out of reach due to memory constraints, but we'll see when we
get there.

## Language Design

This first stage of YAPL evolution is designed to trivialize the lexical
analysis and parsing ("front end") phases of the compilation. These
phases dominate discussions and undergraduate compiler courses, but are
actually the easy parts of a real compiler. Trivializing them allows
focus on the hard parts.

YAPL-1 is a procedural language in the style of BCPL or C. It's just
absurdly simplified.

### Lexical structure

Identifiers consist of a single lower-case letter. Identifiers are used
to name functions and variables.

Keywords and builtins consist of a single upper-case letter.

Numeric constants consist of a single digit 0..9. No operators may be
applied to numeric constants. There is no support for other number
bases.

Whitespace consists of spaces, tabs, and newlines. Whitespace is
required where other rules would be broken without it, e.g. between a
keyword and a variable name. Whitespace is optional elsewhere.

Comments may be introduced with the character # (hash). Comments are
terminated by a newline character. This is the only place in the
language where newline is distinguished from the other whitespace
characters.

Semicolon ; is used as a statement separator. Its use is conventional.
The details are defined in the syntactic structure below.

The following additional characters are recognized and given meanings
described below: { } = +

### Syntactic structure

A program consists of a sequence of declarations. There are two types of
declarations: variables and functions.

Variable declarations consist of the keyword V, a variable name, and an
optional constant assignment.

A constant assignment consists of = followed by a numeric constant as
defined lexically.

Function declarations consist of the keyword F, an identifier, and the
function body. The identifier defines the function's name.

The function body consists of a block.

A block consists of an opening curly brace, a sequence of 0 or more
statements, and a closing curly brace. Variables may not be defined
within a block.

Statements consist of expressions, function calls, and conditional
statements.

An expression begins with an identifier and the character = It may be
followed by either a term and a semicolon or by a term, the character +,
another term, and a semicolon.

A term is a variable name or numeric constant.

A function call is the identifier of the known function followed by a
semicolon. A function becomes known when its declaration is seen, so
functions may be recursive.

A conditional statement consists of the keyword I followed by a
conditional expression followed by a block followed by the keyword E
followed by another block. The E and the second block are required.

A conditional expression consists of two terms separated by whitespace
with no ; or other punctuation.

Every program should define a function named m. Execution begins at this
function.

The builtin variables A, B, C, and D may be assigned. These values are
displayed by the emulator when the program exits.

Execution of the builtin Q causes the program to quit. In the emulator,
this results in a state dump which displays A, B, C, and D among other
values.

### Semantic structure of YAPL-1

All variables have type unsigned 8-bit value, the equivalent of uint8 in
Golang. Variables not initialized by the program are automatically
initialized to 0.

All identifiers must be defined before use in source code textual order.

Initially, the WUT-4 input/output system supports only a single output
stream, equivalent to the standard output. The output of the compiler is
a single document containing messages and, if the source file is
correct, assembly language. Messages are packaged as assembler comments
starting with ";" in the first column.

### Example of YAPL-1

```
    # Compute the first few Fibonacci numbers

    V a = 0 ;     # variable "a" is fib(0)
    V b = 1 ;     # fib(1)
    V r     ;     # variable "r" result
    V n = 8 ;     # variable "n" limit (fib(6))

    F m {              # function "m"
        r = a + b ;
        I r n {        # if r == m
            W = a ;    # write some values to display variables
            X = b ;
            Y = r ;
            Z = n ;
            Q          # quit to OS
        } E {          # else
            a = b      # shift down
            b = r
            m          # recursively call m
        }
    }
```
