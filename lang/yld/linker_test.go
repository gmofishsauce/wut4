package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// ---- WOF construction helper -----------------------------------------------

// wofBuilder assembles a WOF byte slice for use in tests.
// Populate the exported fields, then call bytes() to get the wire format.
// NameOffset in WOFSymbol is computed automatically; set Name instead.
type wofBuilder struct {
	code    []byte
	data    []byte
	symbols []WOFSymbol // Name field used; NameOffset computed by builder
	relocs  []WOFReloc
}

func (b *wofBuilder) build() []byte {
	// Build string table: first byte is always \0 (null / unnamed).
	strtab := []byte{0}
	nameOffsets := make([]uint16, len(b.symbols))
	for i, sym := range b.symbols {
		off := uint16(len(strtab))
		nameOffsets[i] = off
		strtab = append(strtab, []byte(sym.Name)...)
		strtab = append(strtab, 0)
	}

	var buf []byte

	// Header (16 bytes).
	h := make([]byte, 16)
	binary.LittleEndian.PutUint16(h[0:2], MAGIC_WOF)
	h[2] = 1 // version
	binary.LittleEndian.PutUint16(h[4:6], uint16(len(b.code)))
	binary.LittleEndian.PutUint16(h[6:8], uint16(len(b.data)))
	binary.LittleEndian.PutUint16(h[8:10], uint16(len(b.symbols)))
	binary.LittleEndian.PutUint16(h[10:12], uint16(len(b.relocs)))
	binary.LittleEndian.PutUint16(h[12:14], uint16(len(strtab)))
	buf = append(buf, h...)

	// Sections.
	buf = append(buf, b.code...)
	buf = append(buf, b.data...)

	// Symbol table: 8 bytes per entry.
	for i, sym := range b.symbols {
		entry := make([]byte, 8)
		binary.LittleEndian.PutUint16(entry[0:2], nameOffsets[i])
		binary.LittleEndian.PutUint16(entry[2:4], sym.Value)
		entry[4] = sym.Section
		entry[5] = sym.Visibility
		buf = append(buf, entry...)
	}

	// Relocation table: 8 bytes per entry.
	for _, r := range b.relocs {
		entry := make([]byte, 8)
		entry[0] = r.Section
		entry[1] = r.Type
		binary.LittleEndian.PutUint16(entry[2:4], r.Offset)
		binary.LittleEndian.PutUint16(entry[4:6], r.SymIndex)
		buf = append(buf, entry...)
	}

	// String table.
	buf = append(buf, strtab...)
	return buf
}

// writeTempWOF writes a WOF file to a temporary directory and returns its path.
func writeTempWOF(t *testing.T, b *wofBuilder) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.wo")
	if err := os.WriteFile(path, b.build(), 0644); err != nil {
		t.Fatalf("writeTempWOF: %v", err)
	}
	return path
}

// u16le reads a little-endian uint16 at the given offset.
func u16le(b []byte, off int) uint16 {
	return binary.LittleEndian.Uint16(b[off : off+2])
}

// ---- readObjectFile tests ---------------------------------------------------

func TestReadObjectFile_Minimal(t *testing.T) {
	// Empty object: no code, data, symbols, or relocations.
	path := writeTempWOF(t, &wofBuilder{})
	obj, err := readObjectFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj.header.Magic != MAGIC_WOF {
		t.Errorf("magic: got 0x%04X, want 0x%04X", obj.header.Magic, MAGIC_WOF)
	}
	if len(obj.code) != 0 {
		t.Errorf("expected empty code section, got %d bytes", len(obj.code))
	}
	if len(obj.data) != 0 {
		t.Errorf("expected empty data section, got %d bytes", len(obj.data))
	}
	if len(obj.symbols) != 0 {
		t.Errorf("expected no symbols, got %d", len(obj.symbols))
	}
	if len(obj.relocations) != 0 {
		t.Errorf("expected no relocations, got %d", len(obj.relocations))
	}
}

