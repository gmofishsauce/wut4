// YAPL Parser - Symbol Table
// Simplified symbol table: global scope + flat per-function scope

package main

import "fmt"

// Storage represents where a variable is stored
type Storage int

const (
	StorageInvalid Storage = iota
	StorageGlobal          // global variable (uppercase name)
	StorageStatic          // static variable (lowercase name at file level)
	StorageParam           // function parameter
	StorageLocal           // local variable
)

func (s Storage) String() string {
	switch s {
	case StorageGlobal:
		return "GLOBAL"
	case StorageStatic:
		return "STATIC"
	case StorageParam:
		return "PARAM"
	case StorageLocal:
		return "LOCAL"
	default:
		return "INVALID"
	}
}

// SymKind represents the kind of symbol
type SymKind int

const (
	SymInvalid SymKind = iota
	SymConst           // constant
	SymVar             // variable
	SymFunc            // function
	SymStruct          // struct type
)

// Symbol represents an entry in the symbol table
type Symbol struct {
	Name     string
	Kind     SymKind
	Type     *Type   // for SymVar, SymFunc (return type)
	ConstVal int64   // for SymConst
	Storage  Storage // for SymVar
	Offset   int     // data offset for global/static, stack offset for local
	IsPublic bool    // true if name starts with uppercase
	Loc      SourceLoc
	// For functions
	Params []*ParamSymbol
	Locals []*LocalSymbol
	Labels map[string]*LabelSymbol
	FrameSize int // total bytes for locals
}

// ParamSymbol represents a function parameter
type ParamSymbol struct {
	Name    string
	Type    *Type
	Index   int    // 0,1,2 = R1,R2,R3; 3+ = stack position
	Loc     SourceLoc
}

// LocalSymbol represents a local variable or constant
type LocalSymbol struct {
	Name     string
	Type     *Type
	IsConst  bool
	ConstVal int64  // if IsConst
	Offset   int    // stack offset (negative from frame base)
	ArrayLen int    // 0 if not array
	Loc      SourceLoc
}

// LabelSymbol represents a label within a function
type LabelSymbol struct {
	Name      string
	StmtIndex int // index into FuncDecl.Body
	Loc       SourceLoc
}

// SymbolTable holds all symbols for a compilation unit
type SymbolTable struct {
	// Global scope
	Globals map[string]*Symbol
	// Struct definitions (also in Globals, but indexed by name for type lookup)
	Structs map[string]*StructDef
	// Current data segment offset for global/static variables
	DataOffset int
	// Errors collected during symbol table construction
	Errors []string
}

// NewSymbolTable creates a new symbol table
func NewSymbolTable() *SymbolTable {
	return &SymbolTable{
		Globals:    make(map[string]*Symbol),
		Structs:    make(map[string]*StructDef),
		DataOffset: 0,
		Errors:     nil,
	}
}

// DefineConst defines a constant in the global scope
func (st *SymbolTable) DefineConst(name string, value int64, loc SourceLoc) error {
	if _, exists := st.Globals[name]; exists {
		return fmt.Errorf("redefinition of '%s'", name)
	}
	st.Globals[name] = &Symbol{
		Name:     name,
		Kind:     SymConst,
		ConstVal: value,
		IsPublic: isPublic(name),
		Loc:      loc,
	}
	return nil
}

// DefineGlobalVar defines a global or static variable
func (st *SymbolTable) DefineGlobalVar(name string, typ *Type, arrayLen int, loc SourceLoc) error {
	if _, exists := st.Globals[name]; exists {
		return fmt.Errorf("redefinition of '%s'", name)
	}

	storage := StorageStatic
	if isPublic(name) {
		storage = StorageGlobal
	}

	// Calculate size and alignment
	size := typ.Size(st.Structs)
	if arrayLen > 0 {
		size *= arrayLen
	}
	align := typ.Alignment(st.Structs)

	// Align the data offset
	st.DataOffset = alignUp(st.DataOffset, align)
	offset := st.DataOffset
	st.DataOffset += size

	st.Globals[name] = &Symbol{
		Name:     name,
		Kind:     SymVar,
		Type:     typ,
		Storage:  storage,
		Offset:   offset,
		IsPublic: isPublic(name),
		Loc:      loc,
	}
	return nil
}

