package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/philoserf/ctchargen/internal/render"
)

// readings writes the readings: with no ids, every one as its erratum, title
// and status; with ids, those in full, in the order asked. A note's
// generation record names readings by id, and this is where an id is read
// out without the note to hand.
//
// It has no flag set, as version has none: its words are ids. An id that
// names no reading is a usage error, and nothing is written.
//
// Implements: The readings are listed, and those asked for are written in full
// [Command line].
func readings(args []string, out io.Writer) error {
	if asked, err := answerHelp(args, out, readingsUsage); asked {
		return err
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
