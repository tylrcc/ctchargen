package traveller

import "fmt"

// Erratum identifies one recorded reading from rules/data/ERRATA.md.
//
// Every place the held text is silent, ambiguous, or self-contradictory has
// a reading recorded there with its page cite and its stamping condition,
// and the reading is named on every record it governed. A reading applied
// with neither an entry nor a stamp is the one defect no comparison between
// the documents and the code can find.
//
// Ids are stable: never reused, never renumbered. A withdrawn reading keeps
// its constant and its heading.
//
// The rules package lifts ERRATA.md and holds its entries and these
// constants to each other in both directions: a constant with no entry, or
// an entry with no constant, fails the lift.
type Erratum int

// The fifteen readings of rules/data/ERRATA.md.
const (
	// E001: p. 5 prints both "may submit to the draft" and "must submit to
	// the draft"; may governs, so declining ends generation with a
	// civilian.
	E001 Erratum = iota + 1

	// E002: the exposition's per-term order governs over the Jamison
	// example's, which rolls reenlistment before that term's skills.
	E002

	// E003: a 12 on the reenlistment throw recurs without limit past
	// term 7, not once as p. 7's singular "an additional term" reads.
	E003

	// E004: the fatal term counts, so a character who dies in term N is
	// recorded at N terms and age 18 + 4N.
	E004

	// E005: a service-wide rank and service skill is granted once, on
	// entering the service, not once per term.
	E005

	// E006: the aging round is the last step of the term, after the
	// reenlistment throw, and is read off the table's term row.
	E006

	// E007: aging saving throws are thrown in the table's row order, and a
	// characteristic reaching zero resolves its crisis inline.
	E007

	// E008: failing the medical crisis saving throw is death.
	E008

	// E009: no DM applies to the medical crisis throw during generation —
	// there is no attending medical personnel to modify it.
	E009

	// E010: aging reductions floor at 0; every other alteration floors
	// at 1.
	E010

	// E011: title eligibility is assessed once at the end of generation
	// against final Social Standing, for every character including the
	// dead, who are assessed but not asked.
	E011

	// E012: printed names are normalized to the headings their
	// descriptions carry. Stamped on no record.
	E012

	// E013: no promotion throw is made at the top of a service's Table of
	// Ranks.
	E013

	// E014: the Aging Table's last column is terminal, governing every
	// term from the fourteenth on.
	E014

	// E015: where the worked example's stated result contradicts the table
	// it illustrates, the table governs.
	E015
)

// Errata is every recorded reading, in id order, for iteration and for the
// gate.
var Errata = [...]Erratum{
	E001, E002, E003, E004, E005, E006, E007,
	E008, E009, E010, E011, E012, E013, E014, E015,
}

// String is the erratum's id, as ERRATA.md's heading prints it.
func (e Erratum) String() string { return fmt.Sprintf("E%03d", int(e)) }
