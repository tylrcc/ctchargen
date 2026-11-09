package traveller

import "fmt"

// Erratum identifies one recorded reading.
//
// Every place the held text is silent, ambiguous, or self-contradictory has
// a reading recorded with its page cite and its stamping condition, and the
// reading is named on every record it governed. A reading applied with no
// recorded reading and no stamp is the one defect no comparison between the
// readings and the code can find.
//
// An erratum names one decision for good: never reused, never renumbered,
// and a withdrawn reading keeps its constant (ADR 0004).
//
// The rules package declares one reading per constant, with its title and
// its prose, so a reading missing or one too many does not compile, and its
// tests hold each to its constant's place.
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
)

// Errata is every recorded reading, in id order, for iteration and for the
// gate.
var Errata = [...]Erratum{
	E001, E002, E003, E004, E005, E006, E007,
	E008, E009, E010, E011, E012, E013, E014, E015,
}

// String is the erratum's id: E001.
func (e Erratum) String() string { return fmt.Sprintf("E%03d", int(e)) }
