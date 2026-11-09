package rules

import (
	"errors"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// The lift validates the shape of every cell, so these are the tests that a
// malformed table fails loudly rather than lifting into a plausible wrong
// value. Every case
// is one the reprint's font could produce: a target whose sign became a
// digit, a dash that became a 4, a name that lost a glyph.

func TestNamesAreCheckedAgainstTheType(t *testing.T) {
	t.Parallel()

	_, err := parseCharacteristic("Charisma")
	if err == nil {
		t.Error("Charisma was accepted as a characteristic")
	}

	_, err = parseTitle("emperor/empress")
	if err == nil {
		t.Error("emperor/empress was accepted; Book 3 p. 22 stops at duke/duchess")
	}

	// The names the type gives itself are the ones a transcription must use.
	for _, c := range traveller.Characteristics {
		got, err := parseCharacteristic(c.String())

		if err != nil || got != c {
			t.Errorf("%v does not parse from its own name: %v", c, err)
		}
	}
}

func TestParseAlteration(t *testing.T) {
	t.Parallel()

	characteristic, size, ok, err := parseAlteration("-1 Social Standing")
	if err != nil || !ok || characteristic != traveller.SocialStanding || size != -1 {
		t.Errorf("-1 Social Standing lifted to %v %d (%v, %v)", characteristic, size, ok, err)
	}

	// A keyboard types a minus as an ASCII hyphen, and the minus the page
	// prints is U+2212, the minus sign. Someone transcribing
	// p. 11 types what he sees, and the two are indistinguishable on screen,
	// so both must lift the same way.
	minusSign, minusSignSize, ok, err := parseAlteration("\u22121 Social Standing")
	if err != nil || !ok || minusSign != traveller.SocialStanding || minusSignSize != -1 {
		t.Errorf("an alteration typed with U+2212 lifted to %v %d (%v, %v)",
			minusSign, minusSignSize, ok, err)
	}

	_, _, ok, err = parseAlteration("Gunnery")

	if ok || err != nil {
		t.Errorf("Gunnery read as an alteration (%v, %v)", ok, err)
	}

	badCharacteristic := func(cell string) error {
		_, _, _, parseErr := parseAlteration(cell)

		return parseErr
	}

	err = badCharacteristic("+1 Charisma")
	if err == nil {
		t.Error("an alteration to a characteristic that does not exist was accepted")
	}
}

func TestParseDM(t *testing.T) {
	t.Parallel()

	dm, err := parseDM(2, "Intelligence 7+")
	if err != nil || dm.Amount != 2 || dm.Characteristic != traveller.Intelligence {
		t.Errorf("Intelligence 7+ lifted to %+v (%v)", dm, err)
	}

	for _, condition := range []string{"Intelligence", "Charisma 7+", "Intelligence seven", "Intelligence +7"} {
		_, err := parseDM(1, condition)
		if err == nil {
			t.Errorf("%q was accepted as a die modifier", condition)
		}
	}
}

func TestParseBenefitRowRefusesWhatNoRowPrints(t *testing.T) {
	t.Parallel()

	names := map[string]string{"Low Psg": "Low Passage"}

	_, err := parseBenefitRow("Yacht", names)
	if err == nil {
		t.Error("Yacht was accepted as a benefit")
	}

	// The font trap in one line: a dash that extracted as a 4 must not lift
	// as anything at all.
	_, err = parseBenefitRow("4", names)
	if err == nil {
		t.Error("a bare 4 lifted as a benefit; that is what a dash cell extracts as")
	}
}

func TestLiftThrow(t *testing.T) {
	t.Parallel()

	_, err := liftThrow(printedThrow{target: "eight"}, "survival")
	if err == nil {
		t.Error("a target that is not a number lifted")
	}

	// The reprint's font turns a printed minus into a digit, so an N- target
	// extracts as N3. Whatever else that is, it is not the printed throw.
	got, err := liftThrow(printedThrow{target: "83"}, "survival")

	if err != nil || got.Target.Number() != 83 {
		t.Errorf("83 lifted to %v (%v); the guard against this is that nothing is read from extracted text", got.Target, err)
	}

	_, err = liftThrow(printedThrow{target: "8+", dms: []printedDM{{dm: 1, condition: "Wisdom 8+"}}}, "x")
	if err == nil {
		t.Error("a modifier on a characteristic that does not exist lifted")
	}
}

// Loading is memoized, so the tables are lifted and validated once.
func TestLoadIsOnce(t *testing.T) {
	t.Parallel()

	first, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	second, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if first != second {
		t.Error("Load lifted the tables twice")
	}
}

// The page abbreviates characteristics wherever a table is narrow, and each
// abbreviation reads as the one characteristic it begins: p. 9's "+1 Intel",
// p. 10's "Educ 9+", p. 11's "+1 Stren". The full names, as p. 4 prints
// them and the Aging Table repeats them, read too. Anything else is refused.
func TestThePagesCharacteristicAbbreviationsRead(t *testing.T) {
	t.Parallel()

	for printed, want := range map[string]traveller.Characteristic{
		"Stren": traveller.Strength, "Dext": traveller.Dexterity, "Endur": traveller.Endurance,
		"Intel": traveller.Intelligence, "Educ": traveller.Education, "Social": traveller.SocialStanding,
		"Strength": traveller.Strength, "Social Standing": traveller.SocialStanding,
	} {
		got, err := parseCharacteristic(printed)
		if err != nil || got != want {
			t.Errorf("%q read as %v (%v), want %v", printed, got, err, want)
		}
	}

	for _, printed := range []string{"Intl", "Str", "Edu", "Soc", "intel", "Intel "} {
		_, err := parseCharacteristic(printed)
		if err == nil {
			t.Errorf("%q was read as a characteristic", printed)
		}
	}
}

// An em dash is what the page prints where a cell means nothing, and it
// means nothing wherever it stands. A hyphen or an en dash in its place is
// refused by name: the hyphen is what was typed before the page's text was
// restored (#339), and the en dash is the glyph #288 refuses everywhere.
func TestAnEmDashBenefitIsNothing(t *testing.T) {
	t.Parallel()

	benefit, err := parseBenefitRow("—", map[string]string{})
	if err != nil || benefit != (traveller.NoBenefit{}) {
		t.Errorf("an em dash benefit read as %v (%v)", benefit, err)
	}

	for _, typed := range []string{"-", "–"} {
		_, err := parseBenefitRow(typed, map[string]string{})
		if err == nil {
			t.Errorf("a benefit cell of %q was read", typed)
		}
	}
}

// An em dash throw is a throw the service does not have, and an em dash DM
// beside a printed throw is a DM it does not have.
func TestAnEmDashThrowOrDMIsNothing(t *testing.T) {
	t.Parallel()

	none := printedThrow{target: "—", dms: []printedDM{{dm: 1, condition: "—"}}}

	throw, printed, err := liftOptionalThrow(none, "commission")
	if err != nil || printed || throw.Target != (traveller.Target{}) {
		t.Errorf("an em dash throw read as %v, printed %v (%v)", throw, printed, err)
	}

	dashes := printedThrow{target: "3+", dms: []printedDM{{dm: 1, condition: "—"}, {dm: 2, condition: "—"}}}

	throw, err = liftThrow(dashes, "enlistment")
	if err != nil || len(throw.DMs) != 0 {
		t.Errorf("em dash DMs read as %v (%v)", throw.DMs, err)
	}

	for _, typed := range []string{"-", "–"} {
		_, _, refused := liftOptionalThrow(printedThrow{target: typed}, "commission")
		if refused == nil ||
			!strings.Contains(refused.Error(), "the page prints an em dash") && !errors.Is(refused, traveller.ErrEnDash) {
			t.Errorf("a commission of %q was not refused by name: %v", typed, refused)
		}

		mistyped := printedThrow{target: "3+", dms: []printedDM{{dm: 1, condition: typed}}}

		_, refused = liftThrow(mistyped, "enlistment")
		if refused == nil ||
			!strings.Contains(refused.Error(), "the page prints an em dash") && !errors.Is(refused, traveller.ErrEnDash) {
			t.Errorf("a DM of %q was not refused by name: %v", typed, refused)
		}
	}

	// A throw the page does not print has no DM either.
	beside := printedThrow{target: "—", dms: []printedDM{{dm: 1, condition: "Intel 6+"}}}

	_, _, err = liftOptionalThrow(beside, "commission")
	if err == nil {
		t.Error("a DM beside a dashed commission was read")
	}
}

// A column of the Table of Ranks names each rank from Rank 1 and prints a
// dash for each the service does not have; a name below a dash, a hyphen, or
// nothing at all is refused.
func TestAnEmDashRankIsNone(t *testing.T) {
	t.Parallel()

	ranks, err := liftRanks([rankRows]string{"4th Officer", "3rd Officer", "2nd Officer", "1st Officer", "Captain", "—"})
	if err != nil || len(ranks) != 5 {
		t.Errorf("Merchants' ranks read as %v (%v)", ranks, err)
	}

	for name, cells := range map[string][rankRows]string{
		"a hyphen for a rank": {"Ensign", "-", "—", "—", "—", "—"},
		"a rank after a dash": {"Ensign", "—", "Commander", "—", "—", "—"},
		"a rank left empty":   {"Ensign", "", "—", "—", "—", "—"},
	} {
		_, err := liftRanks(cells)
		if err == nil {
			t.Errorf("%s was read", name)
		}
	}
}
