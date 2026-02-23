package main

/* Magic numbers */
const (
	MAGIC_WOF = 0xDDD2 /* WUT-4 Object Format */
	MAGIC_EXE = 0xDDD1 /* WUT-4 Executable */
)

/* WOF section identifiers */
const (
	SEC_UNDEF    = 0
	SEC_CODE_WOF = 1
	SEC_DATA_WOF = 2
)

/* WOF symbol visibility */
const (
	VIS_LOCAL  = 0
	VIS_GLOBAL = 1
)

/* Relocation types */
const (
	R_LDI_CODE    = 0x01
	R_LDI_DATA    = 0x02
	R_JAL         = 0x03
	R_WORD16_CODE = 0x04
	R_WORD16_DATA = 0x05
)

/* WOFHeader represents the 16-byte WOF file header */
type WOFHeader struct {
	Magic          uint16
	Version        uint8
	Flags          uint8
	CodeSize       uint16
	DataSize       uint16
	SymCount       uint16
	RelocCount     uint16
	StringTableSize uint16
	Reserved       uint16
}

/* WOFSymbol is one entry in the symbol table (8 bytes on disk) */
type WOFSymbol struct {
	NameOffset uint16
	Value      uint16
	Section    uint8
	Visibility uint8
	Reserved   uint16
	Name       string /* decoded from string table */
}

/* WOFReloc is one entry in the relocation table (8 bytes on disk) */
type WOFReloc struct {
	Section  uint8  /* 0=code, 1=data */
	Type     uint8
	Offset   uint16
	SymIndex uint16
	Reserved uint16
}

/* ObjectFile holds all data from a parsed .wo file */
type ObjectFile struct {
	path        string
	header      WOFHeader
	code        []byte
	data        []byte
	symbols     []WOFSymbol
	relocations []WOFReloc
	/* codeOffset and dataOffset are assigned by the linker during layout */
	codeOffset  uint16
	dataOffset  uint16
}

/* ResolvedSym is a globally-visible symbol after resolution */
type ResolvedSym struct {
	name     string
	value    uint16   /* section-relative offset */
	section  uint8    /* SEC_CODE_WOF or SEC_DATA_WOF */
	objIndex int      /* which object file defines it */
}
