package rules_test

import (
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/rules"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// Every erratum has its reading, in its place, with no field left empty. E001's
// fields are quoted below, typed rather than taken from the values, so a field
// swapped with its neighbour fails here.
func TestTheReadings(t *testing.T) {
	t.Parallel()

	for _, e := range traveller.Errata {
		t.Run(e.String(), func(t *testing.T) {
			t.Parallel()

			r := rules.Readings()[e-1]
			if r.ID != e {
				t.Errorf("declared as %v", r.ID)
			}

			for field, text := range map[string]string{
				"title": r.Title, "pages": r.Pages, "what the page says": r.PageSays,
				"gap": r.Gap, "reading": r.Reading, "why": r.Why, "stamped on": r.StampedOn,
			} {
				if strings.TrimSpace(text) == "" {
					t.Errorf("%s is empty", field)
				}
			}
		})
	}

	e001 := rules.Readings()[traveller.E001-1]

	for field, tc := range map[string]struct{ got, prefix, suffix string }{
		"title":   {e001.Title, "The draft is _may_", "not _must_"},
		"pages":   {e001.Pages, "Book 1 p. 5", "first under The Draft."},
		"says":    {e001.PageSays, "Enlistment: \"Only one", "submit\nto the draft.\""},
		"gap":     {e001.Gap, "The same page", "has to lose."},
		"reading": {e001.Reading, "_May._", "benefits."},
		"why":     {e001.Why, "The last paragraph under Enlistment", "not\nrequired)."},
		"stamped": {e001.StampedOn, "Records where the enlistment throw", "and those that declined."},
	} {
		if !strings.HasPrefix(tc.got, tc.prefix) || !strings.HasSuffix(tc.got, tc.suffix) {
			t.Errorf("E001 %s is %q, want it to run from %q to %q", field, tc.got, tc.prefix, tc.suffix)
		}
	}
}
