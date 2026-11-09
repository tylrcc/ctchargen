package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/philoserf/ctchargen/internal/render"
)

// readings writes the readings: with no ids, every one as its erratum, title
// and status; with ids, those in full, in the order asked. A record and its
// transcript name readings by id, and this is where an id is read out
// without a transcript to hand.
//
// It has no flag set, as version has none: its words are ids. An id that
// names no reading is a usage error, and nothing is written.
//
// Implements: The readings are listed, and those asked for are written in full
// [CLI sketch].
func readings(args []string, out io.Writer) error {
	if len(args) > 0 && isHelpFlag(args[0]) {
		if len(args) > 1 {
			return fmt.Errorf("%w: %s; help takes no arguments, and was given %q",
				errUsage, readingsUsage, args[1])
		}

		return writeUsageText(out, "usage: ctchargen "+readingsUsage+"\n")
	}

	text := render.ReadingsIndex()

	if len(args) > 0 {
		var err error

		text, err = render.ReadingsNamed(args)

		switch {
		case errors.Is(err, render.ErrNoReading):
			return fmt.Errorf("%w: %s; %w; run `ctchargen readings` for the list", errUsage, readingsUsage, err)
		case err != nil:
			return fmt.Errorf("writing the readings: %w", err)
		}
	}

	_, err := io.WriteString(out, text)
	if err != nil {
		return fmt.Errorf("writing the readings: %w", err)
	}

	return nil
}
