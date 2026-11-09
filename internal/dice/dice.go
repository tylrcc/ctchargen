// Package dice draws the die rolls Classic Traveller's character generation
// procedure calls for.
//
// It rolls and it never judges. Whether a throw meets its target is the
// business of a Target (Book 1 pp. 2-3), which lives in the traveller
// package; keeping that out of here is what lets dice depend on nothing.
//
// The stream is not seeded. No two are alike and none can be brought back:
// the record is the only way back to a character (ADR 0001), and it logs
// every die this hands out. Two properties are load-bearing all the same,
// because every fixture's script and every reader of a record rest on them:
//
//   - The die is one of six faces, each as likely as the others.
//   - Among is one index in n, each as likely as the others, for the
//     choices the book leaves to a player without naming a basis.
package dice

import "math/rand/v2"

// faces is a die's faces. Book 1 throws six-sided dice and nothing else.
const faces = 6

// Stream is a source of dice. The zero value is not usable; call New.
type Stream struct {
	r *rand.Rand
}

// New returns a Stream of its own, drawn from the runtime's random source.
func New() *Stream {
	return &Stream{r: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))}
}

// Die returns one six-sided die: 1 through 6. Which throws are one die is
// listed once, on chargen's Roller.
func (s *Stream) Die() int {
	return s.r.IntN(faces) + 1
}

// Among returns an index into a set of n alternatives: 0 through n-1.
//
// It is not a die. The procedure has no throw here - it is drawn where the
// book hands the choice to a player and names no basis for it, so that
// --auto stops answering every such question the same way (#34). A die and a
// modulo would not do: the weapon lists of pp. 12-13 are longer than six, and
// even where they are not, 6 mod n makes the first names likelier.
//
// n must be positive; IntN says so itself.
func (s *Stream) Among(n int) int {
	return s.r.IntN(n)
}
