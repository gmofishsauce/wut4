package main

import (
	"strings"
	"testing"
)

type peepTest struct {
	name  string
	input string
	want  string
}

// nInstrs returns n copies of "    mv r1, r2\n" (each 2 bytes assembled).
func nInstrs(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString("    mv r1, r2\n")
	}
	return sb.String()
}

var peepTests = []peepTest{
	// --- existing patterns ---
	{
		name:  "mv_self_elim",
		input: "    mv r4, r4\n",
		want:  "",
	},
	{
		name:  "ldi_zero",
		input: "    ldi r4, 0\n",
		want:  "    mv r4, r0\n",
	},
	{
		name: "stw_ldw_same_reg",
		input: "" +
			"    stw r6, r7, 8\n" +
			"    ldw r6, r7, 8\n",
		want: "    stw r6, r7, 8\n",
	},
	{
		name: "stw_ldw_diff_reg",
		input: "" +
			"    stw r6, r7, 8\n" +
			"    ldw r4, r7, 8\n",
		want: "" +
			"    stw r6, r7, 8\n" +
			"    mv r4, r6\n",
	},
	{
		name: "label_breaks_window",
		input: "" +
			"    stw r6, r7, 8\n" +
			"loop:\n" +
			"    ldw r6, r7, 8\n",
		want: "" +
			"    stw r6, r7, 8\n" +
			"loop:\n" +
			"    ldw r6, r7, 8\n",
	},
	{
		name: "comment_transparent",
		input: "" +
			"    stw r6, r7, 8\n" +
			"; a comment\n" +
			"    ldw r6, r7, 8\n",
		want: "" +
			"    stw r6, r7, 8\n" +
			"; a comment\n",
	},
	{
		name: "ldi_add_fold",
		input: "" +
			"    ldi r4, 5\n" +
			"    add r6, r5, r4\n",
		want: "    adi r6, r5, 5\n",
	},
	{
		name: "ldi_add_commute",
		input: "" +
			"    ldi r4, 5\n" +
			"    add r6, r4, r5\n",
		want: "    adi r6, r5, 5\n",
	},
	{
		name: "ldi_add_too_large",
		input: "" +
			"    ldi r4, 100\n" +
			"    add r6, r5, r4\n",
		want: "" +
			"    ldi r4, 100\n" +
			"    add r6, r5, r4\n",
	},
	{
		name: "chain_mv_self",
		input: "" +
			"    mv r6, r6\n" +
			"    stw r4, r7, 4\n" +
			"    ldw r4, r7, 4\n",
		want: "    stw r4, r7, 4\n",
	},
	{
		name: "passthrough_comments",
		input: "" +
			"; first comment\n" +
			"\n" +
			"; second comment\n",
		want: "" +
			"; first comment\n" +
			"\n" +
			"; second comment\n",
	},
	{
		name: "passthrough_labels",
		input: "" +
			"start:\n" +
			"loop:\n",
		want: "" +
			"start:\n" +
			"loop:\n",
	},
	{
		name: "passthrough_directives",
		input: "" +
			".code\n" +
			".data\n",
		want: "" +
			".code\n" +
			".data\n",
	},

	// --- branch-over-jal folding: forward target ---
	//
	// Layout (byte addresses):
	//   0: brnz l_skip0   (2 bytes)
	//   2: jal  l_target  (4 bytes)
	//   6: l_skip0:       (0 bytes)
	//   8: mv r1, r2      (2 bytes)
	//  10: l_target:      (0 bytes)
	//
	// Post-opt offset = (10 - 4) - (0 + 2) = 4.  In range.
	{
		name: "branch_fold_forward",
		input: "" +
			"    brnz l_skip0\n" +
			"    jal  l_target\n" +
			"l_skip0:\n" +
			"    mv r1, r2\n" +
			"l_target:\n" +
			"    ret\n",
		want: "" +
			"    brz l_target\n" +
			"l_skip0:\n" +
			"    mv r1, r2\n" +
			"l_target:\n" +
			"    ret\n",
	},

	// --- branch-over-jal folding: backward target ---
	//
	// Layout:
	//   0: l_target:      (0 bytes)
	//   0: mv r1, r2      (2 bytes)
	//   2: brnz l_skip1   (2 bytes)
	//   4: jal  l_target  (4 bytes)
	//   8: l_skip1:       (0 bytes)
	//
	// Post-opt offset = 0 - (2 + 2) = -4.  In range.
	{
		name: "branch_fold_backward",
		input: "" +
			"l_target:\n" +
			"    mv r1, r2\n" +
			"    brnz l_skip1\n" +
			"    jal  l_target\n" +
			"l_skip1:\n" +
			"    ret\n",
		want: "" +
			"l_target:\n" +
			"    mv r1, r2\n" +
			"    brz l_target\n" +
			"l_skip1:\n" +
			"    ret\n",
	},

	// --- branch-over-jal: comment between branch and jal is transparent ---
	{
		name: "branch_fold_comment_transparent",
		input: "" +
			"    brz l_skip2\n" +
			"; intervening comment\n" +
			"    jal l_target\n" +
			"l_skip2:\n" +
			"    ret\n" +
			"l_target:\n" +
			"    ret\n",
		want: "" +
			"    brnz l_target\n" +
			"; intervening comment\n" +
			"l_skip2:\n" +
			"    ret\n" +
			"l_target:\n" +
			"    ret\n",
	},

	// --- branch-over-jal: label between branch and jal breaks pattern ---
	// The intervening label prevents branch-over-jal folding.
	// The jal targets an external symbol so standalone jal→br also can't fire,
	// leaving the output identical to the input.
	{
		name: "branch_fold_label_breaks",
		input: "" +
			"    brnz l_skip3\n" +
			"other:\n" +
			"    jal  ExternalTarget\n" +
			"l_skip3:\n" +
			"    ret\n",
		want: "" +
			"    brnz l_skip3\n" +
			"other:\n" +
			"    jal  ExternalTarget\n" +
			"l_skip3:\n" +
			"    ret\n",
	},

	// --- branch-over-jal: all invertible condition pairs ---
	{
		name: "branch_fold_brz",
		input: "" +
			"    brz l_s\n" +
			"    jal l_t\n" +
			"l_s:\n" +
			"l_t:\n" +
			"    ret\n",
		want: "" +
			"    brnz l_t\n" +
			"l_s:\n" +
			"l_t:\n" +
			"    ret\n",
	},
	{
		name: "branch_fold_brc",
		input: "" +
			"    brc l_s\n" +
			"    jal l_t\n" +
			"l_s:\n" +
			"l_t:\n" +
			"    ret\n",
		want: "" +
			"    brnc l_t\n" +
			"l_s:\n" +
			"l_t:\n" +
			"    ret\n",
	},
	{
		name: "branch_fold_brsge",
		input: "" +
			"    brsge l_s\n" +
			"    jal l_t\n" +
			"l_s:\n" +
			"l_t:\n" +
			"    ret\n",
		want: "" +
			"    brslt l_t\n" +
			"l_s:\n" +
			"l_t:\n" +
			"    ret\n",
	},

	// --- branch-over-jal: external symbol (not in label map) → no change ---
	{
		name: "branch_fold_external_no_change",
		input: "" +
			"    brnz l_skip4\n" +
			"    jal  ExternalFunc\n" +
			"l_skip4:\n" +
			"    ret\n",
		want: "" +
			"    brnz l_skip4\n" +
			"    jal  ExternalFunc\n" +
			"l_skip4:\n" +
			"    ret\n",
	},

	// --- standalone jal → br ---
	//
	// Layout:
	//   0: jal l_near   (4 bytes)
	//   4: l_near:      (0 bytes)
	//
	// Post-opt offset = (4 - 2) - (0 + 2) = 0.  In range.
	{
		name: "standalone_jal_to_br",
		input: "" +
			"    jal l_near\n" +
			"l_near:\n" +
			"    ret\n",
		want: "" +
			"    br l_near\n" +
			"l_near:\n" +
			"    ret\n",
	},

	// --- standalone jal: external symbol → no change ---
	{
		name: "standalone_jal_external_no_change",
		input: "" +
			"    jal OtherFunc\n" +
			"    ret\n",
		want: "" +
			"    jal OtherFunc\n" +
			"    ret\n",
	},
}

