package traveller

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Modality is how a Target is met — the die roll conventions of Book 1
// pp. 2-3: "A number followed by a plus (such as 8+) indicates that that
// number or greater must be rolled. Similarly, a number followed by a minus
// (such as 3-) indicates that that number or less must be rolled." And:
// "throws are always followed by a sign unless the number must be thrown
// exactly".
type Modality int

// The three modalities (pp. 2-3).
//
// Exactly is first, and deliberately: it makes the zero value of a Target
// "0 exactly", which no throw of dice can meet. A Target that was never set
// therefore fails closed. Were AtLeast the zero value, an unset target would
// read as "0+" and every throw in the procedure would succeed against it.
const (
	Exactly Modality = iota // N, with no sign
	AtLeast                 // N+
	AtMost                  // N-
)

func (m Modality) String() string {
	switch m {
	case Exactly:
		return ""
	case AtLeast:
		return "+"
	case AtMost:
		return "-"
	}

	return fmt.Sprintf("Modality(%d)", int(m))
}

// errTarget reports a target that is not in the notation pp. 2-3 print.
// Every parse failure wraps it and then says what was read.
var errTarget = errors.New("malformed target")

// Target is a throw's target: a number and the modality that says how to
// meet it. It is parsed once, when a table lifts, and never re-parsed at
// throw time.
//
// Implements: Die roll conventions: `N+`, `N−`, exact [1:2–3].
type Target struct {
	number   int
	modality Modality
}

// NewTarget builds a target from a number and a modality.
func NewTarget(number int, modality Modality) Target {
	return Target{number: number, modality: modality}
}

// Sign reads one glyph as the sign of a number: +1 for a plus, and -1 for
// either of the two characters a minus may arrive as: the ASCII hyphen a
// data file is typed with, and U+2212, the minus sign, for the minus the page
// prints. Anything else is no sign, EnDash included.
//
// It is the one place that says which glyphs are signs, so a target and a
// table cell agree on it.
func Sign(glyph rune) (int, bool) {
	switch glyph {
	case '+':
		return 1, true
	case '-', '\u2212':
		return -1, true
	default:
		return 0, false
	}
}

// EnDash is not a minus. It looks like one on screen, so one typed where a
// sign belongs is a slip, and it is refused by name, with ErrEnDash, rather
// than read as a minus.
const EnDash = '\u2013'

// ErrEnDash is wrapped by every refusal of an EnDash.
var ErrEnDash = errors.New("an en dash is not a minus")

// ParseTarget reads a target in the book's own notation: "8+", "3-", "12".
//
// It is deliberately strict about the sign. Pp. 2-3 make the sign the whole
// difference between a throw and an exact requirement, and the reprint's
// font turns a printed minus into a digit under text extraction — an "N-"
// read as "N3" is a target that silently changes. Nothing here should ever
// see extracted text, and a malformed target is a build defect, not a
// runtime condition.
func ParseTarget(s string) (Target, error) {
	runes := []rune(strings.TrimSpace(s))
	if len(runes) == 0 {
		return Target{}, fmt.Errorf("%w: it is empty", errTarget)
	}

	if slices.Contains(runes, EnDash) {
		return Target{}, fmt.Errorf("%w %q: %w", errTarget, s, ErrEnDash)
	}

	modality := Exactly

	if sign, ok := Sign(runes[len(runes)-1]); ok {
		if len(runes) == 1 {
			return Target{}, fmt.Errorf("%w %q: a sign with no number", errTarget, s)
		}

		modality = AtLeast
		if sign < 0 {
			modality = AtMost
		}

		runes = runes[:len(runes)-1]
	}

	// P. 3 puts the sign of a throw after its number and the sign of a
	// modifier before it: "throws are always followed by a sign unless the
	// number must be thrown exactly, and DMs are always preceded by a
	// sign." A leading sign here means a DM was passed off as a target.
	if _, ok := Sign(runes[0]); ok {
		return Target{}, fmt.Errorf("%w %q: a leading sign marks a modifier, not a throw", errTarget, s)
	}

	number, err := strconv.Atoi(string(runes))
	if err != nil {
		return Target{}, fmt.Errorf("%w %q: %w", errTarget, s, err)
	}

	return Target{number: number, modality: modality}, nil
}

// Satisfied reports whether a throw of sum meets the target.
func (t Target) Satisfied(sum int) bool {
	switch t.modality {
	case Exactly:
		return sum == t.number
	case AtLeast:
		return sum >= t.number
	case AtMost:
		return sum <= t.number
	}

	return false
}

// Number is the target's number, without its modality.
func (t Target) Number() int { return t.number }

// Modality is how the target is met.
func (t Target) Modality() Modality { return t.modality }

func (t Target) String() string { return fmt.Sprintf("%d%s", t.number, t.modality) }