func TestReadObjectFile_BadMagic(t *testing.T) {
	raw := (&wofBuilder{}).build()
	binary.LittleEndian.PutUint16(raw[0:2], 0x1234) // corrupt magic
	path := filepath.Join(t.TempDir(), "bad.wo")
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readObjectFile(path); err == nil {
		t.Error("expected error for bad magic, got nil")
	}
}

func TestReadObjectFile_TooShort(t *testing.T) {
	// Only 3 bytes — well short of the 16-byte header minimum.
	path := filepath.Join(t.TempDir(), "short.wo")
	if err := os.WriteFile(path, []byte{0xD2, 0xDD, 0x01}, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := readObjectFile(path); err == nil {
		t.Error("expected error for truncated file, got nil")
	}
}

func TestReadObjectFile_WithCode(t *testing.T) {
	code := []byte{0x00, 0xA0, 0x09, 0x80} // two instruction words
	path := writeTempWOF(t, &wofBuilder{code: code})
	obj, err := readObjectFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(obj.code, code) {
		t.Errorf("code mismatch: got %v, want %v", obj.code, code)
	}
}

func TestReadObjectFile_WithSymbol(t *testing.T) {
	path := writeTempWOF(t, &wofBuilder{
		code: []byte{0x00, 0x60}, // 2-byte code
		symbols: []WOFSymbol{
			{Name: "Foo", Section: SEC_CODE_WOF, Value: 0, Visibility: VIS_GLOBAL},
		},
	})
	obj, err := readObjectFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(obj.symbols) != 1 {
		t.Fatalf("expected 1 symbol, got %d", len(obj.symbols))
	}
	sym := obj.symbols[0]
	if sym.Name != "Foo" {
		t.Errorf("Name: got %q, want %q", sym.Name, "Foo")
	}
	if sym.Section != SEC_CODE_WOF {
		t.Errorf("Section: got %d, want %d", sym.Section, SEC_CODE_WOF)
	}
	if sym.Visibility != VIS_GLOBAL {
		t.Errorf("Visibility: got %d, want %d", sym.Visibility, VIS_GLOBAL)
	}
}

func TestReadObjectFile_WithRelocation(t *testing.T) {
	code := make([]byte, 4) // placeholder LUI+JAL
	binary.LittleEndian.PutUint16(code[0:], 0xA000)
	binary.LittleEndian.PutUint16(code[2:], 0xE000)

	path := writeTempWOF(t, &wofBuilder{
		code: code,
		symbols: []WOFSymbol{
			{Name: "Bar", Section: SEC_UNDEF, Value: 0, Visibility: VIS_GLOBAL},
		},
		relocs: []WOFReloc{
			{Section: 0, Type: R_JAL, Offset: 0, SymIndex: 0},
		},
	})

	obj, err := readObjectFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(obj.relocations) != 1 {
		t.Fatalf("expected 1 relocation, got %d", len(obj.relocations))
	}
	r := obj.relocations[0]
	if r.Type != R_JAL {
		t.Errorf("reloc type: got 0x%02X, want 0x%02X", r.Type, R_JAL)
	}
	if r.Offset != 0 {
		t.Errorf("reloc offset: got %d, want 0", r.Offset)
	}
	if r.SymIndex != 0 {
		t.Errorf("reloc symIndex: got %d, want 0", r.SymIndex)
	}
}

// ---- linker phase tests ----------------------------------------------------

func TestLink_SingleObjectNoCode(t *testing.T) {
	path := writeTempWOF(t, &wofBuilder{})
	obj, err := readObjectFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ld := newLinker(false)
	ld.addObject(obj)
	mergedCode, mergedData, err := ld.link()
	if err != nil {
		t.Fatalf("link error: %v", err)
	}
	if len(mergedCode) != 0 {
		t.Errorf("expected empty code, got %d bytes", len(mergedCode))
	}
	if len(mergedData) != 0 {
		t.Errorf("expected empty data, got %d bytes", len(mergedData))
	}
}

func TestLink_SingleObjectWithCode(t *testing.T) {
	code := []byte{0x00, 0x60, 0x00, 0x00} // 4 bytes, even
	path := writeTempWOF(t, &wofBuilder{code: code})
	obj, err := readObjectFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ld := newLinker(false)
	ld.addObject(obj)
	mergedCode, _, err := ld.link()
	if err != nil {
		t.Fatalf("link error: %v", err)
	}
	if !bytes.Equal(mergedCode, code) {
		t.Errorf("code mismatch: got %v, want %v", mergedCode, code)
	}
}

func TestLink_TwoObjects_Alignment(t *testing.T) {
	// Object A has 3 bytes (odd); linker must pad to 4 before Object B.
	codeA := []byte{0x01, 0x02, 0x03}
	codeB := []byte{0x04, 0x05}
	dir := t.TempDir()
	for _, tc := range []struct {
		name string
		code []byte
	}{
		{"a.wo", codeA},
		{"b.wo", codeB},
	} {
		b := &wofBuilder{code: tc.code}
		if err := os.WriteFile(filepath.Join(dir, tc.name), b.build(), 0644); err != nil {
			t.Fatal(err)
		}
	}

	ld := newLinker(false)
	for _, name := range []string{"a.wo", "b.wo"} {
		obj, err := readObjectFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		ld.addObject(obj)
	}

	mergedCode, _, err := ld.link()
	if err != nil {
		t.Fatalf("link error: %v", err)
	}

	// Expect: [A[0:3], 0x00 pad, B[0:2]] = 6 bytes total.
	if len(mergedCode) != 6 {
		t.Errorf("merged code length: got %d, want 6", len(mergedCode))
	}
	if !bytes.Equal(mergedCode[0:3], codeA) {
		t.Errorf("object A code mismatch at [0:3]: got %v", mergedCode[0:3])
	}
	if mergedCode[3] != 0 {
		t.Errorf("padding byte at [3]: got %d, want 0", mergedCode[3])
	}
	if !bytes.Equal(mergedCode[4:6], codeB) {
		t.Errorf("object B code mismatch at [4:6]: got %v", mergedCode[4:6])
	}
	if ld.objects[0].codeOffset != 0 {
		t.Errorf("A.codeOffset: got %d, want 0", ld.objects[0].codeOffset)
	}
	if ld.objects[1].codeOffset != 4 {
		t.Errorf("B.codeOffset: got %d, want 4", ld.objects[1].codeOffset)
	}
}

// TestLink_SymbolResolution_JAL verifies inter-object R_JAL relocation.
// Object A calls Bar via a LUI+JAL placeholder; Object B defines Bar.
func TestLink_SymbolResolution_JAL(t *testing.T) {
	// Object A code: LUI r0, 0 (0xA000) + JAL r0, r0, 0 (0xE000) — 4 bytes.
	codeA := make([]byte, 4)
	binary.LittleEndian.PutUint16(codeA[0:], 0xA000)
	binary.LittleEndian.PutUint16(codeA[2:], 0xE000)

	// Object B code: a single 2-byte instruction (the body of Bar).
	codeB := []byte{0x00, 0x60}

	dir := t.TempDir()

	// Write A with undefined Bar + R_JAL relocation.
	aWOF := &wofBuilder{
		code: codeA,
		symbols: []WOFSymbol{
			{Name: "Bar", Section: SEC_UNDEF, Value: 0, Visibility: VIS_GLOBAL},
		},
		relocs: []WOFReloc{
			{Section: 0, Type: R_JAL, Offset: 0, SymIndex: 0},
		},
	}
	pathA := filepath.Join(dir, "a.wo")
	if err := os.WriteFile(pathA, aWOF.build(), 0644); err != nil {
		t.Fatal(err)
	}

	// Write B exporting Bar.
	bWOF := &wofBuilder{
		code: codeB,
		symbols: []WOFSymbol{
			{Name: "Bar", Section: SEC_CODE_WOF, Value: 0, Visibility: VIS_GLOBAL},
		},
	}
	pathB := filepath.Join(dir, "b.wo")
	if err := os.WriteFile(pathB, bWOF.build(), 0644); err != nil {
		t.Fatal(err)
	}

	ld := newLinker(false)
	for _, p := range []string{pathA, pathB} {
		obj, err := readObjectFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		ld.addObject(obj)
	}

	mergedCode, _, err := ld.link()
	if err != nil {
		t.Fatalf("link error: %v", err)
	}

	// Layout: A at [0:4], B at [4:6] (A is already even).
	// finalAddr(Bar) = B.codeOffset + 0 = 4.
	// Patch LUI+JAL at offset 0 with addr=4:
	//   upper = (4 >> 6) = 0, lower = 4 & 0x3F = 4, rT = 0, rS = 0
	//   newWord1 = 0xA000 | (0<<3) | 0 = 0xA000
	//   newWord2 = 0xE000 | (4<<6) | (0<<3) | 0 = 0xE100
	want1 := uint16(0xA000)
	want2 := uint16(0xE100)
	got1 := u16le(mergedCode, 0)
	got2 := u16le(mergedCode, 2)
	if got1 != want1 {
		t.Errorf("patched word1: got 0x%04X, want 0x%04X", got1, want1)
	}
	if got2 != want2 {
		t.Errorf("patched word2: got 0x%04X, want 0x%04X", got2, want2)
	}
	if !bytes.Equal(mergedCode[4:6], codeB) {
		t.Errorf("object B code not preserved: got %v", mergedCode[4:6])
	}
}

// TestLink_SymbolResolution_LDI_Data verifies R_LDI_DATA relocation.
// Object A loads the address of a data symbol from Object B using ldi.
func TestLink_SymbolResolution_LDI_Data(t *testing.T) {
	// Object A code: LUI r1, 0 (0xA001) + ADI r1, r1, 0 (0x8009) — 4 bytes.
	codeA := make([]byte, 4)
	binary.LittleEndian.PutUint16(codeA[0:], 0xA001) // LUI r1, 0
	binary.LittleEndian.PutUint16(codeA[2:], 0x8009) // ADI r1, r1, 0

	// Object A also has 2 bytes of data, so B's data will be at offset 2.
	dataA := []byte{0xAA, 0xBB}

	// Object B has 2 bytes of data; Global is the first word.
	dataB := []byte{0xCC, 0xDD}

	dir := t.TempDir()

	aWOF := &wofBuilder{
		code: codeA,
		data: dataA,
		symbols: []WOFSymbol{
			{Name: "Global", Section: SEC_UNDEF, Value: 0, Visibility: VIS_GLOBAL},
		},
		relocs: []WOFReloc{
			{Section: 0, Type: R_LDI_DATA, Offset: 0, SymIndex: 0},
		},
	}
	pathA := filepath.Join(dir, "a.wo")
	if err := os.WriteFile(pathA, aWOF.build(), 0644); err != nil {
		t.Fatal(err)
	}

	bWOF := &wofBuilder{
		data: dataB,
		symbols: []WOFSymbol{
			{Name: "Global", Section: SEC_DATA_WOF, Value: 0, Visibility: VIS_GLOBAL},
		},
	}
	pathB := filepath.Join(dir, "b.wo")
	if err := os.WriteFile(pathB, bWOF.build(), 0644); err != nil {
		t.Fatal(err)
	}

	ld := newLinker(false)
	for _, p := range []string{pathA, pathB} {
		obj, err := readObjectFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		ld.addObject(obj)
	}

	mergedCode, mergedData, err := ld.link()
	if err != nil {
		t.Fatalf("link error: %v", err)
	}

	// Layout: A.data at [0:2], B.data at [2:4].
	// finalAddr(Global) = B.dataOffset + 0 = 2.
	// Patch LUI+ADI at code offset 0 with addr=2:
	//   rT = 0xA001 & 0x7 = 1
	//   upper = (2 >> 6) = 0, lower = 2 & 0x3F = 2
	//   newWord1 = 0xA000 | 0 | 1 = 0xA001  (unchanged)
	//   newWord2 = 0x8000 | (2<<6) | (1<<3) | 1 = 0x8089
	want1 := uint16(0xA001)
	want2 := uint16(0x8089)
	got1 := u16le(mergedCode, 0)
	got2 := u16le(mergedCode, 2)
	if got1 != want1 {
		t.Errorf("patched word1: got 0x%04X, want 0x%04X", got1, want1)
	}
	if got2 != want2 {
		t.Errorf("patched word2: got 0x%04X, want 0x%04X", got2, want2)
	}

	// Merged data should be A's data followed by B's data.
	wantData := []byte{0xAA, 0xBB, 0xCC, 0xDD}
	if !bytes.Equal(mergedData, wantData) {
		t.Errorf("mergedData: got %v, want %v", mergedData, wantData)
	}
}

// TestLink_SymbolResolution_WORD16_Code verifies R_WORD16_CODE relocation.
// Object A has a function-pointer word in its data section that must be
// patched with the final code address of Func from Object B.
func TestLink_SymbolResolution_WORD16_Code(t *testing.T) {
	codeA := []byte{0x00, 0x60}        // 2-byte instruction
	dataA := []byte{0x00, 0x00}        // placeholder word — will hold Func's address
	codeB := []byte{0x00, 0x60}        // body of Func

	dir := t.TempDir()

	aWOF := &wofBuilder{
		code: codeA,
		data: dataA,
		symbols: []WOFSymbol{
			{Name: "Func", Section: SEC_UNDEF, Value: 0, Visibility: VIS_GLOBAL},
		},
		relocs: []WOFReloc{
			// Section=1 (data section of A), patch offset 0 with Func's final addr.
			{Section: 1, Type: R_WORD16_CODE, Offset: 0, SymIndex: 0},
		},
	}
	pathA := filepath.Join(dir, "a.wo")
	if err := os.WriteFile(pathA, aWOF.build(), 0644); err != nil {
		t.Fatal(err)
	}

	bWOF := &wofBuilder{
		code: codeB,
		symbols: []WOFSymbol{
			{Name: "Func", Section: SEC_CODE_WOF, Value: 0, Visibility: VIS_GLOBAL},
		},
	}
	pathB := filepath.Join(dir, "b.wo")
	if err := os.WriteFile(pathB, bWOF.build(), 0644); err != nil {
		t.Fatal(err)
	}

	ld := newLinker(false)
	for _, p := range []string{pathA, pathB} {
		obj, err := readObjectFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		ld.addObject(obj)
	}

	_, mergedData, err := ld.link()
	if err != nil {
		t.Fatalf("link error: %v", err)
	}

	// A.code at [0:2], B.code at [2:4]; finalAddr(Func) = 2.
	// mergedData[0:2] should be 2 (little-endian).
	if len(mergedData) < 2 {
		t.Fatalf("mergedData too short: %d bytes", len(mergedData))
	}
	got := u16le(mergedData, 0)
	if got != 2 {
		t.Errorf("patched data word: got %d, want 2", got)
	}
}

// TestLink_IntraFileRelocation tests a JAL relocation where both the
// call site and the target are in the same object file.
func TestLink_IntraFileRelocation_JAL(t *testing.T) {
	// Code layout: [LUI+JAL placeholder at 0:4] [target instruction at 4:6].
	code := make([]byte, 6)
	binary.LittleEndian.PutUint16(code[0:], 0xA000) // LUI r0, 0
	binary.LittleEndian.PutUint16(code[2:], 0xE000) // JAL r0, r0, 0
	binary.LittleEndian.PutUint16(code[4:], 0x6000) // HLT (target of the call)

	path := writeTempWOF(t, &wofBuilder{
		code: code,
		symbols: []WOFSymbol{
			// Local symbol at code offset 4 (same file).
			{Name: "local", Section: SEC_CODE_WOF, Value: 4, Visibility: VIS_LOCAL},
		},
		relocs: []WOFReloc{
			{Section: 0, Type: R_JAL, Offset: 0, SymIndex: 0},
		},
	})

	obj, err := readObjectFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ld := newLinker(false)
	ld.addObject(obj)
	mergedCode, _, err := ld.link()
	if err != nil {
		t.Fatalf("link error: %v", err)
	}

	// Single object at codeOffset=0; finalAddr("local") = 0 + 4 = 4.
	// Patch at offset 0 with addr=4:
	//   rT=0, rS=0, upper=0, lower=4
	//   newWord1 = 0xA000, newWord2 = 0xE100
	got1 := u16le(mergedCode, 0)
	got2 := u16le(mergedCode, 2)
	if got1 != 0xA000 {
		t.Errorf("word1: got 0x%04X, want 0xA000", got1)
	}
	if got2 != 0xE100 {
		t.Errorf("word2: got 0x%04X, want 0xE100", got2)
	}
	// Target instruction should be unchanged.
	if u16le(mergedCode, 4) != 0x6000 {
		t.Errorf("target instruction clobbered: got 0x%04X", u16le(mergedCode, 4))
	}
}

// ---- error-path tests -------------------------------------------------------

func TestLink_UndefinedSymbol(t *testing.T) {
	code := make([]byte, 4)
	binary.LittleEndian.PutUint16(code[0:], 0xA000)
	binary.LittleEndian.PutUint16(code[2:], 0xE000)

	path := writeTempWOF(t, &wofBuilder{
		code: code,
		symbols: []WOFSymbol{
			{Name: "Missing", Section: SEC_UNDEF, Value: 0, Visibility: VIS_GLOBAL},
		},
		relocs: []WOFReloc{
			{Section: 0, Type: R_JAL, Offset: 0, SymIndex: 0},
		},
	})
	obj, err := readObjectFile(path)
	if err != nil {
		t.Fatal(err)
	}
	ld := newLinker(false)
	ld.addObject(obj)
	if _, _, err = ld.link(); err == nil {
		t.Error("expected error for undefined symbol, got nil")
	}
}

func TestLink_DuplicateGlobal(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.wo", "b.wo"} {
		b := &wofBuilder{
			code: []byte{0x00, 0x60},
			symbols: []WOFSymbol{
				{Name: "Foo", Section: SEC_CODE_WOF, Value: 0, Visibility: VIS_GLOBAL},
			},
		}
		if err := os.WriteFile(filepath.Join(dir, name), b.build(), 0644); err != nil {
			t.Fatal(err)
		}
	}
	ld := newLinker(false)
	for _, name := range []string{"a.wo", "b.wo"} {
		obj, err := readObjectFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		ld.addObject(obj)
	}
	if _, _, err := ld.link(); err == nil {
		t.Error("expected error for duplicate global, got nil")
	}
}

