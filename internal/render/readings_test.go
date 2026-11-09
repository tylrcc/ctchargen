package render_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/render"
	"github.com/philoserf/ctchargen/internal/rules"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// A reading is written as its heading and each labelled part, in the order a
// referee argues it: where, what the page says, the gap, the reading, why,
// and which records it governs. Its prose goes through untouched, the page's
// own emphasis included, and a withdrawn reading says so first, with why and
// what replaced it. The readings here are test-only, so the expected text is
// typed rather than taken from the values.
func TestAReadingIsWrittenInFull(t *testing.T) {
	t.Parallel()

	standing := rules.Reading{
		ID: traveller.E002, Title: "A _short_ title", Pages: "Book 1 p. 5.",
		PageSays: "\"Some words.\"", Gap: "None said.", Reading: "_May._ Two\nlines.",
		Why: "One.\n\nTwo.", StampedOn: "Records.",
	}

	withdrawn := standing

	withdrawn.Withdrawal = rules.Withdrawal{Why: "A later decision replaced it.", ReplacedBy: traveller.E005}

	unreplaced := standing

	unreplaced.Withdrawal = rules.Withdrawal{Why: "The page settles it."}

	body := "**Pages.** Book 1 p. 5.\n\n" +
		"**What the page says.** \"Some words.\"\n\n" +
		"**The gap.** None said.\n\n" +
		"**Reading.** _May._ Two\nlines.\n\n" +
		"**Why.** One.\n\nTwo.\n\n" +
		"**Stamped on.** Records.\n"

	for name, tc := range map[string]struct {
		reading rules.Reading
		want    string
	}{
		"standing": {standing, "### E002 — A _short_ title\n\n" + body},
		"withdrawn and replaced": {withdrawn, "### E002 — A _short_ title\n\n" +
			"**Withdrawn.** A later decision replaced it. Replaced by E005.\n\n" + body},
		"withdrawn": {unreplaced, "### E002 — A _short_ title\n\n" +
			"**Withdrawn.** The page settles it.\n\n" + body},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := render.Reading(tc.reading); got != tc.want {
				t.Errorf("got\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}

// With its readings, a note keeps every inline id and ends, before the lines
// that say what wrote it and what to keep, with each reading its generation
// record names, in full and in id order. One that names none says so.
//
// Holds: A note asked for with its readings carries each reading its
// generation record names, in full, before its closing lines.
func TestANoteWithItsReadingsEndsWithThem(t *testing.T) {
	t.Parallel()

	e001, _ := rules.ReadingNamed("E001")
	e011, _ := rules.ReadingNamed("E011")

	for name, tc := range map[string]struct {
		errata []traveller.Erratum
		want   string
	}{
		"two readings": {
			[]traveller.Erratum{traveller.E001, traveller.E011},
			"## Readings\n\n" + render.Reading(e001) + "\n" + render.Reading(e011) + "\n",
		},
		"none": {nil, "## Readings\n\nThe generation record names no reading.\n\n"},
		"out of order": {
			[]traveller.Erratum{traveller.E011, traveller.E001},
			"## Readings\n\n" + render.Reading(e001) + "\n" + render.Reading(e011) + "\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			character := &chargen.Character{
				Age:        traveller.NewAge(18),
				Enlistment: traveller.DeclinedTheDraft{},
				Errata:     tc.errata,
				Events: []traveller.Event{traveller.OutcomeEvent{
					Seq: 1, Description: "drafted", Errata: []traveller.Erratum{traveller.E001},
				}},
			}

			plain := render.Note(character, false)
			got := render.Note(character, true)

			body, footer, found := strings.Cut(plain, "---\n\n")
			if !found {
				t.Fatalf("the plain note has no footer:\n%s", plain)
			}

			if want := body + tc.want + "---\n\n" + footer; got != want {
				t.Errorf("got\n%s\nwant\n%s", got, want)
			}

			if !strings.Contains(got, "[E001]") {
				t.Errorf("the inline id is gone:\n%s", got)
			}
		})
	}
}

// The index is every reading's erratum, title and status, one row each in
// erratum order. No real reading is withdrawn, so every row is in force.
//
// Holds: The readings are listed, and those asked for are written in full.
func TestTheIndexListsEveryReading(t *testing.T) {
	t.Parallel()

	var want strings.Builder

	want.WriteString("# Readings\n\n| Erratum | Title | Status |\n| --- | --- | --- |\n")

	for _, e := range traveller.Errata {
		want.WriteString("| " + e.String() + " | " + rules.Readings()[e-1].Title + " | in force |\n")
	}

	if got := render.ReadingsIndex(); got != want.String() {
		t.Errorf("got\n%s\nwant\n%s", got, want.String())
	}
}

// The readings named are written in full in the order given, and an id that
// names none fails the whole, naming the id.
//
// Holds: The readings are listed, and those asked for are written in full.
func TestTheReadingsNamedAreWrittenInFull(t *testing.T) {
	t.Parallel()

	e011, _ := rules.ReadingNamed("E011")
	e002, _ := rules.ReadingNamed("E002")

	got, err := render.ReadingsNamed([]string{"E011", "E002"})
	if err != nil {
		t.Fatal(err)
	}

	if want := "# Readings\n\n" + render.Reading(e011) + "\n" + render.Reading(e002); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}

	for _, id := range []string{"E099", "e002", "E02", ""} {
		text, err := render.ReadingsNamed([]string{"E002", id})
		if !errors.Is(err, render.ErrNoReading) || !strings.Contains(err.Error(), `"`+id+`"`) || text != "" {
			t.Errorf("%q: %q, %v", id, text, err)
		}
	}
}
