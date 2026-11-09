package chargen_test

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/render"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// The goldens. They are regenerated, never hand-edited: run
// `go test ./internal/chargen -regenerate` and read the diff. The roster that
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
		t.Fatalf("reading %s: %v (run go test ./internal/chargen -regenerate)", path, err)
	}

	if string(got) != string(want) {
		t.Errorf("%s has changed. If that was intended, regenerate it and read the diff:\n"+
			"  go test ./internal/chargen -regenerate", path)
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
//
// Implements: The two documented examples are generated, not written
// [Documents].
func documentedExample(t *testing.T, name string, encoded []byte) {
	t.Helper()

	path := filepath.Join("..", "..", "docs", name)

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
//
// Holds: Six 2D rolls, in order: values 2–12 initially.
// Holds: All characters begin at 18.
// Holds: One enlistment attempt, against the Prior Service Table.
// Holds: Per-term order is the exposition's.
// Holds: Reenlistment throw every term, no DMs.
// Holds: A commission or promotion confers one skill eligibility.
// Holds: The skills table is designated before the die.
// Holds: Blade and gun combat name a weapon at once.
// Holds: Skills accumulate as Skill-1, Skill-2, with no cap.
// Holds: Saving throws in the table's row order.
// Holds: The mustering-out table is designated before the die.
// Holds: The +1 with gambling on Table 2.
// Holds: Every step, throw, choice and consequence, in order.
// Holds: Each choice names what was offered and what was chosen.
// Holds: A record names the shape it is, so the v1 freeze has something to
// attach to.
// Holds: The two documented examples are generated, not written.
func TestGoldens(t *testing.T) {
	// withReadings is the fixture whose transcript is also kept with its
	// readings. It names E002, whose prose carries the page's own italic, so
	// the golden shows that emphasis reaching the transcript unchanged.
	const withReadings = "army-serve"

	t.Parallel()

	written := map[string]bool{}
	keptWithReadings := false

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

		// The sheet and the transcript are rendered from the record pinned
		// above, so the record on disk renders to these goldens (#363).
		sheet, err := render.Sheet(encoded)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		transcript, err := render.Transcript(encoded, false)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		golden(t, fixture.name+".sheet.md", []byte(sheet))
		golden(t, fixture.name+".transcript.md", []byte(transcript))

		if fixture.name == withReadings {
			full, err := render.Transcript(encoded, true)
			if err != nil {
				t.Fatalf("%s: %v", fixture.name, err)
			}

			golden(t, fixture.name+".readings.transcript.md", []byte(full))

			keptWithReadings = true
		}
	}

	if !keptWithReadings {
		t.Errorf("%s keeps the transcript with its readings but is not in the fixture roster", withReadings)
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
//
// Holds: A fixture generates one golden, and the same one twice.
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
		// for four of them.
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

// numbered is the sequence number a transcript line opens with: an event line
// ("  2. Strength: rolled 5+5") or a step heading ("## 17. term 3").
var numbered = regexp.MustCompile(`(?m)^(?:## | *)(\d+)\. `)

// No golden's transcript skips a number (#249).
//
// The live view numbered its step headings because a watched session whose
// headings printed bare appeared to skip - 17, 18, then 20 - and the kept
// transcript printed them bare still, so army-serve's opened at event 2.
// Every event is one numbered line, in order, from 1 to the last.
//
// Holds: The kept transcript numbers every event, so no number is skipped.
func TestNoTranscriptSkipsANumber(t *testing.T) {
	t.Parallel()

	for _, fixture := range fixtures {
		character := fixture.generate(t)

		encoded, err := render.JSON(character)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		transcript, err := render.Transcript(encoded, false)
		if err != nil {
			t.Fatalf("%s: %v", fixture.name, err)
		}

		var seen []int

		for _, match := range numbered.FindAllStringSubmatch(transcript, -1) {
			n, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatalf("%s: %v", fixture.name, err)
			}

			seen = append(seen, n)
		}

		for i, n := range seen {
			if n != i+1 {
				t.Errorf("%s: the transcript's line %d is numbered %d, want %d", fixture.name, i+1, n, i+1)

				break
			}
		}

		if len(seen) != len(character.Events) {
			t.Errorf("%s: %d numbered lines for %d events", fixture.name, len(seen), len(character.Events))
		}
	}
}
