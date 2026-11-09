package main

import (
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/scripted"
)

// The walks the command tests take. Each is a scripted list of dice and
// draws, extracted once from a record the tool wrote, and named for the
// path it reaches; the test that drives one passes it in through withDice.
// A walk is read against the inputs its comment names, and nothing else:
// another service or another set of answers asks for different dice. No
// record is checked in for a walk; the proof it still reaches its path is
// the assertion the test that drives it makes.
//
// A walk is consumed exactly, as the scripted package says. A walk a person
// answers scripts no draws, because a person, not a draw, answers the
// choices the policy would draw for.

// otherWalked is Other, forced, answered 1 at every question: four terms,
// discharged. It is the walk most interactive tests take, so their event
// numbering is fixed.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var otherWalked = walk{
	name: "otherWalked",
	dice: []int{
		6, 1, 5, 1, 2, 2, 5, 2, 3, 4, 4, 3, 2, 3, 5, 3, 4, 2, 6, 6,
		1, 4, 3, 2, 5, 5, 6, 1, 2, 5, 4, 6, 5, 1, 3, 4, 5, 6, 4, 2,
		5, 2, 4, 1, 6,
	},
	among: nil,
}

// otherByPolicy is the same Other under the default policy: five terms. It is
// what the interactive walk is compared against.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var otherByPolicy = walk{
	name: "otherByPolicy",
	dice: []int{
		6, 1, 5, 1, 2, 2, 5, 2, 3, 4, 4, 3, 2, 3, 5, 3, 4, 2, 6, 6,
		1, 4, 3, 2, 5, 5, 6, 1, 2, 5, 4, 6, 5, 3, 4, 5, 6, 4, 2, 5,
		2, 4, 1, 6, 1, 2, 5, 1, 4, 1, 3, 6, 4, 4, 3, 6, 6,
	},
	among: []int{1},
}

// navyWalked is the Navy, answered 1 at every question: seven terms, with
// Education past 8 so that a term offers all four skills tables.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var navyWalked = walk{
	name: "navyWalked",
	dice: []int{
		5, 5, 3, 3, 2, 6, 6, 4, 5, 5, 6, 6, 5, 1, 2, 5, 2, 3, 2, 1,
		3, 5, 4, 6, 6, 6, 3, 2, 1, 3, 5, 4, 6, 3, 5, 3, 2, 6, 4, 3,
		1, 4, 6, 6, 3, 4, 5, 6, 2, 5, 4, 6, 4, 3, 5, 2, 5, 6, 5, 1,
		6, 3, 1, 2, 6, 6, 3, 5, 2, 5, 1, 2, 1, 4, 2, 6, 2, 5, 4, 4,
		1, 4, 6, 6, 5, 6, 3, 2, 2, 2, 4, 6, 4, 4, 2, 5, 4, 3, 2, 5,
		6, 4, 4, 1,
	},
	among: nil,
}

// draftedWalked is rejected by the Navy, drafted into the Marines, and
// answered 1 at every question: seven terms to a rank that allows the +1 on
// Table 1, and a title to assume. Every yes-or-no question the procedure puts
// is put on this walk.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var draftedWalked = walk{
	name: "draftedWalked",
	dice: []int{
		3, 4, 1, 5, 6, 5, 6, 1, 5, 6, 5, 5, 3, 2, 2, 3, 4, 4, 4, 2,
		6, 2, 4, 2, 2, 2, 4, 3, 5, 3, 2, 6, 6, 2, 6, 4, 1, 3, 6, 4,
		1, 3, 6, 4, 3, 5, 1, 5, 4, 6, 1, 6, 2, 1, 3, 1, 4, 1, 3, 6,
		3, 2, 2, 4, 6, 5, 3, 3, 4, 4, 6, 3, 5, 5, 6, 1, 6, 6, 5, 2,
		3, 2, 6, 3, 6, 4, 2, 4, 1, 1, 4, 5, 1, 1, 6, 4, 3, 1, 6, 1,
		6, 4, 3,
	},
	among: nil,
}

// gamblerWalked is Other, answered 2 at every question: the Service Skills
// Table every time, which is where Gambling comes from, so mustering out
// offers the +1 the expertise allows.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var gamblerWalked = walk{
	name: "gamblerWalked",
	dice: []int{
		2, 3, 1, 4, 4, 2, 6, 3, 6, 3, 1, 3, 5, 2, 6, 6, 2, 6, 3, 5,
		5,
	},
	among: nil,
}

