package rules

import (
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// The lift is the validation, so these are the tests that a malformed table
// fails loudly rather than lifting into a plausible wrong value. Every case
// is one the reprint's font could produce: a target whose sign became a
// digit, a dash that became a 4, a name that lost a glyph.

func TestNamesAreCheckedAgainstTheType(t *testing.T) {
	t.Parallel()

	_, err := parseCharacteristic("Charisma")
	if err == nil {
		t.Error("Charisma was accepted as a characteristic")
	}

	_, err = parseService("Marine")
	if err == nil {
		t.Error("Marine was accepted as a service; p. 5 lists Marines")
	}

	_, err = parseSkillTable("Advanced Education")
	if err == nil {
		t.Error("a table name the type does not print was accepted")
	}

	_, err = parseTitle("emperor/empress")
	if err == nil {
		t.Error("emperor/empress was accepted; Book 3 p. 22 stops at duke/duchess")
	}

	// The names the type gives itself are the ones a data file must use.
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

	// The page sets its minus as U+2212 and a data file is typed with an
	// ASCII hyphen. Someone transcribing p. 11 types what he sees, and the
	// two are indistinguishable on screen, so both must lift the same way.
	printed, printedSize, ok, err := parseAlteration("\u22121 Social Standing")
	if err != nil || !ok || printed != traveller.SocialStanding || printedSize != -1 {
		t.Errorf("an alteration typed with the page's own minus lifted to %v %d (%v, %v)",
			printed, printedSize, ok, err)
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

	_, err := liftThrow(nil, "commission")
	if err == nil {
		t.Error("a missing throw lifted")
	}

	_, err = liftThrow(&wireThrow{Target: "eight"}, "survival")
	if err == nil {
		t.Error("a target that is not a number lifted")
	}

	// The reprint's font turns a printed minus into a digit, so an N- target
	// extracts as N3. Whatever else that is, it is not the printed throw.
	got, err := liftThrow(&wireThrow{Target: "83"}, "survival")

	if err != nil || got.Target.Number() != 83 {
		t.Errorf("83 lifted to %v (%v); the guard against this is that nothing is read from extracted text", got.Target, err)
	}

	_, err = liftThrow(&wireThrow{Target: "8+", DMs: []wireDM{{DM: 1, If: "Wisdom 8+"}}}, "x")
	if err == nil {
		t.Error("a modifier on a characteristic that does not exist lifted")
	}
}

func TestEachServiceChecksTheColumns(t *testing.T) {
	t.Parallel()

	full := []string{"Navy", "Marines", "Army", "Scouts", "Merchants", "Other"}
	row := []string{"a", "b", "c", "d", "e", "f"}
	nothing := func(traveller.ServiceName, string) error { return nil }

	err := eachService(full, row, "x", nothing)
	if err != nil {
		t.Fatalf("a well-formed row was refused: %v", err)
	}

	shuffled := []string{"Marines", "Navy", "Army", "Scouts", "Merchants", "Other"}

	err = eachService(shuffled, row, "x", nothing)
	if err == nil {
		t.Error("columns out of the order p. 10 prints them were accepted")
	}

	err = eachService(full[:5], row[:5], "x", nothing)
	if err == nil {
		t.Error("a table with five services was accepted")
	}

	err = eachService(full, row[:5], "x", nothing)
	if err == nil {
		t.Error("a row with a missing cell was accepted")
	}
}

func TestReadRefusesAMissingFile(t *testing.T) {
	t.Parallel()

	_, err := read[wireAging]("nowhere.json")
	if err == nil {
		t.Error("a data file that is not embedded was read")
	}

	_, err = read[wireAging]("weapons.json")

	if err != nil && !strings.Contains(err.Error(), "weapons.json") {
		t.Errorf("an unmarshalling failure does not name its file: %v", err)
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
