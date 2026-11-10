package traveller

import "fmt"

// ServiceName is one of the six services of the Prior Service Table
// (Book 1 p. 10), in the order that table prints them — which is also
// display order and the order a service draw indexes.
//
// The spelling is p. 5's, where the six are listed together: "Navy, Marines,
// Army, Scouts, Merchants, or Other". The tables spell them three other ways
// between them, which is why E012 normalizes to one.
type ServiceName int

// The six, in Prior Service Table order (p. 10).
const (
	Navy ServiceName = iota
	Marines
	Army
	Scouts
	Merchants
	Other
)

// ServiceNames is the six in book order, for iteration and display.
var ServiceNames = [...]ServiceName{Navy, Marines, Army, Scouts, Merchants, Other}

func (s ServiceName) String() string {
	switch s {
	case Navy:
		return "Navy"
	case Marines:
		return "Marines"
	case Army:
		return "Army"
	case Scouts:
		return "Scouts"
	case Merchants:
		return "Merchants"
	case Other:
		return "Other"
	}

	return fmt.Sprintf("ServiceName(%d)", int(s))
}

// Rank is a character's position in his service's Table of Ranks (p. 10).
//
// Rank 0 is "not commissioned", not a missing value. The highest rank is the
// service's own: the table does not run to the same length in every column,
// and two of the six have no ranks at all (p. 10, "Ranks, commissions, and
// promotions are non-existent in the scout and other services"). So the
// ceiling is data with a cite, in the rules package, not a bound here.
type Rank int

// Commissioned reports whether the rank is one a commission or promotion
// conferred (p. 6).
func (r Rank) Commissioned() bool { return r > 0 }

// Term is a term of service's ordinal, counting from 1 (p. 5).
//
// There is deliberately no upper bound. Seven is the cap on voluntary
// service (p. 7), and a 12 on the reenlistment throw puts a character past
// it (pp. 6-7) — repeatedly, per E003. The cap is a rule applied at a choice
// point; a type that stopped at 7 would reject a character the book permits.
type Term int

// Years is the length of one term of service: "a term of service lasting 4
// years" (p. 5).
//
// Implements: A term is four years [1:5].
const Years = 4

// StartingAge is where every character begins. P. 4: "All characters begin
// the game the same way, untrained, inexperienced, about 18 years of age."
//
// Implements: All characters begin at 18 [1:4].
const StartingAge = 18
