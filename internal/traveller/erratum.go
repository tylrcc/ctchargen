package traveller

import "fmt"

// Erratum identifies one recorded reading.
//
// Every place the held text is silent, ambiguous, or self-contradictory has
// a reading recorded with its page cite and its stamping condition,
// and the reading is named on every record it governed. A reading applied
// with neither an entry nor a stamp is the one defect no comparison between
// the documents and the code can find.
//
// Ids are stable: never reused, never renumbered. A withdrawn reading keeps
// its constant and its heading.
//
// The rules package declares one reading per constant, so a reading missing
// or one too many does not compile, and its tests hold each to its constant's
// place. It still lifts internal/rules/data/ERRATA.md, and holds it to those
// readings field by field.
type Erratum int

// The recorded readings.
const (
	// E001 — The draft is _may_, not _must_.
	E001 Erratum = iota + 1

	// E002 — The per-term order is the exposition's, not the example's.
	E002

	// E003 — A 12 on the reenlistment throw recurs past term 7.
	E003

	// E004 — The age of a character who dies in service.
	E004

	// E005 — When service-wide rank and service skills are granted.
	E005

	// E006 — Where the aging round sits within the term.
	E006

	// E007 — The order of saving throws within an aging round.
	E007

	// E008 — The outcome of a failed medical-crisis saving throw.
	E008

	// E009 — The medical-expertise DM during solo generation.
	E009

	// E010 — How far below 1 aging may carry a characteristic.
	E010

	// E011 — When title eligibility is assessed.
	E011

	// E012 — Printed names normalized to the descriptions' headings.
	E012

	// E013 — No promotion throw at the top of the Table of Ranks.
	E013

	// E014 — The Aging Table's last column is terminal.
	E014

	// E015 — A printed table governs over the worked example's stated result.
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
