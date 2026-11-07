package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/philoserf/ctchargen/render"
)

// renderRecord reads a record written earlier and writes it as a character
// sheet, or as the generation transcript with --history.
//
// It reads the record, which is the only way back to a character (ADR
// 0001): nothing in it can be brought back by running the tool again, and
// reading is what lets a record written by another build still be shown.
func renderRecord(args []string, out io.Writer) error {
	flags := declareRenderFlags()

	err := flags.set.Parse(args)

	switch {
	case errors.Is(err, flag.ErrHelp):
		return writeHelp(out, renderUsage, flags.set)
	case err != nil:
		return fmt.Errorf("%w: %w; run `ctchargen render --help`", errUsage, err)
	}

	// The flag package stops at the first non-flag argument, and the PRD's
	// CLI sketch says so plainly: flags precede the filename. Someone who
	// puts them after gets more than one positional argument, and deserves
	// to be told which of the two mistakes he made.
	if flags.set.NArg() > 1 {
		return fmt.Errorf("%w: flags precede the filename; %s", errUsage, renderUsage)
	}

	if flags.set.NArg() == 0 {
		return fmt.Errorf("%w: %s", errUsage, renderUsage)
	}

	path := flags.set.Arg(0)

	text, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	rendered, err := renderText(text, *flags.history)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	where, err := openDestination(out, *flags.output, *flags.force)
	if err != nil {
		return err
	}

	return where.write(rendered)
}

// renderFlagSet is render's flag set, declared apart from running the
// command so a test can walk the set it declares.
type renderFlagSet struct {
	set            *flag.FlagSet
	history, force *bool
	output         *string
}

func declareRenderFlags() renderFlagSet {
	set := flag.NewFlagSet("render", flag.ContinueOnError)
	set.SetOutput(io.Discard)

	return renderFlagSet{
		set:     set,
		history: set.Bool("history", false, "write the generation record rather than the sheet"),
		output:  set.String("o", "", "write to this file rather than to standard output"),
		force:   set.Bool("force", false, "replace the output file if it already exists"),
	}
}

func renderText(text []byte, history bool) (string, error) {
	if history {
		rendered, err := render.TranscriptFrom(text)

		return rendered, wrapRender(err)
	}

	rendered, err := render.SheetFrom(text)

	return rendered, wrapRender(err)
}
