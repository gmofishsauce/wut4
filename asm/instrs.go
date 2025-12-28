package main

/* Instruction table - base instructions */
var baseInstrs = []InstrDef{
	{"ldw", 0x0000, FMT_BASE, 3, 1, 7},
	{"ldb", 0x2000, FMT_BASE, 3, 1, 7},
	{"stw", 0x4000, FMT_BASE, 3, 1, 7},
	{"stb", 0x6000, FMT_BASE, 3, 1, 7},
	{"adi", 0x8000, FMT_BASE, 3, 1, 7},
	{"lui", 0xA000, FMT_LUI, 2, 1, 10},
	{"jal", 0xE000, FMT_JAL, 3, 1, 6},
}

/* XOP instructions (3-operand) */
var xopInstrs = []InstrDef{
	{"sbb", 0xF000, FMT_XOP, 3, 0, 0}, /* opcode | (0<<9) */
	{"adc", 0xF200, FMT_XOP, 3, 0, 0}, /* opcode | (1<<9) */
	{"sub", 0xF400, FMT_XOP, 3, 0, 0}, /* opcode | (2<<9) */
	{"add", 0xF600, FMT_XOP, 3, 0, 0}, /* opcode | (3<<9) */
	{"xor", 0xF800, FMT_XOP, 3, 0, 0}, /* opcode | (4<<9) */
	{"or", 0xFA00, FMT_XOP, 3, 0, 0},  /* opcode | (5<<9) */
	{"and", 0xFC00, FMT_XOP, 3, 0, 0}, /* opcode | (6<<9) */
}

/* YOP instructions (2-operand) */
var yopInstrs = []InstrDef{
	{"lsp", 0xFE00, FMT_YOP, 2, 0, 0}, /* opcode | (0<<6) */
	{"lsi", 0xFE40, FMT_YOP, 2, 0, 0}, /* opcode | (1<<6) */
	{"ssp", 0xFE80, FMT_YOP, 2, 0, 0}, /* opcode | (2<<6) */
	{"ssi", 0xFEC0, FMT_YOP, 2, 0, 0}, /* opcode | (3<<6) */
	{"lcw", 0xFF00, FMT_YOP, 2, 0, 0}, /* opcode | (4<<6) */
	{"sys", 0xFF40, FMT_YOP, 2, 0, 0}, /* opcode | (5<<6) - special handling */
	{"tst", 0xFF80, FMT_YOP, 2, 0, 0}, /* opcode | (6<<6) */
}

/* ZOP instructions (1-operand) */
var zopInstrs = []InstrDef{
	{"not", 0xFFC0, FMT_ZOP, 1, 0, 0}, /* opcode | (0<<3) */
	{"neg", 0xFFC8, FMT_ZOP, 1, 0, 0}, /* opcode | (1<<3) */
	{"tst", 0xFFD0, FMT_ZOP, 1, 0, 0}, /* opcode | (2<<3) - note: TST is both YOP and ZOP */
	{"sxt", 0xFFD8, FMT_ZOP, 1, 0, 0}, /* opcode | (3<<3) */
	{"sra", 0xFFE0, FMT_ZOP, 1, 0, 0}, /* opcode | (4<<3) */
	{"srl", 0xFFE8, FMT_ZOP, 1, 0, 0}, /* opcode | (5<<3) */
	{"ji", 0xFFF0, FMT_ZOP, 1, 0, 0},  /* opcode | (6<<3) */
}

/* VOP instructions (0-operand) */
var vopInstrs = []InstrDef{
	{"ccf", 0xFFF8, FMT_VOP, 0, 0, 0}, /* opcode | 0 */
	{"scf", 0xFFF9, FMT_VOP, 0, 0, 0}, /* opcode | 1 */
	{"di", 0xFFFA, FMT_VOP, 0, 0, 0},  /* opcode | 2 */
	{"ei", 0xFFFB, FMT_VOP, 0, 0, 0},  /* opcode | 3 */
	{"hlt", 0xFFFC, FMT_VOP, 0, 0, 0}, /* opcode | 4 */
	{"brk", 0xFFFD, FMT_VOP, 0, 0, 0}, /* opcode | 5 */
	{"rti", 0xFFFE, FMT_VOP, 0, 0, 0}, /* opcode | 6 */
	{"die", 0xFFFF, FMT_VOP, 0, 0, 0}, /* opcode | 7 */
}

/* Look up an instruction by name */
func lookupInstr(name string) *InstrDef {
	/* Check base instructions */
	for i := 0; i < len(baseInstrs); i++ {
		if baseInstrs[i].name == name {
			return &baseInstrs[i]
		}
	}

	/* Check XOPs */
	for i := 0; i < len(xopInstrs); i++ {
		if xopInstrs[i].name == name {
			return &xopInstrs[i]
		}
	}

	/* Check YOPs */
	for i := 0; i < len(yopInstrs); i++ {
		if yopInstrs[i].name == name {
			return &yopInstrs[i]
		}
	}

	/* Check ZOPs */
	for i := 0; i < len(zopInstrs); i++ {
		if zopInstrs[i].name == name {
			return &zopInstrs[i]
		}
	}

	/* Check VOPs */
	for i := 0; i < len(vopInstrs); i++ {
		if vopInstrs[i].name == name {
			return &vopInstrs[i]
		}
	}

	return nil
}

/* Check if a string is a register name, return register number or -1 */
func parseRegister(s string) int {
	if val, ok := regNames[s]; ok {
		return val
	}
	return -1
}

/* Sign-extend a value to N bits */
func signExtend(val int, bits int) int {
	mask := (1 << bits) - 1
	val = val & mask
	/* Check sign bit */
	if (val & (1 << (bits - 1))) != 0 {
		/* Negative - extend sign */
		val = val | ^mask
	}
	return val
}

/* Check if value fits in N bits (signed) */
func fitsInSigned(val int, bits int) int {
	min := -(1 << (bits - 1))
	max := (1 << (bits - 1)) - 1
	if val >= min && val <= max {
		return 1
	}
	return 0
}

/* Check if value fits in N bits (unsigned) */
func fitsInUnsigned(val int, bits int) int {
	max := (1 << bits) - 1
	if val >= 0 && val <= max {
		return 1
	}
	return 0
}