// DefineFunc defines a function
func (st *SymbolTable) DefineFunc(name string, returnType *Type, loc SourceLoc) (*Symbol, error) {
	if _, exists := st.Globals[name]; exists {
		return nil, fmt.Errorf("redefinition of '%s'", name)
	}
	sym := &Symbol{
		Name:     name,
		Kind:     SymFunc,
		Type:     returnType,
		IsPublic: isPublic(name),
		Loc:      loc,
		Params:   make([]*ParamSymbol, 0),
		Locals:   make([]*LocalSymbol, 0),
		Labels:   make(map[string]*LabelSymbol),
	}
	st.Globals[name] = sym
	return sym, nil
}

// DefineStruct defines a struct type
func (st *SymbolTable) DefineStruct(name string, fields []*FieldDecl, loc SourceLoc) (*StructDef, error) {
	if _, exists := st.Globals[name]; exists {
		return nil, fmt.Errorf("redefinition of '%s'", name)
	}

	// Compute field offsets, struct size, and alignment
	def := &StructDef{
		Name:   name,
		Fields: make([]FieldDef, len(fields)),
		Size:   0,
		Align:  2, // minimum struct alignment
	}

	offset := 0
	for i, f := range fields {
		fieldAlign := f.FieldType.Alignment(st.Structs)
		if fieldAlign > def.Align {
			def.Align = fieldAlign
		}

		// Align offset
		offset = alignUp(offset, fieldAlign)

		def.Fields[i] = FieldDef{
			Name:     f.Name,
			Type:     f.FieldType,
			ArrayLen: f.ArrayLen,
			Offset:   offset,
		}

		// Calculate field size
		fieldSize := f.FieldType.Size(st.Structs)
		if f.ArrayLen > 0 {
			fieldSize *= f.ArrayLen
		}
		offset += fieldSize
	}

	// Final size rounded up to alignment
	def.Size = alignUp(offset, def.Align)

	// Add to symbol table
	st.Globals[name] = &Symbol{
		Name:     name,
		Kind:     SymStruct,
		IsPublic: isPublic(name),
		Loc:      loc,
	}
	st.Structs[name] = def

	return def, nil
}

// LookupGlobal looks up a symbol in the global scope
func (st *SymbolTable) LookupGlobal(name string) *Symbol {
	return st.Globals[name]
}

// LookupStruct looks up a struct definition
func (st *SymbolTable) LookupStruct(name string) *StructDef {
	return st.Structs[name]
}

// AddError adds an error to the error list
func (st *SymbolTable) AddError(format string, args ...interface{}) {
	st.Errors = append(st.Errors, fmt.Sprintf(format, args...))
}

// HasErrors returns true if there are errors
func (st *SymbolTable) HasErrors() bool {
	return len(st.Errors) > 0
}

// FuncScope represents the scope for a function being parsed
type FuncScope struct {
	Func       *Symbol
	ParamMap   map[string]*ParamSymbol
	LocalMap   map[string]*LocalSymbol
	FrameOffset int // current stack frame offset (grows negative)
}

// NewFuncScope creates a new function scope
func NewFuncScope(funcSym *Symbol) *FuncScope {
	return &FuncScope{
		Func:        funcSym,
		ParamMap:    make(map[string]*ParamSymbol),
		LocalMap:    make(map[string]*LocalSymbol),
		FrameOffset: 0,
	}
}

