package traveller

import (
	"fmt"
	"strings"
)

// Characteristic is one of the six abilities a character is rolled with, in
// the order Book 1 p. 4 rolls them — which is also the order the UPP prints
// them (p. 8) and the order the Aging Table lists them (p. 9).
type Characteristic int

// The six, in rolled order (p. 4).
const (
	Strength Characteristic = iota
	Dexterity
	Endurance
	Intelligence
	Education
	SocialStanding
)

// Characteristics is the six in rolled order, for iteration.
var Characteristics = [...]Characteristic{
	Strength, Dexterity, Endurance, Intelligence, Education, SocialStanding,
}

func (c Characteristic) String() string {
	switch c {
	case Strength:
		return "Strength"
	case Dexterity:
		return "Dexterity"
	case Endurance:
		return "Endurance"
	case Intelligence:
		return "Intelligence"
	case Education:
		return "Education"
	case SocialStanding:
		return "Social Standing"
	}

	return fmt.Sprintf("Characteristic(%d)", int(c))
}

// Bounds on a characteristic value, both from Book 1 p. 4: "Characteristics
// (for player-characters) may never exceed 15, and do not go below 1 except
// for calamitous injury or aging."
//
// The quotation is the page's; the constants below are what it fixes.
const (
	// maxCharacteristic is the ceiling, and it is unconditional.
	maxCharacteristic = 15

	// MinCharacteristic is the floor for every ordinary alteration,
	// including the procedure's one negative table result, Other's -1
	// Social (p. 11).
	MinCharacteristic = 1

	// minUnderAging is how far aging alone may carry a characteristic.
	// The page opens the floor for aging without saying how far; E010
	// reads it as 0, which is the only value below 1 the rules give any
	// meaning to — a medical crisis, resolved at pp. 7-8.
	minUnderAging = 0
)

// Profile is a character's six characteristics, indexed by Characteristic.
//
// Indexing by the type is what removes the lookup that can miss: there is no
// unknown characteristic to return an error about, because there is no
// characteristic outside the six.
type Profile [len(Characteristics)]int

// Alter applies an ordinary characteristic alteration — a skills table
// result, a rank grant, a mustering out row (pp. 9, 11, 23) — and returns
// the new profile. It caps at 15 and floors at 1 (p. 4).
//
// Aging does not go through here. See AgeReduce.
//
// Implements: Never above 15 [1:4].
// Implements: Floor of 1 for every ordinary alteration [1:4 (E010)].
func (p Profile) Alter(c Characteristic, delta int) Profile {
	p[c] = clamp(p[c]+delta, MinCharacteristic, maxCharacteristic)

	return p
}

// AgeReduce applies an aging reduction from the Aging Table (p. 9), where
// delta is the positive size of the reduction, and returns the new profile.
//
// This is the one path that floors at 0 rather than 1 (E010). A profile that
// clamped at 1 unconditionally would make the medical crisis of pp. 7-8
// unreachable, and with it every path that puts months on a character's age.
//
// Implements: Floor of 0 for an aging reduction [1:4, 7–9 (E010)].
func (p Profile) AgeReduce(c Characteristic, delta int) Profile {
	p[c] = clamp(p[c]-delta, minUnderAging, maxCharacteristic)

	return p
}

// UPP renders the Universal Personality Profile: the six characteristics as
// hexadecimal digits in rolled order, 0-9 then A-F for 10-15 (p. 8).
//
// Implements: UPP as hexadecimal, in rolled order [1:8].
func (p Profile) UPP() string {
	var b strings.Builder

	for _, c := range Characteristics {
		fmt.Fprintf(&b, "%X", p[c])
	}

	return b.String()
}

func clamp(v, low, high int) int {
	return max(low, min(v, high))
}
