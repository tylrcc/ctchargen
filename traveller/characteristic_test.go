package traveller_test

import (
	"testing"

	"github.com/philoserf/ctchargen/traveller"
)

func profile(values ...int) traveller.Profile {
	var p traveller.Profile

	copy(p[:], values)

	return p
}

// P. 4: values "may never exceed 15, and do not go below 1 except for
// calamitous injury or aging." Alter is every alteration that is not aging.
//
// Holds: Values 2–12 initially; never above 15.
func TestAlterHoldsTheOrdinaryFloorAndTheCeiling(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		start int
		delta int
		want  int
	}{
		{"ordinary rise", 7, 1, 8},
		{"ordinary fall", 7, -1, 6},
		{"floors at 1, not 0", 1, -1, 1},
		{"floors at 1 from a big fall", 3, -9, 1},
		{"caps at 15", 15, 1, 15},
		{"caps at 15 from a big rise", 14, 4, 15},
		{"reaches 15 exactly", 13, 2, 15},
	} {
		got := profile(0, 0, 0, 0, 0, tc.start).Alter(traveller.SocialStanding, tc.delta)
		if got[traveller.SocialStanding] != tc.want {
			t.Errorf("%s: %d %+d = %d, want %d",
				tc.name, tc.start, tc.delta, got[traveller.SocialStanding], tc.want)
		}
	}
}

// E010: aging alone may carry a characteristic to 0, which is a medical
// crisis (pp. 7-8). A profile that floored at 1 here would make the crisis
// unreachable, and with it the one path that puts months on an age.
//
// Holds: Floor of 0 for an aging reduction.
func TestAgeReduceFloorsAtZero(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		start int
		by    int
		want  int
	}{
		{"ordinary reduction", 7, 1, 6},
		{"reaches zero", 1, 1, 0},
		{"a 2 taken off a 1 stops at zero", 1, 2, 0},
		{"a 2 taken off a 2 reaches zero", 2, 2, 0},
		{"never goes below zero", 1, 9, 0},
	} {
		got := profile(tc.start).AgeReduce(traveller.Strength, tc.by)
		if got[traveller.Strength] != tc.want {
			t.Errorf("%s: %d less %d = %d, want %d",
				tc.name, tc.start, tc.by, got[traveller.Strength], tc.want)
		}
	}
}

// The two floors are the whole of E010, so they must differ.
//
// Holds: Floor of 1 for every ordinary alteration.
func TestTheTwoFloorsDiffer(t *testing.T) {
	t.Parallel()

	start := profile(1)
	if got := start.Alter(traveller.Strength, -1)[traveller.Strength]; got != 1 {
		t.Errorf("an ordinary alteration took a 1 to %d; E010 floors it at 1", got)
	}

	if got := start.AgeReduce(traveller.Strength, 1)[traveller.Strength]; got != 0 {
		t.Errorf("an aging reduction took a 1 to %d; E010 floors it at 0", got)
	}
}

// Altering one characteristic must not disturb another, and a Profile is a
// value: altering a copy must leave the original alone.
func TestAlterTouchesOneCharacteristicAndCopies(t *testing.T) {
	t.Parallel()

	start := profile(7, 7, 7, 7, 7, 7)
	got := start.Alter(traveller.Education, 1)

	if start[traveller.Education] != 7 {
		t.Error("Alter mutated the profile it was called on")
	}

	for _, c := range traveller.Characteristics {
		want := 7
		if c == traveller.Education {
			want = 8
		}

		if got[c] != want {
			t.Errorf("%v = %d, want %d", c, got[c], want)
		}
	}
}

// P. 8: the UPP is "a string of 6 digits, in the order originally rolled",
// hexadecimal, "the digits 10 through 15 are represented by the letters A
// through F."
//
// Holds: UPP as hexadecimal, in rolled order.
func TestUPP(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		p    traveller.Profile
		want string
	}{
		{"totally average", profile(7, 7, 7, 7, 7, 7), "777777"},
		{"the page's own example", profile(7, 7, 7, 11, 7, 7), "777B77"},
		{"Jamison as rolled", profile(6, 8, 8, 12, 8, 9), "688C89"},
		{"the full hex range", profile(10, 11, 12, 13, 14, 15), "ABCDEF"},
		{"a characteristic at zero", profile(0, 7, 7, 7, 7, 7), "077777"},
	} {
		if got := tc.p.UPP(); got != tc.want {
			t.Errorf("%s: UPP() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// Every value of every closed alphabet must name itself, or the event log
// prints a blank where a reader needs a word.
// namesByKind collects what every value of every closed alphabet calls
// itself, keyed by the alphabet's name.
func namesByKind() map[string][]string {
	named := map[string][]string{}

	add := func(kind string, names ...string) {
		named[kind] = append(named[kind], names...)
	}

	for _, c := range traveller.Characteristics {
		add("Characteristic", c.String())
	}

	for _, s := range traveller.ServiceNames {
		add("ServiceName", s.String())
	}

	for _, s := range traveller.SkillTables {
		add("SkillTable", s.String())
	}

	for _, w := range traveller.WeaponCategories {
		add("WeaponCategory", w.String())
	}

	for _, c := range traveller.ChoicePoints {
		add("ChoicePoint", c.String())
	}

	for _, e := range traveller.Errata {
		add("Erratum", e.String())
	}

	for _, p := range traveller.PassageClasses {
		add("PassageClass", p.String())
	}

	for _, s := range traveller.ShipKinds {
		add("ShipKind", s.String())
	}

	for _, title := range traveller.Titles {
		add("Title", title.String())
	}

	for _, i := range traveller.Intents {
		add("Intent", i.String())
	}

	for _, m := range traveller.MusterTables {
		add("MusterTable", m.String())
	}

	for _, d := range traveller.DecidedBys {
		add("DecidedBy", d.String())
	}

	return named
}

// Every value of every closed alphabet must name itself, and no two may
// share a name, or the event log prints a blank or an ambiguity where a
// reader needs a word.
func TestEveryEnumValueNamesItself(t *testing.T) {
	t.Parallel()

	for kind, names := range namesByKind() {
		seen := map[string]bool{}

		for _, name := range names {
			switch {
			case name == "":
				t.Errorf("%s: a value named itself with an empty string", kind)
			case seen[name]:
				t.Errorf("%s: two values both name themselves %q", kind, name)
			}

			seen[name] = true
		}
	}
}
