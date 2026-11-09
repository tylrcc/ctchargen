package traveller_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// Holds: Die roll conventions: `N+`, `N−`, exact.
func TestParseTarget(t *testing.T) {
	t.Parallel()

	// The notation is how the modality is read back: the sign after the
	// number, or none.
	for _, tc := range []struct {
		in       string
		number   int
		notation string
	}{
		{"8+", 8, "8+"},
		{"3+", 3, "3+"},
		{"10+", 10, "10+"},
		// A transcription is typed with an ASCII hyphen, and the minus the book
		// prints may be typed as U+2212. Both must read as the same target.
		{"3-", 3, "3-"},
		{"3−", 3, "3-"},
		{"12", 12, "12"},
		{"  8+  ", 8, "8+"},
	} {
		got, err := traveller.ParseTarget(tc.in)
		if err != nil {
			t.Errorf("ParseTarget(%q): %v", tc.in, err)

			continue
		}

		if got.Number() != tc.number || got.String() != tc.notation {
			t.Errorf("ParseTarget(%q) = %d, %v; want %d, %s",
				tc.in, got.Number(), got, tc.number, tc.notation)
		}
	}
}

// A malformed target is a build defect, not a runtime condition, so parsing
// must refuse rather than guess.
func TestParseTargetRefusesNonsense(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"", "   ", "+", "-", "−", "8++", "eight", "8+3", "12abc", "-8"} {
		got, err := traveller.ParseTarget(in)
		if err == nil {
			t.Errorf("ParseTarget(%q) = %v, want an error", in, got)
		}
	}
}

// A sign is a plus or either minus a transcription may carry. An en dash looks
// like a minus and is not one.
func TestSign(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		glyph  rune
		sign   int
		enDash bool
	}{
		{'+', 1, false},
		{'-', -1, false},
		{'−', -1, false},
		{'–', 0, true},
		{'x', 0, false},
	} {
		sign, err := traveller.Sign(tc.glyph)
		if sign != tc.sign || errors.Is(err, traveller.ErrEnDash) != tc.enDash || err != nil && !tc.enDash {
			t.Errorf("Sign(%q) = %d, %v; want %d, en dash %v", tc.glyph, sign, err, tc.sign, tc.enDash)
		}
	}
}

// An en dash is refused in either position, and the error says why.
func TestParseTargetRefusesAnEnDashByName(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"8–", "–3"} {
		got, err := traveller.ParseTarget(in)
		if err == nil {
			t.Errorf("ParseTarget(%q) = %v, want an error", in, got)

			continue
		}

		if !errors.Is(err, traveller.ErrEnDash) {
			t.Errorf("ParseTarget(%q): %v; want it to wrap ErrEnDash", in, err)
		}

		if !strings.Contains(err.Error(), "an en dash is not a minus") {
			t.Errorf("ParseTarget(%q): %v; want the error to name the en dash", in, err)
		}
	}
}

// Holds: Die roll conventions: `N+`, `N−`, exact.
func TestTargetSatisfied(t *testing.T) {
	t.Parallel()

	parsed := func(s string) traveller.Target {
		target, err := traveller.ParseTarget(s)
		if err != nil {
			t.Fatalf("ParseTarget(%q): %v", s, err)
		}

		return target
	}

	atLeast, atMost, exactly := parsed("8+"), parsed("3-"), parsed("12")

	for _, tc := range []struct {
		name   string
		target traveller.Target
		sum    int
		want   bool
	}{
		{"8+ met above", atLeast, 9, true},
		{"8+ met exactly", atLeast, 8, true},
		{"8+ missed", atLeast, 7, false},
		{"3- met below", atMost, 2, true},
		{"3- met exactly", atMost, 3, true},
		{"3- missed", atMost, 4, false},
		{"12 exactly", exactly, 12, true},
		{"12 not 11", exactly, 11, false},
		{"12 not 13", exactly, 13, false},
	} {
		if got := tc.target.Satisfied(tc.sum); got != tc.want {
			t.Errorf("%s: %v.Satisfied(%d) = %v, want %v", tc.name, tc.target, tc.sum, got, tc.want)
		}
	}
}

// A Target nobody set must fail closed. Two dice cannot total 0, so a zero
// Target satisfies nothing the procedure can throw. Outside this package
// only ParseTarget builds one otherwise, so no other modality than the three
// reaches a caller (#370).
func TestAnUnsetTargetSatisfiesNothing(t *testing.T) {
	t.Parallel()

	unset := traveller.Target{}
	for sum := 1; sum <= 12; sum++ {
		if unset.Satisfied(sum) {
			t.Fatalf("a zero Target was satisfied by a throw of %d; an unset target must satisfy nothing", sum)
		}
	}
}
