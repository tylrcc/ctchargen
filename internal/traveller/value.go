package traveller

import (
	"fmt"
	"strconv"
)

// Credits is money, in whole credits. It is its own type so that it cannot
// be confused with any other integer in the procedure — a term ordinal, a
// die roll, a characteristic.
type Credits int64

func (c Credits) String() string { return fmt.Sprintf("CR %d", int64(c)) }

// MonthsPerYear is the number of months in a year, which is the only reason
// Age carries months at all.
const MonthsPerYear = 12

// Age is a character's age: whole years, plus the months that only a medical
// crisis recovery can add.
//
// All characters begin at 18 (p. 4), and a term of service adds four years
// (p. 5) — so years alone would do, but for pp. 7-8: when a characteristic
// is reduced to zero and the character survives the crisis, "the character
// ages (one die equals the number of months in added age) immediately". The
// sentence begins on p. 7 and ends on p. 8. Months carry into years.
type Age struct {
	years  int
	months int
}

// NewAge returns an age of whole years and no months. Generation starts at
// NewAge(18) (p. 4).
func NewAge(years int) Age { return Age{years: years} }

// PlusYears returns the age advanced by n whole years.
func (a Age) PlusYears(n int) Age { return Age{years: a.years + n, months: a.months} }

// PlusMonths returns the age advanced by n months, carrying into years.
//
// Only a medical crisis recovery adds months, and only ever a die roll of
// them, so n is positive in every use the procedure has. It divides toward
// negative infinity anyway: Go's % keeps the sign of its left operand, so
// the obvious arithmetic would leave a negative month count behind and quietly
// break the one invariant this type has — that Months is 0 through 11.
func (a Age) PlusMonths(n int) Age {
	total := a.months + n

	years, months := a.years+total/MonthsPerYear, total%MonthsPerYear
	if months < 0 {
		years, months = years-1, months+MonthsPerYear
	}

	return Age{years: years, months: months}
}

// Years is the whole years of the age.
func (a Age) Years() int { return a.years }

// Months is the part-year months, always 0 through 11.
func (a Age) Months() int { return a.months }

func (a Age) String() string {
	if a.months == 0 {
		return strconv.Itoa(a.years)
	}

	return fmt.Sprintf("%d years %d months", a.years, a.months)
}

// PassageClass is one of the three forms a passage is available in
// (pp. 21-22): "Passages are available in three forms, one of which the
// specific ticket will reflect: High, Middle, or Low."
type PassageClass int

// The three (pp. 21-22).
const (
	HighPassage PassageClass = iota
	MiddlePassage
	LowPassage
)

// PassageClasses is the three in the order pp. 21-22 define them.
var PassageClasses = [...]PassageClass{HighPassage, MiddlePassage, LowPassage}

func (p PassageClass) String() string {
	switch p {
	case HighPassage:
		return "High Passage"
	case MiddlePassage:
		return "Middle Passage"
	case LowPassage:
		return "Low Passage"
	}

	return fmt.Sprintf("PassageClass(%d)", int(p))
}

// ShipKind is one of the two starships mustering out can deliver (p. 22):
// "Two types of starships are available as potential mustering out benefits:
// Free Traders and Scouts."
type ShipKind int

// The two (pp. 22-23; Book 2 pp. 18-19 for their values).
const (
	ScoutShip ShipKind = iota
	FreeTrader
)

// ShipKinds is both ships.
var ShipKinds = [...]ShipKind{ScoutShip, FreeTrader}

func (s ShipKind) String() string {
	switch s {
	case ScoutShip:
		return "Scout ship, Type S"
	case FreeTrader:
		return "Free Trader, Type A"
	}

	return fmt.Sprintf("ShipKind(%d)", int(s))
}

// Title is a rank of nobility, which Book 3 p. 22 says accrues to a Social
// Standing of 11 or greater: "The nobility table indicates the actual
// designations or titles accruing to specific social standing values."
//
// The book prints each as a pair, and the tool keeps the pair: p. 8 records
// that "Nowhere in these rules is a specific requirement established that
// any character ... be of a specific gender or race", so the record does not
// choose between them.
type Title int

// The five, from Social Standing 11 through 15 (Book 3 p. 22).
const (
	Knight Title = iota
	Baron
	Marquis
	Count
	Duke
)

// Titles is the five, from Social Standing 11 through 15.
var Titles = [...]Title{Knight, Baron, Marquis, Count, Duke}

func (t Title) String() string {
	switch t {
	case Knight:
		return "knight/dame"
	case Baron:
		return "baron/baroness"
	case Marquis:
		return "marquis/marchioness"
	case Count:
		return "count/countess"
	case Duke:
		return "duke/duchess"
	}

	return fmt.Sprintf("Title(%d)", int(t))
}
