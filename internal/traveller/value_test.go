package traveller_test

import (
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// Months exist only because a medical crisis recovery adds "one die equals
// the number of months in added age" (pp. 7-8). They carry into years.
func TestAgeCarriesMonthsIntoYears(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		age        traveller.Age
		wantYears  int
		wantMonths int
	}{
		{"generation begins at 18", traveller.NewAge(18), 18, 0},
		{"a term of service", traveller.NewAge(18).PlusYears(4), 22, 0},
		{"one crisis, one die of months", traveller.NewAge(34).PlusMonths(5), 34, 5},
		{"months short of a year", traveller.NewAge(34).PlusMonths(11), 34, 11},
		{"twelve months is a year", traveller.NewAge(34).PlusMonths(12), 35, 0},
		{"two crises carry over", traveller.NewAge(34).PlusMonths(7).PlusMonths(6), 35, 1},
		{"years added after months keep them", traveller.NewAge(34).PlusMonths(5).PlusYears(4), 38, 5},
		// The procedure never runs an age backwards, but the type's one
		// invariant is that Months is 0 through 11, and it has to hold for
		// whatever it is handed rather than only for what it expects.
		{"a month back borrows a year", traveller.NewAge(34).PlusMonths(-1), 33, 11},
		{"a year back exactly", traveller.NewAge(34).PlusMonths(-12), 33, 0},
		{"more than a year back", traveller.NewAge(34).PlusMonths(-13), 32, 11},
		{"back to where it started", traveller.NewAge(34).PlusMonths(7).PlusMonths(-7), 34, 0},
	} {
		if tc.age.Years() != tc.wantYears || tc.age.Months() != tc.wantMonths {
			t.Errorf("%s: %d years %d months, want %d years %d months",
				tc.name, tc.age.Years(), tc.age.Months(), tc.wantYears, tc.wantMonths)
		}

		if m := tc.age.Months(); m < 0 || m >= traveller.MonthsPerYear {
			t.Errorf("%s: Months() = %d, outside 0 through 11", tc.name, m)
		}
	}
}

func TestAgeReads(t *testing.T) {
	t.Parallel()

	if got := traveller.NewAge(38).String(); got != "38" {
		t.Errorf("a whole-year age reads %q", got)
	}

	if got := traveller.NewAge(38).PlusMonths(5).String(); got != "38 years 5 months" {
		t.Errorf("an age with months reads %q", got)
	}
}

func TestCreditsRead(t *testing.T) {
	t.Parallel()

	// P. 21's retirement pay at five terms.
	if got := traveller.Credits(4000).String(); got != "CR 4000" {
		t.Errorf("Credits(4000) reads %q", got)
	}
}

// Rank 0 is "not commissioned", not a missing value (p. 6).
func TestRankZeroIsNotCommissioned(t *testing.T) {
	t.Parallel()

	if traveller.Rank(0).Commissioned() {
		t.Error("rank 0 reported itself commissioned")
	}

	if !traveller.Rank(1).Commissioned() {
		t.Error("rank 1 reported itself uncommissioned")
	}
}

func TestSkillReads(t *testing.T) {
	t.Parallel()

	// P. 25's own skill list records the weapon, not the category.
	skill := traveller.Skill{Name: "Dagger", Level: 1}
	if got := skill.String(); got != "Dagger-1" {
		t.Errorf("Skill reads %q, want %q", got, "Dagger-1")
	}
}

func TestThrowEventTotalsDiceAndModifier(t *testing.T) {
	t.Parallel()

	// Jamison's first survival throw: 5+ with a DM of +2 for intelligence,
	// he rolls 11 (p. 24).
	throw := traveller.ThrowEvent{Dice: []int{5, 6}, DM: 2}
	if got := throw.Total(); got != 13 {
		t.Errorf("Total() = %d, want 13", got)
	}

	if got := (traveller.ThrowEvent{}).Total(); got != 0 {
		t.Errorf("an empty throw totals %d, want 0", got)
	}
}
