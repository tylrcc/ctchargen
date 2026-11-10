package traveller

import "fmt"

// Erratum identifies one recorded reading.
//
// Every place the held text is silent, ambiguous, or self-contradictory has
// a reading recorded with its page cite and its stamping condition, and the
// reading is stamped on every character it governed. A reading applied with
// no recorded reading and no stamp is the one defect no comparison between
// the readings and the code can find.
//
// An erratum is the authority's own name for a reading, and no note carries
// it: the readings are the authority's record, not the referee's (ADR 0007).
//
// The rules package declares one reading per entry of Errata, with its title
// and its prose: the compiler holds the readings to Errata's length, so a
// reading missing or one too many does not compile, and rules' tests hold
// each to its place. Errata is held to the constants below by a test that
// reads this file, since a const block cannot count itself (#333).
type Erratum int

// The recorded readings, each declared in the rules package with its title.
const (
	E001 Erratum = iota + 1
	E002
	E003
	E004
	E005
	E006
	E007
	E008
	E009
	E010
	E011
	E012
	E013
	E014
	E015
	E016
)

// Errata is every recorded reading, in id order, for iteration and for the
// gate.
//
// Implements: Every erratum constant is in Errata, in order [Domain].
var Errata = [...]Erratum{
	E001, E002, E003, E004, E005, E006, E007,
	E008, E009, E010, E011, E012, E013, E014, E015, E016,
}

// String is the erratum's id: E001.
func (e Erratum) String() string { return fmt.Sprintf("E%03d", int(e)) }