// branchFoldAtLimitInput builds an assembly fragment whose forward branch target
// is exactly 'filler' two-byte instructions after the skip label.
//
// Layout (byte addresses):
//
//	 0: brnz l_skipX       (2 bytes)
//	 2: jal  l_farTarget   (4 bytes)
//	 6: l_skipX:           (0 bytes)
//	 6: <filler × mv>      (filler × 2 bytes)
//	 6+2*filler: l_farTarget:
//
// Post-opt offset = (6 + 2*filler - 4) - (0 + 2) = 2*filler.
func branchFoldAtLimitInput(filler int) string {
	var sb strings.Builder
	sb.WriteString("    brnz l_skipX\n")
	sb.WriteString("    jal  l_farTarget\n")
	sb.WriteString("l_skipX:\n")
	sb.WriteString(nInstrs(filler))
	sb.WriteString("l_farTarget:\n")
	sb.WriteString("    ret\n")
	return sb.String()
}

func TestPeep(t *testing.T) {
	for _, tc := range peepTests {
		t.Run(tc.name, func(t *testing.T) {
			got := runPeep(tc.input)
			if got != tc.want {
				t.Errorf("input:\n%s\ngot:\n%s\nwant:\n%s", tc.input, got, tc.want)
			}
		})
	}
}

// TestBranchFoldRangeBoundary verifies the ±512-byte range check.
// With 255 filler instructions the post-opt offset is 510 (in range).
// With 256 filler instructions the post-opt offset is 512 (out of range).
func TestBranchFoldRangeBoundary(t *testing.T) {
	// 255 fillers → post-opt offset = 510 ≤ 511: should fold.
	inRange := branchFoldAtLimitInput(255)
	got := runPeep(inRange)
	if strings.Contains(got, "jal") {
		t.Errorf("255-filler case: expected jal to be folded, but got:\n%s", got)
	}
	if !strings.Contains(got, "    brz l_farTarget\n") {
		t.Errorf("255-filler case: expected 'brz l_farTarget', got:\n%s", got)
	}

	// 256 fillers → post-opt offset = 512 > 511: should NOT fold.
	outOfRange := branchFoldAtLimitInput(256)
	got = runPeep(outOfRange)
	if !strings.Contains(got, "    jal  l_farTarget\n") {
		t.Errorf("256-filler case: expected jal to remain, but got:\n%s", got)
	}
}

func runPeep(input string) string {
	rawLines := strings.Split(strings.TrimRight(input, "\n"), "\n")
	if input == "" {
		rawLines = nil
	}
	lines := parseAll(rawLines)
	optimize(lines)

	var sb strings.Builder
	for _, l := range lines {
		if l.Kind == LineDeleted {
			continue
		}
		sb.WriteString(l.Raw)
		sb.WriteByte('\n')
	}
	return sb.String()
}
