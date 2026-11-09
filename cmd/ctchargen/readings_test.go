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

// --readings ends the note with the readings its generation record names,
// and without it the note ends without them.
func TestReadingsEndTheNote(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		args    []string
		section bool
	}{
		"with --readings": {[]string{cmdNew, flagAuto, flagReadings}, true},
		"without":         {[]string{cmdNew, flagAuto}, false},
	} {
		out, _ := mustRun(t, tc.args, nil)
		if got := strings.Contains(out, "\n## Readings\n"); got != tc.section {
			t.Errorf("%s: a Readings section %v, want %v", name, got, tc.section)
		}
	}
}
