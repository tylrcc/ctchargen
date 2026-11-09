// Package render writes a generated character as the note a referee keeps:
// the sheet, the generation record, and, when asked for, the readings the
// record names, in markdown (ADR 0006).
//
// It also writes the readings on their own: one in full (Reading), every one
// as an index (ReadingsIndex), and those an id names (ReadingsNamed). That is
// why it imports rules, where the readings are declared, and the only reason.
//
// The domain types are the interface, and the note is written from them
// through one view, built once (viewOf), so the sheet and the generation
// record cannot read the same character two ways.
package render

import (
	"fmt"
	"strings"

	"github.com/philoserf/ctchargen/internal/chargen"
)

// Note is the character as a referee keeps it: the sheet, then the
// generation record, which is every die thrown and every choice made and the
// only way back to the character (ADR 0006). With readings, the full text of
// each reading the record names comes after it, so the ids beside the
// outcomes can be read without the tool to hand.
//
// It ends by saying what wrote it and what to keep. That is last because it
// is not the character: a referee reads the headline and the possessions
// first, and thinks about keeping the note only once he wants this one.
func Note(character *chargen.Character, readings bool) string {
	r := viewOf(character)

	var out strings.Builder

	writeProperties(&out, r)
	writeSheet(&out, r)
	writeGenerationRecord(&out, r)

	if readings {
		writeReadings(&out, r.Errata)
	}

	fmt.Fprintf(&out, "---\n\n%s\n\n%s\n", provenance(r), keepTheNote)

	return out.String()
}

// keepTheNote is the line every note ends with. The dice are not seeded, so
// without this line a note spun, read and liked would be a character
// discarded the moment its generation record was cut from it (ADR 0001).
const keepTheNote = "Keep this note whole: its generation record is the only way back to this character."

// provenance says what the note was generated under, by which build, and who
// answered its choices (ADR 0002): which books its page cites are to, which
// code a referee is reading or filing a report against, and whether a policy
// or a person made the choices.
//
// A note generated in-process by a test names no build, so a golden does not
// differ by machine.
//
// Implements: A note names the ruleset, the build and who answered it
// [Generation record].
func provenance(r view) string {
	generated := "Generated under " + r.Ruleset
	if r.Build != "" {
		generated += " by ctchargen " + r.Build
	}

	if r.Policy == nil {
		return generated + "; every choice answered by the player."
	}

	return fmt.Sprintf("%s; every choice answered by the policy, version %d: career %s, skills %s, muster %s.",
		generated, r.Policy.Version, r.Policy.Career, r.Policy.Skills, r.Policy.Muster)
}

// Title is what a batch names a character's note by: his service and the
// rank he holds, as the headline says them, or "Civilian", and his UPP
// (#400). A batch has no name to give him (#406); a referee names him in
// the vault by his note's name and aliases properties.
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