// ---- patch-function unit tests ----------------------------------------------

func TestPatchLUIPlusADI_Basic(t *testing.T) {
	// LUI r2, 0 + ADI r2, r2, 0 — rT=2.
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:], 0xA002)
	binary.LittleEndian.PutUint16(buf[2:], 0x8012)

	// Patch with addr = 0x0140 (= 320 decimal).
	//   upper = 320 >> 6 = 5,  lower = 320 & 0x3F = 0
	//   newWord1 = 0xA000 | (5<<3) | 2 = 0xA02A
	//   newWord2 = 0x8000 | (0<<6)  | (2<<3) | 2 = 0x8012  (unchanged, lower=0)
	if err := patchLUIPlusADI(buf, 0, 0x0140); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want1, want2 := uint16(0xA02A), uint16(0x8012)
	got1, got2 := u16le(buf, 0), u16le(buf, 2)
	if got1 != want1 {
		t.Errorf("word1: got 0x%04X, want 0x%04X", got1, want1)
	}
	if got2 != want2 {
		t.Errorf("word2: got 0x%04X, want 0x%04X", got2, want2)
	}
}

func TestPatchLUIPlusADI_NonZeroLower(t *testing.T) {
	// Use addr = 0x0083 (= 131): upper=2, lower=3.
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:], 0xA000) // rT = 0
	binary.LittleEndian.PutUint16(buf[2:], 0x8000)

	if err := patchLUIPlusADI(buf, 0, 0x0083); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// upper = 0x83 >> 6 = 2, lower = 0x83 & 0x3F = 3, rT = 0
	// word1 = 0xA000 | (2<<3) = 0xA010
	// word2 = 0x8000 | (3<<6) = 0x80C0
	want1, want2 := uint16(0xA010), uint16(0x80C0)
	got1, got2 := u16le(buf, 0), u16le(buf, 2)
	if got1 != want1 {
		t.Errorf("word1: got 0x%04X, want 0x%04X", got1, want1)
	}
	if got2 != want2 {
		t.Errorf("word2: got 0x%04X, want 0x%04X", got2, want2)
	}
}

