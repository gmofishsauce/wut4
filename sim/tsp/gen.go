/*
 * Copyright (c) Jeff Berkowitz 2025.
 *
 * This file is part of wut4, which is licensed under the Affero GPL.
 * Code generated by the transpiler is owned by the schematic owner.
 */
package main

import (
	"fmt"
	//"unicode" TODO when processing type names - may be non-ASCII
	//"unicode/utf8" TODO when processing type names - may be non-ASCII
)

func transpile(root *ModelNode) error {
	componentTypes, err := getTypes(root)
	if err != nil {
		return err
	}

	componentInstances, err := getInstances(root, componentTypes)
	if err != nil {
		return err
	}

	if err := emitTopComment(root); err != nil {
		return err
	}
	if len(componentTypes) > 1_000_000_000 { // not
		msg("types %v instances %v\n", componentTypes, componentInstances)
	}

	return nil
}

// Type information for each type
type ComponentType struct {
	lib string
	part string
	pins []*ModelNode
	emit bool
}

func getTypes(root *ModelNode) ([]*ComponentType, error) {
	var componentTypes []*ComponentType

	for _, t := range(q(root, "libparts:libpart")) {
		lib := qss(t, "lib")
		part := qss(t, "part")
		pins := q(t, "pins:pin")
		componentTypes = append(componentTypes, &ComponentType{lib, part, pins, false})
		//msg("%s:%s has %d pins\n", lib, part, len(pins))
	}

	return componentTypes, nil
}

// Component instance for each component
// Points to a ComponentType
type ComponentInstance struct {
	ref string
	componentType *ComponentType
}

func getInstances(root *ModelNode, types []*ComponentType) ([]*ComponentInstance, error) {
	var componentInstances []*ComponentInstance
	for _, c := range(q(root, "components:comp")) {
		ref := qss(c, "ref")
		// Very primitive selector - emit for any A*, J*, or U*
		// This is (oddly) Unicode safe because we're only looking
		// for single-byte characters (for now)
		refCh := rune(ref[0])
		if refCh == 'A' || refCh == 'J' || refCh == 'U' {
			lib := qss(c, "libsource:lib")
			part := qss(c, "libsource:part")
			// Find the ComponentType for lib:part
			for _, t := range(types) {
				if lib == t.lib && part == t.part {
					t.emit = true // this type is referenced
					componentInstances = append(componentInstances, &ComponentInstance{ref, t})
					msg("emit %s type %s:%s\n", ref, t.lib, t.part)
				}
			}
		}
	}
	return componentInstances, nil
}

/*
Uncomment import "unicode" and "unicode/utf8" when needed

func emitTypes(root *ModelNode) error {
	emit("#include \"types.h\"\n")
	for _, t := range(q(root, "libparts:libpart")) {
		lib := qss(t, "lib")
		part := qss(t, "part")
		// The lib name or part name could be Unicode, although it's unlikely
		r, _ := utf8.DecodeRuneInString(lib)
		if !unicode.IsLetter(r) {
			lib = "T" + lib
		}
		msg("loop lib, part = %s %s\n", lib, part)
		// Emit a bitvec16_t if the part has 16 or fewer pins
		// Else emit a bitvec64_t (more pins not supported)
		n := len(q(t, "pins"))
		if n > 16 {
			emit("typedef %s_%s_t bitvec16_6\n", lib, part)
		} else {
			emit("typedef %s_%s_t bitvec64_t\n", lib, part)
		}
	}
	return nil
}
*/

// Emitters

// Use this wrapper for printf to emit all the content
func emit(format string, a ...any) (n int, err error) {
	n, err = fmt.Printf(format, a...)
	return n, err
}

// Generate a useful top comment
// TODO get the company and put it in the copyright
const topCommentStart = `/*
 * Copyright (c) %s 2025. All rights reserved.
 * This file was generated from a KiCad schematic. Do not edit.
 *
 * Tool: KiCad %s (schema version %s)
 * From: %s
 * Date: %s
 *
`

const topCommentEnd = " */\n"

func emitTopComment(root *ModelNode) error {
	schemaVersion := qss(root, "version")
	designSource := qss(root, "design:source")
	designDate := qss(root, "design:date")
	designTool := qss(root, "design:tool")
	companyName := "(TODO owning company here)"
	if designTool != "Eeschema 8.0.8" || schemaVersion != "E" {
		msg("WARNING: netlist was written by an untested version of KiCad. YMMV.\n")
	}
	emit(topCommentStart, companyName, designTool, schemaVersion, designSource, designDate)

	// Now emit a line for each sheet in the schematic.
	for _, sheet := range(q(root, "design:sheet")) {
		sheetNumber := qss(sheet, "number")
		sheetName := qss(sheet, "name")
		title := qss(sheet, "title_block:title")
		if valid(title) {
			emit(" * sheet %s: %s (%s)\n", sheetNumber, sheetName, title)
		} else {
			emit(" * sheet %s: %s\n", sheetNumber, sheetName)
		}
	}

	emit(topCommentEnd)
	return nil
}


func emitInstances(root *ModelNode) error {
/*
	for _, c := range(q("components:comp")) {
		key := qss(c, "ref")
		lib := qss(c, "libsource:lib")
		part := qss(c, "libsource:part")
	}
*/
	return nil
}
