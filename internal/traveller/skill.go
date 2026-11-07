package traveller

import "fmt"

// SkillTable is one of the four Acquired Skills tables (Book 1 p. 11), in
// the order the page prints them.
type SkillTable int

// The four (p. 11). The fourth is "allowed only for characters with
// education 8+", a gate the engine tests when the table is designated.
const (
	PersonalDevelopment SkillTable = iota
	ServiceSkills
	AdvancedEducation
	AdvancedEducationEight
)

// SkillTables is the four in book order, for iteration and tie-breaking.
var SkillTables = [...]SkillTable{
	PersonalDevelopment, ServiceSkills, AdvancedEducation, AdvancedEducationEight,
}

func (t SkillTable) String() string {
	switch t {
	case PersonalDevelopment:
		return "Personal Development Table"
	case ServiceSkills:
		return "Service Skills Table"
	case AdvancedEducation:
		return "Advanced Education Table"
	case AdvancedEducationEight:
		return "Advanced Education Table (education 8+)"
	}

	return fmt.Sprintf("SkillTable(%d)", int(t))
}

// WeaponCategory is one of the two categories in which the rules demand a
// specific weapon be named at once (pp. 11-13): "When blade or gun combat is
// acquired, the specific weapon in which expertise is achieved must be
// specified immediately."
//
// Brawling and Gunnery are not categories. P. 12 is explicit: the immediate
// choice applies to Blade Combat and Gun Combat, "not Brawling or Gunnery".
type WeaponCategory int

// The two (pp. 12-13).
const (
	Blade WeaponCategory = iota
	Gun
)

// WeaponCategories is the two in book order.
var WeaponCategories = [...]WeaponCategory{Blade, Gun}

func (c WeaponCategory) String() string {
	switch c {
	case Blade:
		return "Blade Combat"
	case Gun:
		return "Gun Combat"
	}

	return fmt.Sprintf("WeaponCategory(%d)", int(c))
}

// WeaponName is the one name in the domain that does not close at compile
// time. The blades and polearms list and the guns list are printed data
// (pp. 12-13), so names lift from those lists in the rules package and are
// checked against the list for their category. The category is the closed
// type; the name is data.
type WeaponName string

// SkillName is a skill's name, normalized to the heading its description
// carries on pp. 12-20 (E012). The p. 11 tables abbreviate and do not spell
// every name the same way twice, and the p. 23 box carries a reprint typo;
// normalizing is what keeps one skill from accumulating under two names.
type SkillName string

// Skill is a name and the level at which it is held (p. 12): "Upon the first
// acquisition of a skill, the player writes the skill name, followed by a
// dash and the number 1 ... Additional acquisitions of the same skill
// increase this skill level to 3, 4 or higher." There is no cap.
//
// Where the rules demand a specific weapon, the name is that weapon's:
// the worked example's own skill list records Dagger-1 and Cutlass-1, not
// Blade Combat (p. 25).
//
// The category a weapon belongs to is deliberately not stored here. It is
// printed data on pp. 12-13, held in the rules package, and a copy kept
// beside it could disagree with the list it came from.
type Skill struct {
	Name  SkillName
	Level int
}

func (s Skill) String() string { return fmt.Sprintf("%s-%d", s.Name, s.Level) }
