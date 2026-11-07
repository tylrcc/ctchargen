package traveller_test

import (
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// Holds: Die roll conventions: `N+`, `N−`, exact.
func TestParseTarget(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in       string
		number   int
		modality traveller.Modality
	}{
		{"8+", 8, traveller.AtLeast},
		{"3+", 3, traveller.AtLeast},
		{"10+", 10, traveller.AtLeast},
		// The book sets its minus as U+2212; a data file is typed with an
		// ASCII hyphen. Both must read as the same target.
		{"3-", 3, traveller.AtMost},
		{"3−", 3, traveller.AtMost},
		{"12", 12, traveller.Exactly},
		{"  8+  ", 8, traveller.AtLeast},
	} {
		got, err := traveller.ParseTarget(tc.in)
		if err != nil {
			t.Errorf("ParseTarget(%q): %v", tc.in, err)

			continue
		}

		if got.Number() != tc.number || got.Modality() != tc.modality {
			t.Errorf("ParseTarget(%q) = %d%v, want %d%v",
				tc.in, got.Number(), got.Modality(), tc.number, tc.modality)
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

// Holds: Die roll conventions: `N+`, `N−`, exact.
func TestTargetSatisfied(t *testing.T) {
	t.Parallel()

	atLeast := traveller.NewTarget(8, traveller.AtLeast)
	atMost := traveller.NewTarget(3, traveller.AtMost)
	exactly := traveller.NewTarget(12, traveller.Exactly)

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

func TestTargetRoundTripsItsNotation(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"8+", "3-", "12"} {
		got, err := traveller.ParseTarget(in)
		if err != nil {
			t.Fatalf("ParseTarget(%q): %v", in, err)
		}

		if got.String() != in {
			t.Errorf("ParseTarget(%q).String() = %q", in, got.String())
		}
	}
}

// A target built with a modality that does not exist satisfies nothing, and
// neither does one nobody set.
func TestAnUnknownModalitySatisfiesNothing(t *testing.T) {
	t.Parallel()

	if traveller.NewTarget(8, traveller.Modality(99)).Satisfied(9) {
		t.Error("a target with an unknown modality was satisfied")
	}

	// A Target nobody set must fail closed. Two dice cannot total 0, so a
	// zero Target satisfies nothing the procedure can throw.
	unset := traveller.Target{}
	for sum := 1; sum <= 12; sum++ {
		if unset.Satisfied(sum) {
			t.Fatalf("a zero Target was satisfied by a throw of %d; an unset target must satisfy nothing", sum)
		}
	}
}