func TestPatchLUIPlusADI_PreservesReg(t *testing.T) {
	// Verify rT (bits [2:0] of word1) is preserved for all 8 registers.
	for rT := 0; rT < 8; rT++ {
		buf := make([]byte, 4)
		binary.LittleEndian.PutUint16(buf[0:], uint16(0xA000)|uint16(rT))
		binary.LittleEndian.PutUint16(buf[2:], uint16(0x8000)|uint16(rT<<3)|uint16(rT))

		if err := patchLUIPlusADI(buf, 0, 0x0040); err != nil {
			t.Fatalf("rT=%d: unexpected error: %v", rT, err)
		}
		// 0x0040: upper=1, lower=0.
		gotReg := int(u16le(buf, 0) & 0x7)
		if gotReg != rT {
			t.Errorf("rT=%d: rT not preserved after patch, got rT=%d", rT, gotReg)
		}
	}
}

func TestPatchLUIPlusADI_MaxAddress(t *testing.T) {
	// addr = 0xFFFF: upper=0x3FF (10 bits all set), lower=0x3F (6 bits all set).
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:], 0xA000) // rT=0
	binary.LittleEndian.PutUint16(buf[2:], 0x8000)

	if err := patchLUIPlusADI(buf, 0, 0xFFFF); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// word1 = 0xA000 | (0x3FF<<3) | 0 = 0xA000 | 0x1FF8 = 0xBFF8
	// word2 = 0x8000 | (0x3F<<6)  | 0 | 0 = 0x8000 | 0x0FC0 = 0x8FC0
	want1, want2 := uint16(0xBFF8), uint16(0x8FC0)
	got1, got2 := u16le(buf, 0), u16le(buf, 2)
	if got1 != want1 {
		t.Errorf("word1: got 0x%04X, want 0x%04X", got1, want1)
	}
	if got2 != want2 {
		t.Errorf("word2: got 0x%04X, want 0x%04X", got2, want2)
	}
}

