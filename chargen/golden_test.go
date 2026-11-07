package chargen_test

import (
	"flag"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/render"
	"github.com/philoserf/ctchargen/traveller"
)

// The goldens. They are regenerated, never hand-edited: run
// `go test ./chargen -regenerate` and read the diff. The roster that
// generates them is in fixtures_test.go.
//
//nolint:gochecknoglobals // the flag package has no other shape.
var regenerate = flag.Bool("regenerate", false, "rewrite the golden fixtures")

func golden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", name)

	if *regenerate {
		err := os.WriteFile(path, got, 0o600)
		if err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v (run go test ./chargen -regenerate)", path, err)
	}

	if string(got) != string(want) {
		t.Errorf("%s has changed. If that was intended, regenerate it and read the diff:\n"+
			"  go test ./chargen -regenerate", path)
	}
}

// The two examples that sit beside docs/character.schema.json, written by
// the same path that writes the goldens so they cannot drift from what the
// engine emits.
//
// minimal is the civilian who declined the draft, which is the smallest
// complete record there is. complete is the fullest character the tables
// permit - and it is not complete, because no character can hold both
// Travellers' Aid and a starship: p. 9 prints the membership only in the
// Navy and Marines columns and the ships only in the Scout and Merchant
// columns, and a character serves one service. The schema says so.
//
//nolint:gochecknoglobals // an immutable pair, and Go has no const map.
var documented = map[string]string{
	"scouts-civilian":   "character.minimal.json",
	"merchants-captain": "character.complete.json",
}

// documentedExample writes or checks one of the two examples that sit beside
// the schema, by the same path that writes the goldens.
func documentedExample(t *testing.T, name string, encoded []byte) {
	t.Helper()

	path := filepath.Join("..", "docs", name)

	if *regenerate {
		err := os.WriteFile(path, encoded, 0o600)
		if err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	if string(encoded) != string(want) {
		t.Errorf("%s has drifted from the record it documents; regenerate it", path)
	}
}

// Every fixture's JSON, sheet and transcript, pinned.
func TestGoldens(t *testing.T) {
	t.Parallel()

	written := map[string]bool{}

	for _, fixture := range fixtures {
		character := fixture.generate(t)

		encoded, err := render.JSON(character)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		golden(t, fixture.name+".json", encoded)

		if name, documents := documented[fixture.name]; documents {
			documentedExample(t, name, encoded)

			written[fixture.name] = true
		}

		sheet, err := render.Sheet(character)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		transcript, err := render.Transcript(character)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		golden(t, fixture.name+".sheet.md", []byte(sheet))
		golden(t, fixture.name+".transcript.md", []byte(transcript))
	}

	// Without this, renaming or dropping a fixture that documents an
	// example leaves the example behind: nothing regenerates it, nothing
	// compares it, and the schema's published record silently stops being
	// one the engine emits.
	for name := range documented {
		if !written[name] {
			t.Errorf("%s documents an example but is not in the fixture roster", name)
		}
	}
}

// Every golden generates the same bytes twice in one process, from its own
// fixture.
//
// That is same-process determinism, and it is less than this comment used to
// claim (#60). "Generated from its own fixture" is carried by TestGoldens,
// which compares against the file on disk - written by an earlier process,
// in an earlier build. The pair is adequate; the claim was on the wrong half
// of it.
func TestGoldensRegenerate(t *testing.T) {
	t.Parallel()

	for _, f := range fixtures {
		first, err := render.JSON(f.generate(t))
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}

		second, err := render.JSON(f.generate(t))
		if err != nil {
			t.Fatalf("%s: %v", f.name, err)
		}

		if string(first) != string(second) {
			t.Errorf("%s did not generate the same record twice from its fixture", f.name)
		}
	}
}

