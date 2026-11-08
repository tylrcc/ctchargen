// Package scripted is the roller a test drives: the dice and draws of one
// path, handed out in order.
//
// A script is strict both ways. It states every die and every draw of its
// path, so a die or draw asked for past its end fails the test naming the
// roller and the count, and one the procedure never asked for is reported by
// Finished. Either way a script that no longer walks its path says which
// throw moved, rather than answering with a value nobody chose.
//
// It is test support, imported only by test files: the chargen fixtures and
// the command's walks both drive their runs through it (#280). When Finished
// runs is the caller's: a run that stops partway by design leaves its tail
// unread, and only the caller knows whether it did.
package scripted

import (
	"slices"
	"testing"
)

// Roller hands out a script's dice and draws in order. It satisfies
// chargen.Roller.
type Roller struct {
	t     testing.TB
	name  string
	dice  []int
	among []int
	die   int // dice handed out so far
	draw  int // draws handed out so far
}

// New is a roller for the script name states: every die, in the order the
// procedure throws them, and every draw among alternatives, as an index into
// the offer. It keeps its own copy, so the caller's slices are never read
// again.
func New(tb testing.TB, name string, dice, among []int) *Roller {
	tb.Helper()

	return &Roller{t: tb, name: name, dice: slices.Clone(dice), among: slices.Clone(among)}
}

// Die is the script's next die.
func (r *Roller) Die() int {
	r.t.Helper()

	if r.die == len(r.dice) {
		r.t.Fatalf("%s: the procedure asked for die %d, and the script holds %d",
			r.name, r.die+1, len(r.dice))
	}

	face := r.dice[r.die]
	r.die++

	return face
}

// TwoDice is the script's next two dice.
func (r *Roller) TwoDice() (int, int) {
	r.t.Helper()

	first := r.Die()
	second := r.Die()

	return first, second
}

// Among is the script's next draw, which must index one of n alternatives.
func (r *Roller) Among(n int) int {
	r.t.Helper()

	if r.draw == len(r.among) {
		r.t.Fatalf("%s: the procedure asked for draw %d, and the script holds %d",
			r.name, r.draw+1, len(r.among))
	}

	index := r.among[r.draw]
	r.draw++

	if index >= n {
		r.t.Fatalf("%s: draw %d is index %d of %d alternatives", r.name, r.draw, index, n)
	}

	return index
}

// Finished fails the test if the procedure left any of the script unread,
// naming how much and where it starts.
func (r *Roller) Finished() {
	r.t.Helper()

	if r.die < len(r.dice) {
		r.t.Errorf("%s: the procedure finished with %d of the script's %d dice unread, from die %d",
			r.name, len(r.dice)-r.die, len(r.dice), r.die+1)
	}

	if r.draw < len(r.among) {
		r.t.Errorf("%s: the procedure finished with %d of the script's %d draws unread, from draw %d",
			r.name, len(r.among)-r.draw, len(r.among), r.draw+1)
	}
}