func TestPatchLUIPlusADI_OutOfBounds(t *testing.T) {
	buf := make([]byte, 3) // too short for a 4-byte patch
	if err := patchLUIPlusADI(buf, 0, 0x1234); err == nil {
		t.Error("expected error for out-of-bounds patch, got nil")
	}
}

func TestPatchLUIPlusJAL_Basic(t *testing.T) {
	// LUI r3, 0 + JAL r3, r5, 0 — rT=3, rS=5.
	// JAL word2 = 0xE000 | (lower<<6) | (rS<<3) | rT
	// With lower=0, rS=5, rT=3: 0xE000 | 0 | (5<<3) | 3 = 0xE000 | 0x28 | 3 = 0xE02B.
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint16(buf[0:], 0xA003)
	binary.LittleEndian.PutUint16(buf[2:], 0xE02B)

	// Patch with addr = 0x00C0 (= 192): upper=3, lower=0.
	if err := patchLUIPlusJAL(buf, 0, 0x00C0); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// newWord1 = 0xA000 | (3<<3) | 3 = 0xA000 | 0x18 | 3 = 0xA01B
	// newWord2 = 0xE000 | (0<<6)  | (5<<3) | 3 = 0xE02B  (unchanged, lower=0)
	want1, want2 := uint16(0xA01B), uint16(0xE02B)
	got1, got2 := u16le(buf, 0), u16le(buf, 2)
	if got1 != want1 {
		t.Errorf("word1: got 0x%04X, want 0x%04X", got1, want1)
	}
	if got2 != want2 {
		t.Errorf("word2: got 0x%04X, want 0x%04X", got2, want2)
	}
}