// navyRearmed is the Navy, answered 1 at every question, mustering out with a
// Blade Combat benefit twice: the first takes a Dagger, so the second's menu
// offers "+1 expertise in the Dagger" as its twelfth line, and it is the
// thirtieth question asked. A menu line throws no dice, so the walk is the
// same whichever line that question is answered with.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var navyRearmed = walk{
	name: "navyRearmed",
	dice: []int{
		6, 2, 3, 4, 6, 4, 5, 4, 5, 5, 6, 5, 3, 2, 1, 2, 2, 2, 6, 3,
		6, 3, 5, 1, 5, 5, 5, 4, 6, 5, 2, 4, 2, 6, 3, 6, 6, 3, 5, 5,
		5, 5, 3, 4, 2, 4, 3, 6, 4, 1, 5, 6, 3, 5, 2, 2, 1, 4, 6, 3,
		6, 3, 1, 5, 1, 2, 5, 3, 1, 2, 5, 2, 6, 3, 2, 4, 6, 1, 6, 6,
		3, 4, 2, 3, 6, 4, 5, 6, 5,
	},
	among: nil,
}

// scoutKilled is a Scout killed by the survival throw in his first term.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var scoutKilled = walk{
	name: "scoutKilled",
	dice: []int{
		2, 2, 1, 2, 2, 6, 6, 1, 5, 6, 4, 5, 4, 3, 1, 4,
	},
	among: nil,
}

// otherKilled is Other, killed by the survival throw in his fifth term under
// the default policy.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var otherKilled = walk{
	name: "otherKilled",
	dice: []int{
		5, 4, 2, 1, 1, 3, 2, 4, 6, 6, 2, 3, 5, 6, 3, 6, 4, 6, 5, 2,
		5, 1, 3, 4, 5, 1, 6, 4, 5, 2, 1, 5, 1, 3, 5, 2, 5, 5, 5, 5,
		3, 1, 2,
	},
	among: []int{3},
}

// navyOfficer is the Navy under --career oneterm: commissioned Lieutenant in
// his one term, then forced out.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var navyOfficer = walk{
	name: "navyOfficer",
	dice: []int{
		6, 4, 6, 2, 3, 2, 1, 2, 2, 4, 6, 3, 6, 4, 5, 4, 6, 5, 6, 5,
		6, 1, 6, 4, 2, 1, 6, 6,
	},
	among: nil,
}

// navyDeclined is the Navy under --career oneterm, refused at enlistment and
// declining the draft: an 18-year-old civilian (E001).
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var navyDeclined = walk{
	name:  "navyDeclined",
	dice:  []int{4, 4, 1, 6, 5, 1, 5, 1, 4, 1, 6, 5, 3, 4},
	among: nil,
}

// walk is the dice and draws of one path the command's tests take.
type walk struct {
	name  string
	dice  []int
	among []int
}

// roller plays the walk through a scripted roller. Each call starts the walk
// again, so a batch test can hand the same walk to every member. When the
// test ends, every playing must have been read to its end: a walk states the
// dice of its path, and a die or draw left over is one the path never throws.
func (w walk) roller(t *testing.T) source {
	t.Helper()

	return w.played(t, true)
}

// partly is roller for a test whose run stops partway by design - an
// interactive session cut off, a refusal - so the walk's tail is expected
// to go unread. A site that uses it says why.
func (w walk) partly(t *testing.T) source {
	t.Helper()

	return w.played(t, false)
}

// played hands out a fresh playing of the walk for each roller asked of it,
// and, exactly, checks every playing in the test's cleanup. No test holds the
// cleanup to calling Finished: one that stopped would leave every test green,
// which is a deleted line for review to see rather than a rule the suite
// keeps (#280).
func (w walk) played(t *testing.T, exactly bool) source {
	t.Helper()

	var playings []*scripted.Roller

	if exactly {
		t.Cleanup(func() {
			if t.Failed() {
				return
			}

			for _, s := range playings {
				s.Finished()
			}
		})
	}

	return func() chargen.Roller {
		s := scripted.New(t, w.name, w.dice, w.among)

		playings = append(playings, s)

		return s
	}
}
