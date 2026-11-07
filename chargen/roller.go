package chargen

import "github.com/philoserf/ctchargen/traveller"

// Roller is the source of the procedure's dice, declared here because this
// is where they are consumed. *dice.Stream satisfies it, and so does the
// scripted roller the tests use to walk a particular path.
type Roller interface {
	// Die returns one six-sided die, for the draft (p. 5), the skills
	// tables (p. 11), a mustering out roll (p. 9), and the months a medical
	// crisis recovery adds (pp. 7-8).
	Die() int

	// TwoDice returns the two dice of a 2D throw, first die then second.
	// P. 10: "All rolls except draft are two-die throws."
	TwoDice() (int, int)

	// Among returns an index into a set of n alternatives, for a choice the
	// book hands to a player without naming any basis to choose on (#34).
	// It is not a throw: no page prints one there.
	Among(n int) int
}

// Vary is the narrow view of a Roller that a decider is given: it may draw
// among the alternatives it was offered, and nothing else.
//
// A decider handed the whole Roller could throw dice the procedure never
// makes, and a die thrown there reaches no event: the record would stop
// carrying every die, and the record is the only way back to a character
// (ADR 0001). This is the only variation a decider may take.
type Vary interface{ Among(n int) int }

// varyOnly is the Vary a decider is handed: a Roller with everything but
// Among taken off it, so the narrowing is the value's and not only the
// parameter's. Passing the Roller itself as a Vary would leave its dynamic
// type behind it, and one type assertion would hand a decider the dice back
// (#136), and a die thrown there reaches no event: the log would stop
// carrying every die, and the log is the only way back to the character.
type varyOnly struct{ roller Roller }

func (v varyOnly) Among(n int) int { return v.roller.Among(n) }

// Throw is one 2D throw against a target, with its modifier.
type throw struct {
	dice      [2]int
	modifier  int
	target    traveller.Target
	succeeded bool
}

func (t throw) total() int { return t.dice[0] + t.dice[1] + t.modifier }

// roll makes a 2D throw against a target and reports whether it was met.
func roll(r Roller, target traveller.Target, modifier int) throw {
	first, second := r.TwoDice()
	t := throw{dice: [2]int{first, second}, modifier: modifier, target: target}

	t.succeeded = target.Satisfied(t.total())

	return t
}
