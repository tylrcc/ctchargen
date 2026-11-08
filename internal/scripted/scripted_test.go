package scripted_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/philoserf/ctchargen/internal/scripted"
)

// recording is a testing.TB that keeps its failures instead of failing, so a
// test can hold the roller to failing the test it is handed. Fatalf ends the
// goroutine it is called on, as the real one does.
type recording struct {
	testing.TB

	failures []string
}

func (r *recording) Helper() {}

// Errorf keeps the failure. Only the goroutine on runs a roller on calls it,
// and on reads the failures after that goroutine ends.
func (r *recording) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func (r *recording) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
	runtime.Goexit()
}

// on runs play against a roller on its own goroutine, so a fatal failure ends
// that goroutine and not the test, and returns what the roller reported.
func on(t *testing.T, dice, among []int, play func(*scripted.Roller)) []string {
	t.Helper()

	failed := &recording{TB: t}
	roller := scripted.New(failed, "the path", dice, among)

	done := make(chan struct{})

	go func() {
		defer close(done)

		play(roller)
	}()

	<-done

	return failed.failures
}

// A script is strict both ways: a die or draw past its end fails, naming the
// count, and one left unread when the procedure finishes is reported, naming
// how many and where they start.
func TestAScriptIsStrictBothWays(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		dice, among []int
		play        func(*scripted.Roller)
		want        string
	}{
		"a die past the end": {
			dice: []int{3}, play: func(r *scripted.Roller) { r.TwoDice() },
			want: "the path: the procedure asked for die 2, and the script holds 1",
		},
		"a draw past the end": {
			play: func(r *scripted.Roller) { r.Among(2) },
			want: "the path: the procedure asked for draw 1, and the script holds 0",
		},
		"a draw outside the offer": {
			among: []int{2}, play: func(r *scripted.Roller) { r.Among(2) },
			want: "the path: draw 1 is index 2 of 2 alternatives",
		},
		"dice left unread": {
			dice: []int{3, 4, 5}, play: func(r *scripted.Roller) { r.Die(); r.Finished() },
			want: "the path: the procedure finished with 2 of the script's 3 dice unread, from die 2",
		},
		"draws left unread": {
			among: []int{0, 1}, play: func(r *scripted.Roller) { r.Finished() },
			want: "the path: the procedure finished with 2 of the script's 2 draws unread, from draw 1",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			failures := on(t, tc.dice, tc.among, tc.play)
			if len(failures) != 1 || failures[0] != tc.want {
				t.Errorf("reported %q, want exactly %q", failures, tc.want)
			}
		})
	}
}

// A script read to its end reports nothing, and hands out what it states in
// order: the same script read twice is the same path both times, whatever the
// caller does to its own slices afterwards.
func TestAScriptReadToItsEndIsSilent(t *testing.T) {
	t.Parallel()

	dice, among := []int{2, 6, 1}, []int{1}

	var read string

	failures := on(t, dice, among, func(r *scripted.Roller) {
		dice[0], among[0] = 5, 0

		first, second := r.TwoDice()

		read = fmt.Sprint(first, second, r.Die(), r.Among(3))
		r.Finished()
	})

	if len(failures) != 0 {
		t.Errorf("a script read to its end reported %q", failures)
	}

	if read != "2 6 1 1" {
		t.Errorf("read %q, want the script as given, 2 6 1 1", read)
	}
}
