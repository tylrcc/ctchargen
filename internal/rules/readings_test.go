package rules_test

import (
	"errors"
	"fmt"
	"slices"
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

			r := rules.ReadingOf(e)
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

	e001 := rules.ReadingOf(traveller.E001)

	for field, tc := range map[string]struct{ got, prefix, suffix string }{
		"title":   {e001.Title, "The draft is _may_", "not _must_"},
		"pages":   {e001.Pages, "Book 1 p. 5", "Draft paragraphs."},
		"says":    {e001.PageSays, "Enlistment: \"Only one", "submit\nto the draft.\""},
		"gap":     {e001.Gap, "The same page", "has to lose."},
		"reading": {e001.Reading, "_May._", "benefits."},
		"why":     {e001.Why, "The Enlistment paragraph", "not\nrequired)."},
		"stamped": {e001.StampedOn, "Records where the enlistment throw", "and those that declined."},
	} {
		if !strings.HasPrefix(tc.got, tc.prefix) || !strings.HasSuffix(tc.got, tc.suffix) {
			t.Errorf("E001 %s is %q, want it to run from %q to %q", field, tc.got, tc.prefix, tc.suffix)
		}
	}
}

// A withdrawal says why, and names as its replacement only an erratum that
// exists and is not its own. No real reading is withdrawn yet, so the cases
// that must be refused are test-only readings, each one real reading with its
// withdrawal broken one way; every real reading is held to the same check.
//
// Holds: A withdrawal says why, and its replacement is another erratum that
// exists.
func TestAWithdrawalNamesARealReplacement(t *testing.T) {
	t.Parallel()

	for _, r := range rules.Readings() {
		err := refusedWithdrawal(r)
		if err != nil {
			t.Errorf("%v: %v", r.ID, err)
		}
	}

	withdrawn := func(w rules.Withdrawal) rules.Reading {
		r := rules.ReadingOf(traveller.E004)

		r.Withdrawal = w

		return r
	}

	const why = "A later decision replaced it."

	for name, w := range map[string]rules.Withdrawal{
		"with no reason, replaced":  {ReplacedBy: traveller.E005},
		"replaced by itself":        {Why: why, ReplacedBy: traveller.E004},
		"replaced by no erratum":    {Why: why, ReplacedBy: traveller.Erratum(len(traveller.Errata) + 1)},
		"replaced by a negative id": {Why: why, ReplacedBy: -1},
		"with a reason of spaces":   {Why: " \n "},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if refusedWithdrawal(withdrawn(w)) == nil {
				t.Errorf("withdrawal %+v was accepted", w)
			}
		})
	}

	for name, w := range map[string]rules.Withdrawal{
		"with no replacement": {Why: "The page settles it after all."},
		"replaced":            {Why: why, ReplacedBy: traveller.E005},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r := withdrawn(w)

			err := refusedWithdrawal(r)
			if err != nil || !r.Withdrawn() {
				t.Errorf("withdrawal %+v: withdrawn %v, refused with %v", w, r.Withdrawn(), err)
			}
		})
	}
}

// errWithdrawal is a withdrawal that does not say why, or names a replacement
// that is no other erratum.
var errWithdrawal = errors.New("malformed withdrawal")

// refusedWithdrawal is why a reading's withdrawal is malformed, or nil.
//
// Implements: A withdrawal says why, and its replacement is another erratum
// that exists [Documents] in The record.
func refusedWithdrawal(r rules.Reading) error {
	w := r.Withdrawal

	switch {
	case !r.Withdrawn():
		return nil
	case strings.TrimSpace(w.Why) == "":
		return fmt.Errorf("%w: %v is withdrawn and does not say why", errWithdrawal, r.ID)
	case w.ReplacedBy == r.ID:
		return fmt.Errorf("%w: %v is replaced by itself", errWithdrawal, r.ID)
	case w.ReplacedBy != 0 && !slices.Contains(traveller.Errata[:], w.ReplacedBy):
		return fmt.Errorf("%w: %v is replaced by %v, which is no erratum", errWithdrawal, r.ID, w.ReplacedBy)
	}

	return nil
}
