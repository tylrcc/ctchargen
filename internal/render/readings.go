package render

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/ctchargen/internal/rules"
	"github.com/philoserf/ctchargen/internal/traveller"
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

// writeReadings writes each reading the generation record names, in full,
// in id order. Every erratum a character carries is one this build declares,
// so every one has a reading.
//
// Implements: A note asked for with its readings carries each reading its
// generation record names, in full, before its closing lines [Command line].
func writeReadings(out *strings.Builder, errata []traveller.Erratum) {
	out.WriteString("## Readings\n\n")

	if len(errata) == 0 {
		out.WriteString("The generation record names no reading.\n\n")

		return
	}

	for _, erratum := range slices.Sorted(slices.Values(errata)) {
		out.WriteString(Reading(rules.Readings()[erratum-1]) + "\n")
	}
}

// ErrNoReading is an id that names no reading this build has.
var ErrNoReading = errors.New("names no reading")

// ReadingsIndex lists every reading as its erratum, its title and whether it
// is in force or withdrawn, one row each in erratum order.
//
// Implements: The readings are listed, and those asked for are written in full
// [Command line].
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
// [Command line].
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
