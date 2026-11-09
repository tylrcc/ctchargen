package dice_test

import (
	"testing"

	"github.com/philoserf/ctchargen/internal/dice"
)

// A die is 1 through 6, and every face comes up. Nothing else is a die.
func TestDieStaysInRangeAndReachesEveryFace(t *testing.T) {
	t.Parallel()

	s := dice.New()
	seen := [7]bool{}

	for i := range 6000 {
		got := s.Die()
		if got < 1 || got > 6 {
			t.Fatalf("draw %d: Die() = %d, want 1..6", i, got)
		}

		seen[got] = true
	}

	for face := 1; face <= 6; face++ {
		if !seen[face] {
			t.Errorf("face %d never appeared in 6000 draws; the die does not reach it", face)
		}
	}
}

// Among reaches every alternative and nothing past them.
func TestAmongReachesEveryAlternative(t *testing.T) {
	t.Parallel()

	const alternatives = 11 // the longer of the two weapon lists (pp. 12-13)

	s := dice.New()
	seen := [alternatives]bool{}

	for i := range 6000 {
		got := s.Among(alternatives)
		if got < 0 || got >= alternatives {
			t.Fatalf("draw %d: Among(%d) = %d, want 0..%d", i, alternatives, got, alternatives-1)
		}

		seen[got] = true
	}

	for index, reached := range seen {
		if !reached {
			t.Errorf("index %d never appeared in 6000 draws; the draw does not reach it", index)
		}
	}
}

// Two streams are two streams. A weak property, but it fails loudly if the
// source ever stops being random.
func TestTwoStreamsDiverge(t *testing.T) {
	t.Parallel()

	a, b := dice.New(), dice.New()
	for range 40 {
		if a.Die() != b.Die() {
			return
		}
	}

	t.Error("two streams produced 40 identical dice; the source is not random")
}