// Every recorded reading is reachable by some fixture.
//
// This is the gate the PRD asks for, and until the engine walked the ranked
// services it could not hold. Two readings are stamped on no record by
// design: E012, a spelling, which is a transcription rather than a reading
// and changes nothing about a character; and E015, which settles the worked
// example against the table it illustrates, and which only the replay of
// that example depends on.
func TestEveryReadingIsReachable(t *testing.T) {
	t.Parallel()

	stamped := make(map[traveller.Erratum]bool)

	for _, f := range fixtures {
		for _, erratum := range f.generate(t).Errata {
			stamped[erratum] = true
		}
	}

	// The two readings that name no record. E012 is a spelling, which
	// changes nothing about a character; E015 settles the worked example
	// against a table, and only the replay of that example depends on it.
	namesNoRecord := []traveller.Erratum{traveller.E012, traveller.E015}

	for _, erratum := range traveller.Errata {
		if slices.Contains(namesNoRecord, erratum) {
			if stamped[erratum] {
				t.Errorf("%v is stamped on a record; ERRATA.md says it names none", erratum)
			}

			continue
		}

		if !stamped[erratum] {
			t.Errorf("%v is reachable by no fixture on the roster", erratum)
		}
	}
}

// pathsReached names every path a single character walked, which is what
// the roster is checked against below.
func pathsReached(character *chargen.Character, reached map[string]bool) {
	mark := func(when bool, path string) {
		if when {
			reached[path] = true
		}
	}

	_, drafted := character.Enlistment.(traveller.Drafted)
	_, dead := character.Departure.(traveller.KilledBySurvivalThrow)

	_, served := character.ServedIn()

	mark(!served, "civilian")
	mark(drafted, "draftee")
	mark(dead, "death")
	mark(character.Nobility.Eligible, "title")
	mark(character.Age.Months() > 0, "months on an age")
	mark(len(character.Benefits.Weapons) > 0, "a weapon benefit")
	mark(len(character.Benefits.Weapons) > 1, "a second weapon benefit")
	mark(character.Rank.Commissioned(), "a commission")
	mark(drafted && character.Rank.Commissioned(), "a draftee commissioned later")
	mark(character.Rank > 1, "a promotion")
	mark(character.RankName != "", "a rank title")
	mark(character.Pension > 0, "retirement pay")

	for _, ship := range character.Benefits.Ships {
		mark(ship.Kind == traveller.FreeTrader, "a Free Trader")
		mark(ship.Kind == traveller.ScoutShip, "a scout ship")
		mark(ship.Kind == traveller.FreeTrader && ship.Years > 0, "a Free Trader received twice")
	}

	mark(character.DuplicateShips > 0, "a scout ship received twice")

	for _, erratum := range character.Errata {
		// A slice and not a map keyed by Erratum: a map keyed by the enum
		// claims to be a complete table of every reading, and this looks
		// for four of fifteen.
		for _, watched := range []struct {
			erratum traveller.Erratum
			path    string
		}{
			{traveller.E005, "service grant"},
			{traveller.E006, "aging"},
			{traveller.E008, "medical crisis"},
			{traveller.E013, "the top of a rank column"},
		} {
			mark(erratum == watched.erratum, watched.path)
		}
	}
}

// The roster must reach every path it claims to, or a fixture that stopped
// reaching one would read exactly like a fixture that still does.
func TestTheRosterReachesItsPaths(t *testing.T) {
	t.Parallel()

	reached := map[string]bool{}
	for _, f := range fixtures {
		pathsReached(f.generate(t), reached)
	}

	for _, path := range []string{
		"civilian", "draftee", "death", "title", "service grant",
		"medical crisis", "aging", "months on an age",
		"a weapon benefit", "a second weapon benefit",
		"a commission", "a draftee commissioned later", "a promotion",
		"a rank title", "retirement pay",
		"a Free Trader", "a Free Trader received twice",
		"a scout ship", "a scout ship received twice",
		"the top of a rank column",
	} {
		if !reached[path] {
			t.Errorf("no fixture reaches %q", path)
		}
	}
}
