// YAPL Code Generator - IR Parser
// Parses the text-based IR format from Pass 3

package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// IRParser parses IR text into IRProgram
type IRParser struct {
	scanner *bufio.Scanner
	lineNum int
	line    string
	prog    *IRProgram
}

// NewIRParser creates a new IR parser
func NewIRParser(r *bufio.Reader) *IRParser {
	return &IRParser{
		scanner: bufio.NewScanner(r),
		prog:    &IRProgram{},
	}
}

// Parse reads and parses the entire IR
func (p *IRParser) Parse() (*IRProgram, error) {
	for p.nextLine() {
		if p.line == "" {
			continue
		}

		// Handle directives
		if strings.HasPrefix(p.line, "#ir ") {
			// Version directive - ignore for now
			continue
		}
		if strings.HasPrefix(p.line, "#source ") {
			p.prog.SourceFile = strings.TrimPrefix(p.line, "#source ")
			continue
		}

		// Handle top-level declarations
		fields := p.tokenize(p.line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "ASM":
			// File-level inline assembly
			if len(fields) >= 2 {
				p.prog.AsmDecls = append(p.prog.AsmDecls, unquote(fields[1]))
			}
		case "STRUCT":
			s, err := p.parseStruct(fields)
			if err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
			p.prog.Structs = append(p.prog.Structs, s)
		case "CONST":
			c, err := p.parseConst(fields)
			if err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
			p.prog.Constants = append(p.prog.Constants, c)
		case "DATA":
			d, err := p.parseData(fields)
			if err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
			p.prog.Globals = append(p.prog.Globals, d)
		case "FUNC":
			f, err := p.parseFunction(fields)
			if err != nil {
				return nil, fmt.Errorf("line %d: %v", p.lineNum, err)
			}
			p.prog.Functions = append(p.prog.Functions, f)
		}
	}

	return p.prog, nil
}

func (p *IRParser) nextLine() bool {
	for p.scanner.Scan() {
		p.lineNum++
		p.line = strings.TrimSpace(p.scanner.Text())
		// Strip comments
		if idx := strings.Index(p.line, ";"); idx >= 0 {
			p.line = strings.TrimSpace(p.line[:idx])
		}
		return true
	}
	return false
}

