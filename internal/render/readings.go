package render

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/ctchargen/internal/rules"
)

// Reading writes one reading in full, as markdown: its heading, then each
// labelled part in the order a referee argues it. The prose is the reading's
// own, the page's emphasis included, and nothing is reworded on the way out.
// A withdrawn reading says so before anything else, with why, and names the
// erratum that replaced it where one did.
func Reading(r rules.Reading) string {
	var out strings.Builder

	fmt.Fprintf(&out, "### %v — %s\n\n", r.ID, r.Title)

	if r.Withdrawn() {
		fmt.Fprintf(&out, "**Withdrawn.** %s", r.Withdrawal.Why)

		if r.Withdrawal.ReplacedBy != 0 {
			fmt.Fprintf(&out, " Replaced by %v.", r.Withdrawal.ReplacedBy)
		}

		out.WriteString("\n\n")
	}

	for i, part := range []struct{ label, text string }{
		{"Pages", r.Pages},
		{"What the page says", r.PageSays},
		{"The gap", r.Gap},
		{"Reading", r.Reading},
		{"Why", r.Why},
		{"Stamped on", r.StampedOn},
	} {
		if i > 0 {
			out.WriteString("\n")
		}

		fmt.Fprintf(&out, "**%s.** %s\n", part.label, part.text)
	}

	return out.String()
}

// writeReadings writes the transcript's last section: each reading the record
// names, in full, in id order. A record this build wrote lists its errata in
// that order already; one written elsewhere is sorted here. A record
// written by another build may name an erratum this one has no reading for,
// and that is said, not refused: the rest of the transcript is still the
// record's.
//
// Implements: A transcript with its readings ends with each reading its record
// names, in full [CLI sketch].
func writeReadings(out *strings.Builder, errata []string) {
	out.WriteString("## Readings\n\n")

	if len(errata) == 0 {
		out.WriteString("The record names no reading.\n\n")

		return
	}

	for _, id := range slices.Sorted(slices.Values(errata)) {
		r, ok := rules.ReadingNamed(id)
		if !ok {
			fmt.Fprintf(out, "### %s\n\nThe record names %s, and this build has no reading of that id.\n\n", id, id)

			continue
		}

		out.WriteString(Reading(r) + "\n")
	}
}

// ErrNoReading is an id that names no reading this build has.
var ErrNoReading = errors.New("names no reading")

// ReadingsIndex lists every reading as its erratum, its title and whether it
// is in force or withdrawn, one row each in erratum order.
//
// Implements: The readings are listed, and those asked for are written in full
// [CLI sketch].
func ReadingsIndex() string {
	var out strings.Builder

	out.WriteString("# Readings\n\n| Erratum | Title | Status |\n| --- | --- | --- |\n")

	for _, r := range rules.Readings() {
		status := "in force"
		if r.Withdrawn() {
			status = "withdrawn"
		}

		fmt.Fprintf(&out, "| %v | %s | %s |\n", r.ID, r.Title, status)
	}

	return out.String()
}

// ReadingsNamed writes the readings the ids name, each in full, in the order
// given. The text is composed before it is returned, so the first id that
// names no reading fails the whole and nothing reaches the caller.
//
// Implements: The readings are listed, and those asked for are written in full
// [CLI sketch].
func ReadingsNamed(ids []string) (string, error) {
	var out strings.Builder

	out.WriteString("# Readings\n\n")

	for i, id := range ids {
		r, ok := rules.ReadingNamed(id)
		if !ok {
			return "", fmt.Errorf("%q %w", id, ErrNoReading)
		}

		if i > 0 {
			out.WriteString("\n")
		}

		out.WriteString(Reading(r))
	}

	return out.String(), nil
}
