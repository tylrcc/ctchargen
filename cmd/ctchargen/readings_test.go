package main

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/render"
)

// `readings` writes the index with no ids, and the readings asked for in full
// with ids; an id that names no reading is a usage error that writes nothing.
// What the index and a reading say is render's, and held there.
//
// Holds: The readings are listed, and those asked for are written in full.
func TestReadingsWritesTheIndexOrThoseAskedFor(t *testing.T) {
	t.Parallel()

	const (
		e002 = "E002"
		e011 = "E011"
	)

	if out, _ := mustRun(t, []string{cmdReadings}, nil); out != render.ReadingsIndex() {
		t.Errorf("readings wrote\n%s\nnot the index", out)
	}

	want, err := render.ReadingsNamed([]string{e011, e002})
	if err != nil {
		t.Fatal(err)
	}

	if out, _ := mustRun(t, []string{cmdReadings, e011, e002}, nil); out != want {
		t.Errorf("readings E011 E002 wrote\n%s\nwant\n%s", out, want)
	}

	for _, id := range []string{"E099", "e002", "2"} {
		var written strings.Builder

		err := run([]string{cmdReadings, e002, id}, nil, &written, io.Discard)

		switch {
		case err == nil:
			t.Errorf("readings E002 %s was accepted", id)
		case !errors.Is(err, errUsage):
			t.Errorf("readings E002 %s: %v, want a usage error", id, err)
		case !strings.Contains(err.Error(), id):
			t.Errorf("readings E002 %s: %q does not name the id", id, err)
		case written.Len() > 0:
			t.Errorf("readings E002 %s wrote %q before refusing", id, written.String())
		}
	}
}

// --readings ends a transcript and does nothing else, so where no transcript
// is written it is refused rather than accepted and ignored; with --history,
// on both commands that write one, the transcript ends with its readings,
// and without --readings it ends without them.
//
// Holds: --readings without --history is refused, not ignored.
func TestReadingsNeedsTheTranscript(t *testing.T) {
	t.Parallel()

	path := writeRecord(t)

	for name, args := range map[string][]string{
		"generating":         {cmdNew, flagAuto, flagReadings},
		"generating a sheet": {cmdNew, flagAuto, flagSheet, flagReadings},
		"reading back":       {cmdRender, flagReadings, path},
		// Refused before the file is read, as every other pairing is.
		"reading back nothing": {cmdRender, flagReadings, "no-such-record.json"},
	} {
		err := run(args, nil, io.Discard, io.Discard)
		if err == nil || !errors.Is(err, errUsage) || !strings.Contains(err.Error(), "--history") {
			t.Errorf("%s: %v, want a usage error naming --history", name, err)
		}
	}

	for name, tc := range map[string]struct {
		args    []string
		section bool
	}{
		"generating":                {[]string{cmdNew, flagAuto, flagHistory, flagReadings}, true},
		"reading back":              {[]string{cmdRender, flagHistory, flagReadings, path}, true},
		"reading back, no readings": {[]string{cmdRender, flagHistory, path}, false},
	} {
		out, _ := mustRun(t, tc.args, nil)
		if got := strings.Contains(out, "\n## Readings\n"); got != tc.section {
			t.Errorf("%s: a Readings section %v, want %v", name, got, tc.section)
		}
	}
}
