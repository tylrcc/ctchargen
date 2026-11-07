package traveller

import (
	"errors"
	"fmt"
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
// Implements: Die roll conventions: `N+`, `N−`, exact [1:2–3] in
// Characteristics and the record.
type Target struct {
	number   int
	modality Modality
}

// NewTarget builds a target from a number and a modality.
func NewTarget(number int, modality Modality) Target {
	return Target{number: number, modality: modality}
}

// minusSigns are the two characters a minus may arrive as: the ASCII hyphen
// a data file is typed with, and the U+2212 the page is set in.
const minusSigns = "-−"

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

	modality := Exactly

	if last := runes[len(runes)-1]; last == '+' || strings.ContainsRune(minusSigns, last) {
		if len(runes) == 1 {
			return Target{}, fmt.Errorf("%w %q: a sign with no number", errTarget, s)
		}

		modality = AtLeast
		if last != '+' {
			modality = AtMost
		}

		runes = runes[:len(runes)-1]
	}

	// P. 3 puts the sign of a throw after its number and the sign of a
	// modifier before it: "throws are always followed by a sign unless the
	// number must be thrown exactly, and DMs are always preceded by a
	// sign." A leading sign here means a DM was passed off as a target.
	digits := string(runes)
	if digits == "" || digits[0] == '+' || digits[0] == '-' || strings.HasPrefix(digits, "−") {
		return Target{}, fmt.Errorf("%w %q: a leading sign marks a modifier, not a throw", errTarget, s)
	}

	number, err := strconv.Atoi(digits)
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
