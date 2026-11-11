package chargen_test

import (
	"flag"
	"fmt"
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
// `go test ./internal/chargen -update` once a failure's diff, which it prints
// before anything is regenerated, shows the change was intended. The roster
// that generates them is in fixtures_test.go.
var update = flag.Bool("update", false, "rewrite the golden fixtures")

func golden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", name)

	if *update {
		err := os.WriteFile(path, got, 0o600)
		if err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v (run go test ./internal/chargen -update)", path, err)
	}

	if string(got) != string(want) {
		t.Errorf("%s has changed.\n%s"+
			"If that was intended, regenerate it: go test ./internal/chargen -update",
			path, difference(string(want), string(got)))
	}
}

// diffContext is how many agreeing lines difference shows on each side of
// the lines that changed, and diffLimit how many changed lines it shows from
// each side before it only counts the rest.
const (
	diffContext = 3
	diffLimit   = 20
)

// difference is the one hunk in which a golden and the note generated for
// it part: the lines both share before and after it as context, and between
// them the golden's lines ("-") against the note's ("+"), so a failure can be
// read before anything is regenerated.
func difference(want, got string) string {
	wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")

	first := 0
	for first < len(wantLines) && first < len(gotLines) && wantLines[first] == gotLines[first] {
		first++
	}

	shared := 0
	for shared < len(wantLines)-first && shared < len(gotLines)-first &&
		wantLines[len(wantLines)-1-shared] == gotLines[len(gotLines)-1-shared] {
		shared++
	}

	var diff strings.Builder

	fmt.Fprintf(&diff, "The golden (-) against the note (+), from line %d:\n", first+1)

	lines := func(prefix string, block []string) {
		for i, line := range block {
			if i == diffLimit {
				fmt.Fprintf(&diff, "%s… %d more\n", prefix, len(block)-diffLimit)

				return
			}

			diff.WriteString(prefix + line + "\n")
		}
	}

	wantEnd, gotEnd := len(wantLines)-shared, len(gotLines)-shared

	lines("  ", wantLines[max(0, first-diffContext):first])
	lines("- ", wantLines[first:wantEnd])
	lines("+ ", gotLines[first:gotEnd])
	lines("  ", wantLines[wantEnd:min(len(wantLines), wantEnd+diffContext)])

	return diff.String()
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
	t.Parallel()

	for _, fixture := range fixtures {
		golden(t, fixture.name+".md", []byte(render.Note(fixture.generate(t))))
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
		first, second := render.Note(f.generate(t)), render.Note(f.generate(t))
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

	_, drafted := character.Enlistment().(traveller.Drafted)
	_, dead := character.Departure().(traveller.KilledBySurvivalThrow)

	_, served := character.ServedIn()

	mark(!served, "civilian")
	mark(drafted, "draftee")
	mark(dead, "death")
	mark(character.Nobility().Eligible, "title")
	mark(character.Age().Months() > 0, "months on an age")
	mark(len(character.Benefits().Weapons) > 0, "a weapon benefit")
	mark(len(character.Benefits().Weapons) > 1, "a second weapon benefit")
	mark(character.Rank().Commissioned(), "a commission")
	mark(drafted && character.Rank().Commissioned(), "a draftee commissioned later")
	mark(character.Rank() > 1, "a promotion")
	mark(character.RankName() != "", "a rank title")
	mark(character.Pension() > 0, "retirement pay")

	for _, ship := range character.Benefits().Ships {
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
// ("2. Strength: rolled 5+5") or a step heading ("### 17. term 3 (pp. 5-7)").
var numbered = regexp.MustCompile(`(?m)^(?:#{3,4} )?(\d+)\. `)

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

		_, record, found := strings.Cut(render.Note(character), "## Generation record")
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

// pronoun is a pronoun a sentence about a character would use. A character
// has no sex in these rules (p. 8, "Any character is potentially of any race
// or of either sex"), so the note names none (#459).
var pronoun = regexp.MustCompile(`(?i)\b(he|him|his|himself|she|her|hers|herself)\b`)

// No note uses a pronoun for its character, on the sheet or in the record
// (#459). Every fixture's note is read, so a pronoun any path can print is
// found.
//
// Holds: No note uses a pronoun for its character.
func TestNoNoteUsesAPronoun(t *testing.T) {
	t.Parallel()

	for _, fixture := range fixtures {
		note := render.Note(fixture.generate(t))

		for line := range strings.Lines(note) {
			if found := pronoun.FindString(line); found != "" {
				t.Errorf("%s says %q: %s", fixture.name, found, strings.TrimSpace(line))
			}
		}
	}
}

// listItem is a line a Markdown viewer reads as an ordered list item, and its
// number.
var listItem = regexp.MustCompile(`^(\d+)\. `)

// No viewer renumbers a golden's generation record (#577, #552 §4). A
// Markdown viewer numbers an ordered list from its first item's number, one
// up for each item after, whatever the later items say; a blank line does not
// end the list, and a heading or a paragraph does. So a line numbered other
// than one more than the item before it in the same list is shown with a
// number the line does not carry, and a "from step N" naming it points at
// the wrong line.
//
// Holds: No viewer renumbers the generation record's lines.
func TestNoViewerRenumbersTheGenerationRecord(t *testing.T) {
	t.Parallel()

	for _, fixture := range fixtures {
		_, record, found := strings.Cut(render.Note(fixture.generate(t)), "## Generation record\n")
		if !found {
			t.Fatalf("%s: the note has no generation record", fixture.name)
		}

		previous := 0

		for line := range strings.Lines(record) {
			line = strings.TrimSuffix(line, "\n")

			// A line of spaces is blank to a viewer, and ends nothing.
			if strings.TrimSpace(line) == "" {
				continue
			}

			match := listItem.FindStringSubmatch(line)
			if match == nil {
				previous = 0

				continue
			}

			n, err := strconv.Atoi(match[1])
			if err != nil {
				t.Fatalf("%s: %v", fixture.name, err)
			}

			if previous != 0 && n != previous+1 {
				t.Errorf("%s: %q follows item %d in one list, so a viewer shows it as %d",
					fixture.name, line, previous, previous+1)

				break
			}

			previous = n
		}
	}
}

// heading is a heading of the generation record: its level, and the step it
// heads.
var heading = regexp.MustCompile(`^(#+) (\d+\. .*)$`)

// phase is a step that is part of a term without opening it, with the term
// an aging step names; termEntry is the step that opens a term, and its term.
var (
	phase     = regexp.MustCompile(`^\d+\. (?:skills and training|aging, end of term (\d+))`)
	termEntry = regexp.MustCompile(`^\d+\. term (\d+) \(`)
)

// Every golden's generation record heads each step at the third level, and a
// term's skills and training and its aging at the fourth, under the heading of
// the term they belong to (#577), so a reader landing mid-record finds the
// term in the outline and not by scrolling up.
//
// Holds: The generation record heads each step, a term's phases under the
// term.
func TestEveryGoldenSetsATermsPhasesUnderIt(t *testing.T) {
	t.Parallel()

	for _, fixture := range fixtures {
		character := fixture.generate(t)
		_, record, _ := strings.Cut(render.Note(character), "## Generation record\n")

		term, headed := "", 0

		for line := range strings.Lines(record) {
			line = strings.TrimSuffix(line, "\n")

			match := heading.FindStringSubmatch(line)
			if match == nil {
				continue
			}

			headed++

			level, text := len(match[1]), match[2]

			within := phase.FindStringSubmatch(text)
			if within == nil {
				if level != 3 {
					t.Errorf("%s: %q is at level %d, want 3", fixture.name, line, level)
				}

				term = ""
				if opens := termEntry.FindStringSubmatch(text); opens != nil {
					term = opens[1]
				}

				continue
			}

			if level != 4 || term == "" || (within[1] != "" && within[1] != term) {
				t.Errorf("%s: %q is at level %d under term %q, want level 4 under its own term",
					fixture.name, line, level, term)
			}
		}

		steps := 0

		for _, event := range character.Events {
			if _, isStep := event.(traveller.StepEvent); isStep {
				steps++
			}
		}

		if headed != steps {
			t.Errorf("%s: %d headings for %d steps", fixture.name, headed, steps)
		}
	}
}
