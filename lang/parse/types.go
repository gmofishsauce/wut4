// YAPL Parser - Type Representation
// Types for the YAPL type system

package main

import "fmt"

// TypeKind represents the kind of type
type TypeKind int

const (
	TypeInvalid TypeKind = iota
	TypeVoid             // void (only for function returns)
	TypeBase             // base types: uint8, int16, uint16, block32, etc.
	TypePointer          // pointer to another type: @T
	TypeArray            // array of elements: [N]T
	TypeStruct           // struct type
)

// BaseType represents the built-in base types
type BaseType int

const (
	BaseInvalid BaseType = iota
	BaseUint8            // byte, uint8
	BaseInt16            // int16
	BaseUint16           // uint16
	BaseBlock32          // block32
	BaseBlock64          // block64
	BaseBlock128         // block128
)

// Type represents a YAPL type
type Type struct {
	Kind       TypeKind
	Base       BaseType // when Kind == TypeBase
	Pointee    *Type    // when Kind == TypePointer
	ElemType   *Type    // when Kind == TypeArray
	ArrayLen   int      // when Kind == TypeArray
	StructName string   // when Kind == TypeStruct
}

// Predefined types for convenience
var (
	TypeVoidType    = &Type{Kind: TypeVoid}
	TypeUint8Type   = &Type{Kind: TypeBase, Base: BaseUint8}
	TypeInt16Type   = &Type{Kind: TypeBase, Base: BaseInt16}
	TypeUint16Type  = &Type{Kind: TypeBase, Base: BaseUint16}
	TypeBlock32Type = &Type{Kind: TypeBase, Base: BaseBlock32}
	TypeBlock64Type = &Type{Kind: TypeBase, Base: BaseBlock64}
	TypeBlock128Type = &Type{Kind: TypeBase, Base: BaseBlock128}
)

// NewPointerType creates a pointer type to the given type
func NewPointerType(pointee *Type) *Type {
	return &Type{
		Kind:    TypePointer,
		Pointee: pointee,
	}
}

// NewArrayType creates an array type
func NewArrayType(elemType *Type, length int) *Type {
	return &Type{
		Kind:     TypeArray,
		ElemType: elemType,
		ArrayLen: length,
	}
}

// NewStructType creates a struct type reference
func NewStructType(name string) *Type {
	return &Type{
		Kind:       TypeStruct,
		StructName: name,
	}
}

// String returns a string representation of the type
func (t *Type) String() string {
	if t == nil {
		return "<nil>"
	}
	switch t.Kind {
	case TypeVoid:
		return "void"
	case TypeBase:
		return t.Base.String()
	case TypePointer:
		return "@" + t.Pointee.String()
	case TypeArray:
		return fmt.Sprintf("[%d]%s", t.ArrayLen, t.ElemType.String())
	case TypeStruct:
		return t.StructName
	default:
		return "<invalid>"
	}
}

// String returns a string representation of a base type
func (b BaseType) String() string {
	switch b {
	case BaseUint8:
		return "uint8"
	case BaseInt16:
		return "int16"
	case BaseUint16:
		return "uint16"
	case BaseBlock32:
		return "block32"
	case BaseBlock64:
		return "block64"
	case BaseBlock128:
		return "block128"
	default:
		return "<invalid>"
	}
}

// Size returns the size in bytes of a type
// Returns -1 if the size cannot be determined (e.g., unknown struct)
func (t *Type) Size(structs map[string]*StructDef) int {
	if t == nil {
		return -1
	}
	switch t.Kind {
	case TypeVoid:
		return 0
	case TypeBase:
		return t.Base.Size()
	case TypePointer:
		return 2 // all pointers are 16-bit
	case TypeArray:
		elemSize := t.ElemType.Size(structs)
		if elemSize < 0 {
			return -1
		}
		return elemSize * t.ArrayLen
	case TypeStruct:
		if structs == nil {
			return -1
		}
		if def, ok := structs[t.StructName]; ok {
			return def.Size
		}
		return -1
	default:
		return -1
	}
}

// Alignment returns the alignment requirement in bytes
func (t *Type) Alignment(structs map[string]*StructDef) int {
	if t == nil {
		return 1
	}
	switch t.Kind {
	case TypeVoid:
		return 1
	case TypeBase:
		return t.Base.Alignment()
	case TypePointer:
		return 2 // all pointers are 16-bit aligned
	case TypeArray:
		return t.ElemType.Alignment(structs)
	case TypeStruct:
		if structs == nil {
			return 2
		}
		if def, ok := structs[t.StructName]; ok {
			return def.Align
		}
		return 2
	default:
		return 1
	}
}

// Size returns the size in bytes of a base type
func (b BaseType) Size() int {
	switch b {
	case BaseUint8:
		return 1
	case BaseInt16, BaseUint16:
		return 2
	case BaseBlock32:
		return 4 // 32 bits = 4 bytes
	case BaseBlock64:
		return 8 // 64 bits = 8 bytes
	case BaseBlock128:
		return 16 // 128 bits = 16 bytes
	default:
		return -1
	}
}

// Alignment returns the alignment requirement in bytes
func (b BaseType) Alignment() int {
	switch b {
	case BaseUint8:
		return 1
	case BaseInt16, BaseUint16:
		return 2
	case BaseBlock32, BaseBlock64, BaseBlock128:
		return 4 // block types aligned on 32-bit boundaries
	default:
		return 1
	}
}

// IsBlockType returns true if this is a block type
func (b BaseType) IsBlockType() bool {
	return b == BaseBlock32 || b == BaseBlock64 || b == BaseBlock128
}

// Equal returns true if two types are equal
func (t *Type) Equal(other *Type) bool {
	if t == nil || other == nil {
		return t == other
	}
	if t.Kind != other.Kind {
		return false
	}
	switch t.Kind {
	case TypeVoid:
		return true
	case TypeBase:
		return t.Base == other.Base
	case TypePointer:
		return t.Pointee.Equal(other.Pointee)
	case TypeArray:
		return t.ArrayLen == other.ArrayLen && t.ElemType.Equal(other.ElemType)
	case TypeStruct:
		return t.StructName == other.StructName
	default:
		return false
	}
}

// IsIntegral returns true if this is an integral type (uint8, int16, uint16)
func (t *Type) IsIntegral() bool {
	if t == nil || t.Kind != TypeBase {
		return false
	}
	return t.Base == BaseUint8 || t.Base == BaseInt16 || t.Base == BaseUint16
}

// IsPointer returns true if this is a pointer type
func (t *Type) IsPointer() bool {
	return t != nil && t.Kind == TypePointer
}

// StructDef holds struct definition info needed for size/alignment calculations
// (forward declaration - full definition in symtab.go)
type StructDef struct {
	Name   string
	Fields []FieldDef
	Size   int
	Align  int
}

// FieldDef holds field definition info
type FieldDef struct {
	Name     string
	Type     *Type
	ArrayLen int // 0 if not array
	Offset   int
}
