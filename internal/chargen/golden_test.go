package chargen_test

import (
	"flag"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
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

// Every fixture's note, pinned.
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
func TestGoldens(t *testing.T) {
	// withReadings is the fixture whose note is also kept with its
	// readings. It names E002, whose prose carries the page's own italic, so
	// the golden shows that emphasis reaching the note unchanged.
	const withReadings = "army-serve"

	t.Parallel()

	keptWithReadings := false

	for _, fixture := range fixtures {
		character := fixture.generate(t)

		golden(t, fixture.name+".md", []byte(render.Note(character, false)))

		if fixture.name == withReadings {
			golden(t, fixture.name+".readings.md", []byte(render.Note(character, true)))

			keptWithReadings = true
		}
	}

	if !keptWithReadings {
		t.Errorf("%s keeps its note with its readings but is not in the fixture roster", withReadings)
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
		first, second := render.Note(f.generate(t), false), render.Note(f.generate(t), false)
		if first != second {
			t.Errorf("%s did not generate the same note twice from its fixture", f.name)
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

	// A lost ship is said only by the record (p. 23; #444), so the mark reads
	// it there, by the outcome's prose, typed out here rather than shared with
	// the engine so a reworded outcome fails the roster.
	mark(slices.ContainsFunc(character.Events, func(event traveller.Event) bool {
		outcome, ok := event.(traveller.OutcomeEvent)

		return ok && outcome.Description == "a second scout ship, which is lost"
	}), "a scout ship received twice")

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

// numbered is the sequence number a generation record line opens with: an event line
// ("  2. Strength: rolled 5+5") or a step line ("**17. term 3 (pp. 5-7)**").
var numbered = regexp.MustCompile(`(?m)^(?:\*\*| *)(\d+)\. `)

// No golden's generation record skips a number (#249).
//
// The live view numbered its step headings because a watched session whose
// headings printed bare appeared to skip - 17, 18, then 20 - and the kept
// record printed them bare still, so army-serve's opened at event 2. Every
// event is one numbered line, in order, from 1 to the last. The sheet above
// the record is left out: it numbers nothing.
//
// Holds: The kept generation record numbers every event, so no number is
// skipped.
func TestNoGenerationRecordSkipsANumber(t *testing.T) {
	t.Parallel()

	for _, fixture := range fixtures {
		character := fixture.generate(t)

		_, record, found := strings.Cut(render.Note(character, false), "## Generation record")
		if !found {
			t.Fatalf("%s: the note has no generation record", fixture.name)
		}

		var seen []int

		for _, match := range numbered.FindAllStringSubmatch(record, -1) {
			n, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatalf("%s: %v", fixture.name, err)
			}

			seen = append(seen, n)
		}

		for i, n := range seen {
			if n != i+1 {
				t.Errorf("%s: the generation record's line %d is numbered %d, want %d", fixture.name, i+1, n, i+1)

				break
			}
		}

		if len(seen) != len(character.Events) {
			t.Errorf("%s: %d numbered lines for %d events", fixture.name, len(seen), len(character.Events))
		}
	}
}
