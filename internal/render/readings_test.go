package render_test

import (
	"errors"
	"strings"
	"testing"

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

// With its readings, a transcript keeps every inline id and ends, before the
// line that says to keep the record, with each reading the record names, in
// full and in id order. A record from another build may name an erratum this
// one has no reading for, and the section says so rather than failing. A
// record that names none says that too.
//
// Holds: A transcript with its readings ends with each reading its record
// names, in full.
func TestATranscriptWithItsReadingsEndsWithThem(t *testing.T) {
	t.Parallel()

	const keep = "---\n\nKeep the JSON record: it is the only way back to this character.\n"

	e001, _ := rules.ReadingNamed("E001")
	e011, _ := rules.ReadingNamed("E011")

	for name, tc := range map[string]struct {
		errata, want string
	}{
		"two readings": {
			`["E001","E011"]`,
			"## Readings\n\n" + render.Reading(e001) + "\n" + render.Reading(e011) + "\n",
		},
		"one this build does not know": {
			`["E001","E099"]`,
			"## Readings\n\n" + render.Reading(e001) + "\n" +
				"### E099\n\nThe record names E099, and this build has no reading of that id.\n\n",
		},
		"none": {`[]`, "## Readings\n\nThe record names no reading.\n\n"},
		"out of order": {
			`["E011","E001"]`,
			"## Readings\n\n" + render.Reading(e001) + "\n" + render.Reading(e011) + "\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			record := minimalRecord(`"errata":` + tc.errata + `,` +
				`"events":[{"seq":1,"kind":"outcome","description":"drafted","errata":["E001"]}]`)

			plain, err := render.TranscriptFrom(record, false)
			if err != nil {
				t.Fatal(err)
			}

			got, err := render.TranscriptFrom(record, true)
			if err != nil {
				t.Fatal(err)
			}

			body, ok := strings.CutSuffix(plain, keep)
			if !ok {
				t.Fatalf("the plain transcript does not end with the keep line:\n%s", plain)
			}

			if want := body + tc.want + keep; got != want {
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
		want.WriteString("| " + e.String() + " | " + rules.ReadingOf(e).Title + " | in force |\n")
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
