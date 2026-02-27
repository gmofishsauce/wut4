package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// LineKind classifies each line of assembly source.
type LineKind int

const (
	LineBlank       LineKind = iota
	LineComment                // ; ...
	LineDirective              // .code, .data, etc.
	LineLabel                  // name:
	LineInstruction            // (4-space indent) op args...
	LineDeleted                // sentinel: omit from output
)

// Line holds one parsed line of assembly.
type Line struct {
	Kind LineKind
	Raw  string   // preserved verbatim for pass-through
	Op   string   // lower-cased opcode (instructions only)
	Args []string // trimmed operands split on comma (instructions only)
}

// invertCond maps each conditional branch mnemonic to its logical inverse.
var invertCond = map[string]string{
	"brz":   "brnz",
	"brnz":  "brz",
	"breq":  "brneq",
	"brneq": "breq",
	"brc":   "brnc",
	"brnc":  "brc",
	"bruge": "brult",
	"brult": "bruge",
	"brsge": "brslt",
	"brslt": "brsge",
}

// wordSize returns the byte size a line contributes to the assembled output.
// Non-instruction lines (labels, blanks, etc.) and deleted lines contribute 0.
// Pseudo-instructions that expand to multiple words:
//
//	jal  → LUI + JAL  = 4 bytes
//	ldi  → LUI + ADI  = 4 bytes  (assembler always uses 2-word form)
//	srr  → ldi + LSP  = 6 bytes
//	srw  → ldi + SSP  = 6 bytes
//
// Every other instruction assembles to exactly 1 word = 2 bytes.
func wordSize(l *Line) int {
	if l.Kind != LineInstruction {
		return 0
	}
	switch l.Op {
	case "jal", "ldi":
		return 4
	case "srr", "srw":
		return 6
	default:
		return 2
	}
}

// addrMap holds the byte address of every label and every line.
type addrMap struct {
	label map[string]int // label name → byte address
	line  []int          // line index → byte address
}

// buildAddrMap computes pseudo-addresses by scanning lines in order.
// Deleted lines contribute 0 bytes (they have already been removed logically).
func buildAddrMap(lines []*Line) *addrMap {
	am := &addrMap{
		label: make(map[string]int),
		line:  make([]int, len(lines)),
	}
	addr := 0
	for i, l := range lines {
		am.line[i] = addr
		switch l.Kind {
		case LineLabel:
			am.label[labelName(l.Raw)] = addr
		case LineInstruction:
			addr += wordSize(l)
		}
	}
	return am
}

// labelName strips the trailing ':' from a label line's raw text.
func labelName(raw string) string {
	return strings.TrimSuffix(strings.TrimSpace(raw), ":")
}

// parseAll converts raw text lines into Line structs.
func parseAll(rawLines []string) []*Line {
	lines := make([]*Line, len(rawLines))
	for i, raw := range rawLines {
		lines[i] = parseLine(raw)
	}
	return lines
}

// parseLine classifies and parses a single raw line.
func parseLine(raw string) *Line {
	l := &Line{Raw: raw}
	trimmed := strings.TrimSpace(raw)

	if trimmed == "" {
		l.Kind = LineBlank
		return l
	}
	if strings.HasPrefix(trimmed, ";") {
		l.Kind = LineComment
		return l
	}
	if strings.HasPrefix(trimmed, ".") {
		l.Kind = LineDirective
		return l
	}
	if strings.HasSuffix(trimmed, ":") {
		l.Kind = LineLabel
		return l
	}
	// Must be an instruction (4-space indent).
	l.Kind = LineInstruction
	parts := strings.Fields(trimmed)
	l.Op = strings.ToLower(parts[0])
	if len(parts) > 1 {
		rest := strings.Join(parts[1:], " ")
		for _, arg := range strings.Split(rest, ",") {
			l.Args = append(l.Args, strings.TrimSpace(arg))
		}
	}
	return l
}

// writeAll emits all non-deleted lines to w.
func writeAll(w *bufio.Writer, lines []*Line) {
	for _, l := range lines {
		if l.Kind == LineDeleted {
			continue
		}
		fmt.Fprintln(w, l.Raw)
	}
}

// nextInstr returns the index of the next LineInstruction at or after start,
// skipping blank, comment, and deleted lines.  Returns -1 if a label or
// directive is encountered first, or end of slice is reached.
func nextInstr(lines []*Line, start int) int {
	for i := start; i < len(lines); i++ {
		switch lines[i].Kind {
		case LineInstruction:
			return i
		case LineBlank, LineComment, LineDeleted:
			// transparent
		default:
			// Label or Directive — break the window
			return -1
		}
	}
	return -1
}

// nextNonTrivial returns the index of the next line that is not blank,
// comment, or deleted, starting at index start.
func nextNonTrivial(lines []*Line, start int) int {
	for i := start; i < len(lines); i++ {
		switch lines[i].Kind {
		case LineBlank, LineComment, LineDeleted:
			// transparent
		default:
			return i
		}
	}
	return -1
}

// makeInstr builds a Line for a synthesised instruction.
func makeInstr(op string, args ...string) *Line {
	raw := "    " + op + " " + strings.Join(args, ", ")
	return &Line{
		Kind: LineInstruction,
		Raw:  raw,
		Op:   op,
		Args: args,
	}
}