// tokenize splits a line into tokens, handling quoted strings
func (p *IRParser) tokenize(line string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := false
	escape := false

	for _, ch := range line {
		if escape {
			current.WriteRune(ch)
			escape = false
			continue
		}
		if ch == '\\' && inQuote {
			current.WriteRune(ch)
			escape = true
			continue
		}
		if ch == '"' {
			current.WriteRune(ch)
			inQuote = !inQuote
			continue
		}
		if !inQuote && (ch == ' ' || ch == '\t' || ch == ',') {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteRune(ch)
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func (p *IRParser) parseStruct(fields []string) (*IRStruct, error) {
	// STRUCT name size align
	if len(fields) < 4 {
		return nil, fmt.Errorf("invalid STRUCT declaration")
	}
	s := &IRStruct{
		Name:  fields[1],
		Size:  parseInt(fields[2]),
		Align: parseInt(fields[3]),
	}

	// Read fields until ENDSTRUCT
	for p.nextLine() {
		if p.line == "" {
			continue
		}
		if p.line == "ENDSTRUCT" {
			break
		}
		ff := p.tokenize(p.line)
		if len(ff) >= 4 && ff[0] == "FIELD" {
			s.Fields = append(s.Fields, &IRField{
				Name:   ff[1],
				Offset: parseInt(ff[2]),
				Type:   ff[3],
			})
		}
	}
	return s, nil
}

func (p *IRParser) parseConst(fields []string) (*IRConst, error) {
	// CONST name visibility type value
	if len(fields) < 5 {
		return nil, fmt.Errorf("invalid CONST declaration")
	}
	return &IRConst{
		Name:       fields[1],
		Visibility: fields[2],
		Type:       fields[3],
		Value:      parseInt64(fields[4]),
	}, nil
}

func (p *IRParser) parseData(fields []string) (*IRData, error) {
	// DATA name visibility type size [init]
	// For BYTES/STRING/WORDS: DATA name visibility type count size [init]
	if len(fields) < 5 {
		return nil, fmt.Errorf("invalid DATA declaration")
	}
	d := &IRData{
		Name:       fields[1],
		Visibility: fields[2],
		Type:       fields[3],
	}

	// BYTES, STRING, and WORDS types have an extra count field
	typ := fields[3]
	if typ == "BYTES" || typ == "STRING" || typ == "WORDS" {
		// Format: DATA name visibility type count size [init]
		if len(fields) < 6 {
			return nil, fmt.Errorf("invalid %s DATA declaration: missing size", typ)
		}
		d.Size = parseInt(fields[5])
		if len(fields) >= 7 {
			d.Init = fields[6]
		}
	} else {
		// Format: DATA name visibility type size [init]
		d.Size = parseInt(fields[4])
		if len(fields) >= 6 {
			d.Init = fields[5]
		}
	}
	return d, nil
}

func (p *IRParser) parseFunction(fields []string) (*IRFunction, error) {
	// FUNC name
	if len(fields) < 2 {
		return nil, fmt.Errorf("invalid FUNC declaration")
	}
	f := &IRFunction{
		Name: fields[1],
	}

	// Read function metadata and instructions
	for p.nextLine() {
		if p.line == "" {
			continue
		}
		if p.line == "ENDFUNC" {
			break
		}

		ff := p.tokenize(p.line)
		if len(ff) == 0 {
			continue
		}

		switch ff[0] {
		case "VISIBILITY":
			if len(ff) >= 2 {
				f.Visibility = ff[1]
			}
		case "RETURN":
			if len(ff) >= 2 {
				f.ReturnType = ff[1]
			}
		case "PARAMS":
			// Just skip count, we'll read PARAM entries
		case "PARAM":
			if len(ff) >= 4 {
				f.Params = append(f.Params, &IRParam{
					Name:  ff[1],
					Type:  ff[2],
					Index: parseInt(ff[3]),
				})
			}
		case "LOCALS":
			// Just skip count
		case "LOCAL":
			if len(ff) >= 4 {
				f.Locals = append(f.Locals, &IRLocal{
					Name:   ff[1],
					Type:   ff[2],
					Offset: parseInt(ff[3]),
				})
			}
		case "FRAMESIZE":
			if len(ff) >= 2 {
				f.FrameSize = parseInt(ff[1])
			}
		default:
			// Parse instruction
			instr := p.parseInstruction(ff)
			if instr != nil {
				instr.LineNum = p.lineNum
				f.Instrs = append(f.Instrs, instr)
			}
		}
	}
	return f, nil
}

func (p *IRParser) parseInstruction(fields []string) *IRInstr {
	if len(fields) == 0 {
		return nil
	}

	// Check for label (ends with :)
	if strings.HasSuffix(fields[0], ":") {
		return &IRInstr{
			Op:    OpLabel,
			Label: strings.TrimSuffix(fields[0], ":"),
		}
	}

	// Check for assignment form: dest = OP args...
	if len(fields) >= 3 && fields[1] == "=" {
		return p.parseAssignInstr(fields)
	}

	// Non-assignment instructions
	return p.parseSimpleInstr(fields)
}

func (p *IRParser) parseAssignInstr(fields []string) *IRInstr {
	// dest = OP args...
	dest := fields[0]
	op := fields[2]

	instr := &IRInstr{
		Op:   op,
		Dest: dest,
	}

	// Handle CALL specially
	if op == "CALL" {
		// dest = CALL func, nargs
		if len(fields) >= 5 {
			instr.Args = []string{fields[3], fields[4]}
		}
		return instr
	}

	// Regular instruction with operands
	for i := 3; i < len(fields); i++ {
		instr.Args = append(instr.Args, fields[i])
	}

	return instr
}

func (p *IRParser) parseSimpleInstr(fields []string) *IRInstr {
	op := fields[0]
	instr := &IRInstr{Op: op}

	switch op {
	case "JUMP":
		// JUMP label
		if len(fields) >= 2 {
			instr.Target = fields[1]
		}
	case "JUMPZ", "JUMPNZ":
		// JUMPZ cond, label / JUMPNZ cond, label
		if len(fields) >= 3 {
			instr.Args = []string{fields[1]}
			instr.Target = fields[2]
		}
	case "RETURN":
		// RETURN [value]
		if len(fields) >= 2 {
			instr.Args = []string{fields[1]}
		}
	case "CALL":
		// CALL func, nargs (void return)
		if len(fields) >= 3 {
			instr.Args = []string{fields[1], fields[2]}
		}
	case "ARG":
		// ARG n, value
		if len(fields) >= 3 {
			instr.Args = []string{fields[1], fields[2]}
		}
	case "STORE.W", "STORE.B":
		// STORE.W [addr], value
		if len(fields) >= 3 {
			instr.Args = []string{fields[1], fields[2]}
		}
	case "SETPARAM":
		// SETPARAM n, value
		if len(fields) >= 3 {
			instr.Args = []string{fields[1], fields[2]}
		}
	case "ASM":
		// ASM "text"
		if len(fields) >= 2 {
			instr.Args = []string{unquote(fields[1])}
		}
	default:
		// Other instructions
		for i := 1; i < len(fields); i++ {
			instr.Args = append(instr.Args, fields[i])
		}
	}

	return instr
}

func parseInt(s string) int {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		v, _ := strconv.ParseInt(s[2:], 16, 32)
		return int(v)
	}
	v, _ := strconv.ParseInt(s, 0, 32)
	return int(v)
}

func parseInt64(s string) int64 {
	v, _ := strconv.ParseInt(s, 0, 64)
	return v
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