func TestPatchLUIPlusJAL_PreservesRegs(t *testing.T) {
	// Verify rT (word1[2:0]) and rS (word2[5:3]) are independently preserved.
	for rT := 0; rT < 8; rT++ {
		for rS := 0; rS < 8; rS++ {
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint16(buf[0:], uint16(0xA000)|uint16(rT))
			binary.LittleEndian.PutUint16(buf[2:], uint16(0xE000)|uint16(rS<<3)|uint16(rT))

			// 0x0200 = 512: upper=8, lower=0.
			if err := patchLUIPlusJAL(buf, 0, 0x0200); err != nil {
				t.Fatalf("rT=%d rS=%d: %v", rT, rS, err)
			}
			// word1 = 0xA000 | (8<<3) | rT = 0xA040 | rT  → bits[2:0] = rT
			// word2 = 0xE000 | (0<<6) | (rS<<3) | rT      → bits[5:3] = rS
			gotRT := int(u16le(buf, 0) & 0x7)
			gotRS := int((u16le(buf, 2) >> 3) & 0x7)
			if gotRT != rT {
				t.Errorf("rT=%d rS=%d: rT not preserved, got %d", rT, rS, gotRT)
			}
			if gotRS != rS {
				t.Errorf("rT=%d rS=%d: rS not preserved, got %d", rT, rS, gotRS)
			}
		}
	}
}