// optimize applies all peephole patterns in a fixed-point loop.
// The address map is rebuilt at the start of each iteration so that
// branch-range checks reflect prior deletions.
func optimize(lines []*Line) {
	for {
		changed := false
		am := buildAddrMap(lines)

		for i, l := range lines {
			if l.Kind != LineInstruction {
				continue
			}

			// --- Single-instruction patterns ---

			// mv rX, rX  →  delete
			if l.Op == "mv" && len(l.Args) == 2 && l.Args[0] == l.Args[1] {
				l.Kind = LineDeleted
				changed = true
				continue
			}

			// ldi rX, 0  →  mv rX, r0
			if l.Op == "ldi" && len(l.Args) == 2 && l.Args[1] == "0" {
				*l = *makeInstr("mv", l.Args[0], "r0")
				changed = true
				continue
			}

			// --- Branch-over-jal folding ---
			//
			// Pattern (skipping blank/comment/deleted between lines):
			//   br{cond} SKIP       ← line i
			//   jal      TARGET     ← next instruction j
			//   SKIP:               ← next non-trivial line k
			//
			// If TARGET is within BRX range, collapse to:
			//   br{inv(cond)} TARGET
			// and delete the jal.  The dead SKIP: label is left in place.
			//
			// Range is computed with post-optimization addresses:
			// if TARGET is after the jal, it moves 4 bytes closer when the
			// jal is deleted; if before, it is unchanged.
			if inv, ok := invertCond[l.Op]; ok && len(l.Args) == 1 {
				skipName := l.Args[0]
				j := nextInstr(lines, i+1)
				if j >= 0 && lines[j].Op == "jal" && len(lines[j].Args) == 1 {
					targetName := lines[j].Args[0]
					k := nextNonTrivial(lines, j+1)
					if k >= 0 && lines[k].Kind == LineLabel &&
						labelName(lines[k].Raw) == skipName &&
						strings.HasPrefix(targetName, "l_") {
						if targetAddr, ok := am.label[targetName]; ok {
							branchAddr := am.line[i]
							jalAddr := am.line[j]
							var offset int
							if targetAddr > jalAddr {
								// Forward: jal (4 bytes) will be deleted
								offset = (targetAddr - 4) - (branchAddr + 2)
							} else {
								// Backward: target address is unchanged
								offset = targetAddr - (branchAddr + 2)
							}
							if offset >= -512 && offset <= 511 {
								*l = *makeInstr(inv, targetName)
								lines[j].Kind = LineDeleted
								changed = true
								continue
							}
						}
					}
				}
			}

			// --- Standalone jal → br ---
			//
			// jal TARGET  →  br TARGET   when TARGET is within BRX range.
			// Saves 2 bytes per occurrence.  Only fires when the jal was not
			// already consumed by the branch-over-jal pattern above.
			//
			// Restricted to L_-prefixed targets: the code generator uses L_
			// for all intra-function control-flow labels.  Non-L_ labels are
			// function entry points; converting their jal to br would destroy
			// the link register update needed for the callee to return.
			if l.Op == "jal" && len(l.Args) == 1 &&
				strings.HasPrefix(l.Args[0], "l_") {
				targetName := l.Args[0]
				if targetAddr, ok := am.label[targetName]; ok {
					jalAddr := am.line[i]
					var offset int
					if targetAddr > jalAddr {
						// Forward: jal (4 bytes) → br (2 bytes), target shifts 2 bytes closer
						offset = (targetAddr - 2) - (jalAddr + 2)
					} else {
						// Backward: target address unchanged
						offset = targetAddr - (jalAddr + 2)
					}
					if offset >= -512 && offset <= 511 {
						*l = *makeInstr("br", targetName)
						changed = true
						continue
					}
				}
			}

			// --- Two-instruction patterns ---

			j := nextInstr(lines, i+1)
			if j < 0 {
				continue
			}
			m := lines[j]

			// stw rX, rB, N  then  ldw r?, rB, N
			if l.Op == "stw" && m.Op == "ldw" &&
				len(l.Args) == 3 && len(m.Args) == 3 &&
				l.Args[1] == m.Args[1] && l.Args[2] == m.Args[2] {

				stwSrc := l.Args[0] // rX stored
				ldwDst := m.Args[0] // rY loaded

				if ldwDst == stwSrc {
					// ldw rX, rB, N  →  delete (value already in rX)
					m.Kind = LineDeleted
					changed = true
				} else {
					// ldw rY, rB, N  →  mv rY, rX
					*m = *makeInstr("mv", ldwDst, stwSrc)
					changed = true
				}
				continue
			}

			// ldi rX, K  then  add rY, rZ, rX  or  add rY, rX, rZ
			// where K fits in [-64, 63]
			if l.Op == "ldi" && m.Op == "add" &&
				len(l.Args) == 2 && len(m.Args) == 3 {

				ldiDst := l.Args[0]
				kStr := l.Args[1]
				k, err := parseImm(kStr)
				if err == nil && k >= -64 && k <= 63 {
					addDst := m.Args[0]
					addS1 := m.Args[1]
					addS2 := m.Args[2]

					var otherSrc string
					if addS1 == ldiDst && addS2 != ldiDst {
						otherSrc = addS2
					} else if addS2 == ldiDst && addS1 != ldiDst {
						otherSrc = addS1
					}

					if otherSrc != "" {
						*m = *makeInstr("adi", addDst, otherSrc, kStr)
						l.Kind = LineDeleted
						changed = true
					}
				}
			}
		}
		if !changed {
			break
		}
	}
}

// parseImm parses a decimal or hex integer string.
func parseImm(s string) (int64, error) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		v, err := strconv.ParseInt(s[2:], 16, 64)
		return v, err
	}
	return strconv.ParseInt(s, 10, 64)
}
