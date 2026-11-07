package main

import (
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
)

// The walks the command tests take. Each is a scripted list of dice and
// draws, extracted once from a record the tool wrote, and named for the
// path it reaches; the test that drives one passes it in through withDice.
// A walk is read against the inputs its comment names, and nothing else:
// another service or another set of answers asks for different dice. No
// record is checked in for a walk; the proof it still reaches its path is
// the assertion the test that drives it makes.
//
// A walk is strict, as a fixture is: a die or a draw asked for beyond the
// list fails the test naming it.

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
	among: []int{0},
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
	among: []int{0, 0, 0},
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

// walk is a scripted roller for the command: the dice and draws of one path.
type walk struct {
	name  string
	dice  []int
	among []int
}

// roller hands out the walk's dice in order, failing the test when asked
// for one it does not have. Each call starts the walk again, so a batch
// test can hand the same walk to every member.
func (w walk) roller(t *testing.T) source {
	t.Helper()

	return func() chargen.Roller {
		return &scripted{t: t, name: w.name, dice: slices.Clone(w.dice), among: slices.Clone(w.among)}
	}
}

// scripted is one playing of a walk.
type scripted struct {
	t     *testing.T
	name  string
	dice  []int
	among []int
	die   int
	draw  int
}

func (s *scripted) Die() int {
	s.t.Helper()

	if s.die == len(s.dice) {
		s.t.Fatalf("%s: the procedure asked for die %d, and the walk scripts %d", s.name, s.die+1, len(s.dice))
	}

	face := s.dice[s.die]
	s.die++

	return face
}

func (s *scripted) TwoDice() (int, int) {
	s.t.Helper()

	first := s.Die()
	second := s.Die()

	return first, second
}

func (s *scripted) Among(n int) int {
	s.t.Helper()

	if s.draw == len(s.among) {
		s.t.Fatalf("%s: the procedure asked for draw %d, and the walk scripts %d", s.name, s.draw+1, len(s.among))
	}

	index := s.among[s.draw]
	s.draw++

	if index >= n {
		s.t.Fatalf("%s: draw %d is index %d of %d alternatives", s.name, s.draw, index, n)
	}

	return index
}
