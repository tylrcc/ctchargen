package traveller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// modality is how a Target is met — the die roll conventions of Book 1
// pp. 2-3: "A number followed by a plus (such as 8+) indicates that that
// number or greater must be rolled. Similarly, a number followed by a minus
// (such as 3-) indicates that that number or less must be rolled." And:
// "throws are always followed by a sign unless the number must be thrown
// exactly".
type modality int

// The three modalities (pp. 2-3).
//
// exactly is first, and deliberately: it makes the zero value of a Target
// "0 exactly", which no throw of dice can meet. A Target that was never set
// therefore fails closed. Were atLeast the zero value, an unset target would
// read as "0+" and every throw in the procedure would succeed against it.
const (
	exactly modality = iota // N, with no sign
	atLeast                 // N+
	atMost                  // N-
)

// String is the sign the modality prints after the number, if any. Every
// modality is a case, so the exhaustive linter names one that is added;
// exactly, which prints none, is the return the switch falls to.
func (m modality) String() string {
	switch m {
	case atLeast:
		return "+"
	case atMost:
		return "-"
	case exactly:
	}

	return ""
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
	modality modality
}

// Sign reads one glyph as the sign of a number: +1 for a plus, and -1 for
// either of the two characters a minus may arrive as: the ASCII hyphen a
// keyboard types for one, and U+2212, the minus sign, which is the minus the
// page prints. Anything else is no sign, 0, except EnDash, which is
// ErrEnDash.
//
// It is the one place that says which glyphs are signs, and the one place
// that recognises an EnDash, so a target and every table cell agree on both.
func Sign(glyph rune) (int, error) {
	switch glyph {
	case '+':
		return 1, nil
	case '-', '\u2212':
		return -1, nil
	case EnDash:
		return 0, ErrEnDash
	default:
		return 0, nil
	}
}

// EnDash is not a minus. It looks like one on screen, so one typed where a
// sign belongs is a slip, and it is refused by name, with ErrEnDash, rather
// than read as a minus or as no sign.
const EnDash = '\u2013'

// ErrEnDash is what Sign returns for an EnDash, so every refusal of one read
// where a sign belongs wraps it.
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

	meets := exactly

	sign, err := Sign(runes[len(runes)-1])
	if err != nil {
		return Target{}, fmt.Errorf("%w %q: %w", errTarget, s, err)
	}

	if sign != 0 {
		if len(runes) == 1 {
			return Target{}, fmt.Errorf("%w %q: a sign with no number", errTarget, s)
		}

		meets = atLeast
		if sign < 0 {
			meets = atMost
		}

		runes = runes[:len(runes)-1]
	}

	// P. 3 puts the sign of a throw after its number and the sign of a
	// modifier before it: "throws are always followed by a sign unless the
	// number must be thrown exactly, and DMs are always preceded by a
	// sign." A leading sign here means a DM was passed off as a target.
	leading, err := Sign(runes[0])
	if err != nil {
		return Target{}, fmt.Errorf("%w %q: %w", errTarget, s, err)
	}

	if leading != 0 {
		return Target{}, fmt.Errorf("%w %q: a leading sign marks a modifier, not a throw", errTarget, s)
	}

	number, err := strconv.Atoi(string(runes))
	if err != nil {
		return Target{}, fmt.Errorf("%w %q: %w", errTarget, s, err)
	}

	return Target{number: number, modality: meets}, nil
}

// Satisfied reports whether a throw of sum meets the target. Exactly, the
// zero value, is the return the switch falls to, so an unset target is met
// only by a throw of 0, which two dice cannot make.
func (t Target) Satisfied(sum int) bool {
	switch t.modality {
	case atLeast:
		return sum >= t.number
	case atMost:
		return sum <= t.number
	case exactly:
	}

	return sum == t.number
}

// Number is the target's number, without its modality.
func (t Target) Number() int { return t.number }

func (t Target) String() string { return fmt.Sprintf("%d%s", t.number, t.modality) }
