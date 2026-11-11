// Package render writes a generated character as the note a referee keeps:
// its properties, its heading, the sheet, which opens with the profile line
// and the summary, and the generation record, in markdown (ADR 0006).
//
// It names no erratum. The readings are the authority's record, not the
// referee's (ADR 0007), so it does not import rules, where they are declared.
// And it names the character by no pronoun: a character has no sex in these
// rules (p. 8), and the note says none (#459).
//
// Implements: A note names no erratum [Generation record].
//
// Implements: No note uses a pronoun for its character [Sheet].
//
// The domain types are the interface, and the note is written from them
// through one view, built once (viewOf), so the sheet and the generation
// record cannot read the same character two ways.
package render

import (
	"strings"

	"github.com/philoserf/ctchargen/internal/chargen"
)

// Note is the character as a referee keeps it: its properties, its heading
// and the sheet, which opens with the profile line and the summary, then the
// generation record, which is every die thrown and every choice made and the
// only way back to the character (ADR 0006). It ends with the record. The
// build that wrote it is a property, who made each choice is on that choice's
// line, and the ruleset is the same on every note, so nothing follows the
// record to say them again.
//
// Implements: A note ends with its generation record [Generation record].
func Note(character *chargen.Character) string {
	r := viewOf(character)

	var out strings.Builder

	writeProperties(&out, r)
	writeSheet(&out, r)
	writeGenerationRecord(&out, r)

	// The record's last line ends the note, with no blank line after it.
	return strings.TrimRight(out.String(), "\n") + "\n"
}

// Title is what a batch names a character's note by: his service and the rank
// he holds, as the profile line says them, and his UPP (#400).
// A batch has no name to give him (#406); a referee names him in the vault by
// his note's name and aliases properties.
//
// Implements: A batch member's note is named for its character [Command line].
func Title(character *chargen.Character) string { return titleOf(viewOf(character)) }

// titleOf is a character's title, from the view.
func titleOf(r view) string {
	if r.Service == "" {
		return "Civilian " + r.UPP
	}

	return serving(r) + " " + r.UPP
}