// AddParam adds a parameter to the function scope
func (fs *FuncScope) AddParam(name string, typ *Type, loc SourceLoc) error {
	if _, exists := fs.ParamMap[name]; exists {
		return fmt.Errorf("duplicate parameter '%s'", name)
	}
	if _, exists := fs.LocalMap[name]; exists {
		return fmt.Errorf("'%s' already declared", name)
	}

	index := len(fs.Func.Params)
	param := &ParamSymbol{
		Name:  name,
		Type:  typ,
		Index: index,
		Loc:   loc,
	}
	fs.ParamMap[name] = param
	fs.Func.Params = append(fs.Func.Params, param)
	return nil
}

// AddLocal adds a local variable to the function scope
func (fs *FuncScope) AddLocal(name string, typ *Type, arrayLen int, structs map[string]*StructDef, loc SourceLoc) error {
	if _, exists := fs.ParamMap[name]; exists {
		return fmt.Errorf("'%s' shadows parameter", name)
	}
	if _, exists := fs.LocalMap[name]; exists {
		return fmt.Errorf("redefinition of '%s'", name)
	}

	// Calculate size and alignment
	size := typ.Size(structs)
	if arrayLen > 0 {
		size *= arrayLen
	}
	align := typ.Alignment(structs)

	// Allocate space (stack grows down, so offset is negative)
	fs.FrameOffset -= size
	fs.FrameOffset = alignDown(fs.FrameOffset, align)

	local := &LocalSymbol{
		Name:     name,
		Type:     typ,
		IsConst:  false,
		Offset:   fs.FrameOffset,
		ArrayLen: arrayLen,
		Loc:      loc,
	}
	fs.LocalMap[name] = local
	fs.Func.Locals = append(fs.Func.Locals, local)
	return nil
}

// AddLocalConst adds a local constant to the function scope
func (fs *FuncScope) AddLocalConst(name string, value int64, loc SourceLoc) error {
	if _, exists := fs.ParamMap[name]; exists {
		return fmt.Errorf("'%s' shadows parameter", name)
	}
	if _, exists := fs.LocalMap[name]; exists {
		return fmt.Errorf("redefinition of '%s'", name)
	}

	local := &LocalSymbol{
		Name:     name,
		IsConst:  true,
		ConstVal: value,
		Loc:      loc,
	}
	fs.LocalMap[name] = local
	fs.Func.Locals = append(fs.Func.Locals, local)
	return nil
}

// AddLabel adds a label to the function scope
func (fs *FuncScope) AddLabel(name string, stmtIndex int, loc SourceLoc) error {
	if _, exists := fs.Func.Labels[name]; exists {
		return fmt.Errorf("duplicate label '%s'", name)
	}
	fs.Func.Labels[name] = &LabelSymbol{
		Name:      name,
		StmtIndex: stmtIndex,
		Loc:       loc,
	}
	return nil
}

// Finalize computes the final frame size
func (fs *FuncScope) Finalize() {
	// Frame size is the absolute value of the lowest offset
	if fs.FrameOffset < 0 {
		fs.Func.FrameSize = -fs.FrameOffset
	} else {
		fs.Func.FrameSize = 0
	}

	// Ensure frame size is even for word alignment
	// The padding is conceptually at the bottom of the frame (most negative address)
	// Offsets are NOT adjusted - the conversion formula (frameSize + negativeOffset)
	// automatically gives correct positive offsets
	if fs.Func.FrameSize%2 != 0 {
		fs.Func.FrameSize++
	}
}

// LookupLocal looks up a name in the function scope (params then locals)
func (fs *FuncScope) LookupLocal(name string) (*ParamSymbol, *LocalSymbol) {
	if p, ok := fs.ParamMap[name]; ok {
		return p, nil
	}
	if l, ok := fs.LocalMap[name]; ok {
		return nil, l
	}
	return nil, nil
}

// Helper functions

func isPublic(name string) bool {
	if len(name) == 0 {
		return false
	}
	ch := name[0]
	return ch >= 'A' && ch <= 'Z'
}

func alignUp(n, align int) int {
	return (n + align - 1) &^ (align - 1)
}

func alignDown(n, align int) int {
	return n &^ (align - 1)
}