func TestPatchLUIPlusJAL_OutOfBounds(t *testing.T) {
	buf := make([]byte, 3) // too short for a 4-byte patch
	if err := patchLUIPlusJAL(buf, 0, 0x1234); err == nil {
		t.Error("expected error for out-of-bounds patch, got nil")
	}
}

// ---- output format test -----------------------------------------------------

func TestWriteExecutable_Format(t *testing.T) {
	code := []byte{0x01, 0x02, 0x03, 0x04}
	data := []byte{0x05, 0x06}
	outPath := filepath.Join(t.TempDir(), "out.exe")

	if err := writeExecutable(outPath, code, data); err != nil {
		t.Fatalf("writeExecutable: %v", err)
	}

	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}

	wantSize := 16 + len(code) + len(data)
	if len(raw) != wantSize {
		t.Errorf("file size: got %d, want %d", len(raw), wantSize)
	}
	if gotMagic := u16le(raw, 0); gotMagic != MAGIC_EXE {
		t.Errorf("magic: got 0x%04X, want 0x%04X", gotMagic, MAGIC_EXE)
	}
	if gotCode := int(u16le(raw, 2)); gotCode != len(code) {
		t.Errorf("codeSize: got %d, want %d", gotCode, len(code))
	}
	if gotData := int(u16le(raw, 4)); gotData != len(data) {
		t.Errorf("dataSize: got %d, want %d", gotData, len(data))
	}
	// Reserved bytes [6:16] must be zero.
	for i := 6; i < 16; i++ {
		if raw[i] != 0 {
			t.Errorf("reserved byte[%d]: got %d, want 0", i, raw[i])
		}
	}
	if !bytes.Equal(raw[16:16+len(code)], code) {
		t.Errorf("code section mismatch")
	}
	if !bytes.Equal(raw[16+len(code):], data) {
		t.Errorf("data section mismatch")
	}
}
