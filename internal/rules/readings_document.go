package rules

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// readingsFile is the readings document: hand-written, read on GitHub as
// prose. The readings are declared in readings.go; the document is lifted
// here only so that a malformed one still stops the binary, and its tests
// hold every field of it to those values while both exist.
const readingsFile = "data/ERRATA.md"

// entryHeading is an entry's heading: "### E001 — The draft is _may_, ...".
// Every level-three heading of the document is an entry, so one that does
// not match is a malformed entry and not prose.
var entryHeading = regexp.MustCompile(`^### (E\d{3}) — (\S.*)$`)

// label is a bold lead-in and the field it opens.
type label struct {
	names []string
	field func(*Reading) *string
}

// labels are the bold lead-ins that open an entry's fields, in the order
// every entry prints them. An entry resting on one page says "Page.".
var labels = []label{
	{[]string{"**Pages.**", "**Page.**"}, func(r *Reading) *string { return &r.Pages }},
	{[]string{"**What the page says.**"}, func(r *Reading) *string { return &r.PageSays }},
	{[]string{"**The gap.**"}, func(r *Reading) *string { return &r.Gap }},
	{[]string{"**Reading.**"}, func(r *Reading) *string { return &r.Reading }},
	{[]string{"**Why.**"}, func(r *Reading) *string { return &r.Why }},
	{[]string{"**Stamped on.**"}, func(r *Reading) *string { return &r.StampedOn }},
}

// errata indexes the Erratum constants by id, the way an entry's heading
// spells it.
var errata = named(traveller.Errata[:])

// liftReadings reads the embedded readings document and refuses one that will
// not lift. What it lifts is discarded: the readings are the values in
// readings.go, and the tests hold the document to them.
func (r *Rules) liftReadings() error {
	text, err := files.ReadFile(readingsFile)
	if err != nil {
		return fmt.Errorf("%w: reading %s: %w", errMalformed, readingsFile, err)
	}

	_, err = parseReadings(string(text))

	return err
}

// parseReadings parses the readings document's entries. The index above them
// and the determinate findings below them are prose and are not read.
//
// It fails in both directions, naming the id: an entry whose id is no
// Erratum constant, and a constant with no entry. Every problem is reported,
// not only the first, because a misspelt id is both at once.
func parseReadings(text string) ([len(traveller.Errata)]Reading, error) {
	var (
		lifted [len(traveller.Errata)]Reading
		found  = make(map[traveller.Erratum]bool)
		errs   []error
	)

	for _, section := range sections(text) {
		reading, err := liftEntry(section)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		if found[reading.ID] {
			errs = append(errs, fmt.Errorf("%w: %s: %v has more than one entry; an id is never reused",
				errMalformed, readingsFile, reading.ID))

			continue
		}

		found[reading.ID] = true
		lifted[reading.ID-1] = reading
	}

	for _, e := range traveller.Errata {
		if !found[e] {
			errs = append(errs, fmt.Errorf("%w: %s: %v has no entry", errMalformed, readingsFile, e))
		}
	}

	return lifted, errors.Join(errs...)
}

// sections splits the document at its level-three headings, each running to
// the next level-two or level-three heading, without the rule and blank lines
// that close it. Only those two levels bound an entry, so a line of prose that
// happens to start with a hash does not cut a field short.
func sections(text string) [][]string {
	lines := strings.Split(text, "\n")

	var all [][]string

	start := -1

	for i, line := range lines {
		if !strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			continue
		}

		if start >= 0 {
			all = append(all, closed(lines[start:i]))
		}

		start = -1
		if strings.HasPrefix(line, "### ") {
			start = i
		}
	}

	if start >= 0 {
		all = append(all, closed(lines[start:]))
	}

	return all
}

// closed is a section without the blank lines and the rule that end it.
func closed(section []string) []string {
	for len(section) > 1 {
		last := strings.TrimSpace(section[len(section)-1])
		if last != "" && last != "---" {
			break
		}

		section = section[:len(section)-1]
	}

	return section
}

// liftEntry reads one entry: its heading, then each label in order, each
// field running to the next label.
func liftEntry(section []string) (Reading, error) {
	heading := entryHeading.FindStringSubmatch(section[0])
	if heading == nil {
		return Reading{}, fmt.Errorf("%w: %s: heading %q is not \"### Ennn — title\"",
			errMalformed, readingsFile, section[0])
	}

	id, ok := errata[heading[1]]
	if !ok {
		return Reading{}, fmt.Errorf("%w: %s: %s has an entry and no Erratum constant",
			errMalformed, readingsFile, heading[1])
	}

	reading := Reading{ID: id, Title: heading[2]}
	next := 0

	var field *string

	for _, line := range section[1:] {
		index, name := labelAt(line)

		switch {
		case index < 0:
			if field == nil && strings.TrimSpace(line) != "" {
				return Reading{}, fmt.Errorf("%w: %s: %v: text before the first label", errMalformed, readingsFile, id)
			}

			if field != nil {
				*field += "\n" + line
			}
		case index != next:
			return Reading{}, fmt.Errorf("%w: %s: %v: %s where %s belongs", errMalformed, readingsFile, id,
				labels[index].names[0], labels[next].names[0])
		default:
			field = labels[index].field(&reading)
			*field = strings.TrimPrefix(line, name)
			next++
		}
	}

	if next != len(labels) {
		return Reading{}, fmt.Errorf("%w: %s: %v has no %s", errMalformed, readingsFile, id, labels[next].names[0])
	}

	for _, l := range labels {
		f := l.field(&reading)

		*f = strings.TrimSpace(*f)
		if *f == "" {
			return Reading{}, fmt.Errorf("%w: %s: %v: %s is empty", errMalformed, readingsFile, id, l.names[0])
		}
	}

	return reading, nil
}

// labelAt is which label a line opens with and how it is spelt, or -1 for a
// line of prose.
func labelAt(line string) (int, string) {
	for i, l := range labels {
		for _, name := range l.names {
			if strings.HasPrefix(line, name) {
				return i, name
			}
		}
	}

	return -1, ""
}
