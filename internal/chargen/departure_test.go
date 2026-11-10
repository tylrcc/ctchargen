package chargen_test

import (
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// firstRetiringTerm is p. 7's "retire any time after the end of the 5th
// term", typed out here rather than read from the engine. The engine's own
// constant is what this checks; a test that read it would agree with it
// however wrong both were.
const firstRetiringTerm = 5

// The three departures a living character can have, as this test names them.
const (
	retired    = "retired"
	forcedOut  = "forced out"
	discharged = "discharged"
)

// P. 21 decides a departure by the term count, and nothing else does.
//
// "A character who leaves the service at the end of the 5th or later term of
// service is considered to have retired." So a character who left alive at or
// past that term is Retired, and one who left alive below it never is - he is
// Discharged if he chose to go and ForcedOut if the service refused him.
//
// This is here because until #45 the choice between the last two was made by
// comparing the log's own sentence against the literal "reenlistment denied",
// so editing the wording changed the domain outcome. Only the goldens held
// it, and a golden holds an invariant by accident of who is on the roster.
//
// Holds: A departure below term 5 is forced out if the service refused him,
// discharged if he chose to go.
func TestADepartureIsDecidedByTheTermCount(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}

	for _, fixture := range fixtures {
		departedByTheBook(t, fixture.name, fixture.generate(t), seen)
	}

	// A roster reaching only one of the three would pass every check above
	// while proving nothing about the other two.
	for _, want := range []string{retired, forcedOut, discharged} {
		if !seen[want] {
			t.Errorf("no fixture departs %q, so that branch is unchecked", want)
		}
	}
}

// departedByTheBook holds one character's departure to p. 21, and records
// which of the three it was so the caller can see the roster reached them all.
func departedByTheBook(
	t *testing.T, name string, character *chargen.Character, seen map[string]bool,
) {
	t.Helper()

	var how string

	switch character.Departure().(type) {
	case traveller.Retired:
		how = retired
	case traveller.ForcedOut:
		how = forcedOut
	case traveller.Discharged:
		how = discharged
	default:
		// A death, or a civilian who never joined. Neither is decided by
		// the term count.
		return
	}

	seen[how] = true

	if (how == retired) != (character.Terms() >= firstRetiringTerm) {
		t.Errorf("%s: %s after %d terms; p. 21 makes a living departure at or past "+
			"the 5th a retirement, and one below it never one", name, how, character.Terms())
	}

	if how == retired {
		// A retirement is decided by the term count alone, whether the
		// character chose to go or the service refused to keep him.
		return
	}

	kept, threw := lastReenlistment(character)
	if !threw {
		t.Errorf("%s: %s without a reenlistment throw", name, how)

		return
	}

	// Below the retiring term the two part on one thing: whether the
	// service would have him. P. 6's throw is what says so.
	if (how == forcedOut) == kept {
		t.Errorf("%s: %s after a reenlistment throw that %s",
			name, how, map[bool]string{true: "succeeded", false: "failed"}[kept])
	}
}

// reenlistmentStep is the step name the engine logs a reenlistment throw
// under, typed out here rather than read from the engine for the reason the
// term count is: a test that read it would agree with it however wrong both
// were.
const reenlistmentStep = "reenlistment"

// lastReenlistment reports whether the character's final reenlistment throw
// succeeded, and whether he made one at all.
//
// It is the last that decides a departure: an earlier failure would have
// ended the service already, and a 12 exactly overrides the character's own
// intent without ending anything (p. 6).
func lastReenlistment(character *chargen.Character) (bool, bool) {
	succeeded, found := false, false

	for _, event := range character.Events {
		throw, isThrow := event.(traveller.ThrowEvent)
		if !isThrow || throw.Step != reenlistmentStep {
			continue
		}

		succeeded, found = throw.Succeeded, true
	}

	return succeeded, found
}

// A character who declined the draft entered no service, and nothing on the
// record says otherwise.
//
// The zero value of traveller.ServiceName is Navy, so before #42 a civilian
// carried Navy in a Service field and only a Served flag beside it kept him
// from reading as a sailor. ServedIn folds the Enlistment instead, and there
// is no second field left to disagree with it.
//
// Holds: A civilian entered no service, and the record names none.
func TestACivilianEnteredNoService(t *testing.T) {
	t.Parallel()

	var civilians int

	for _, fixture := range fixtures {
		character := fixture.generate(t)

		_, declined := character.Enlistment().(traveller.DeclinedTheDraft)
		if !declined {
			continue
		}

		civilians++

		service, served := character.ServedIn()
		if served {
			t.Errorf("%s: declined the draft and served in the %v", fixture.name, service)
		}
	}

	if civilians == 0 {
		t.Error("no fixture declines the draft, so this checks nothing")
	}
}
