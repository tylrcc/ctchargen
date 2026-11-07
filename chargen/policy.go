package chargen

import (
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/ctchargen/traveller"
)

// Policy is the auto decider of docs/POLICY.md. It is total - it answers
// every method of Decider - and deterministic, tie-breaking by first-listed
// order in Book 1.
//
// Every strategy is a pure function of what it is handed. Where a strategy
// needs to know that an option is legal, the engine supplies that by what it
// puts in the offered set, never by the strategy reaching into the
// character.
//
// Every offered set must be non-empty. The engine never offers an empty
// one, so nothing here checks; a caller handing a Policy one directly gets
// the panic of indexing it (#131).
type Policy struct {
	Career Career
	Skills Skills
	Muster Muster
}

// PolicyVersion is the version of docs/POLICY.md that this build applies.
//
// Every record carries it, beside the ruleset that governed the rules and the
// build that wrote the file. Those two move for reasons that have nothing to
// do with what --auto answers; this moves only when an answer does.
//
// It is what a referee reading a record consults to tell which decision
// table answered it. Without it he can see only what was answered, and
// `build` cannot tell him which table did - it changes when a comment
// changes.
//
// Bumping it is a decision recorded in POLICY.md, and docsgate holds the two
// to each other. Adding a row for a question that was never asked is not a
// bump: no record was answered differently under the old table.
// Version 2 draws the weapon rather than taking the first name; version 3
// designates the skills table he has trained on least (#34).
const PolicyVersion = 3

// Career is the --auto career strategy: continue, retire, or take one term.
//
// It is a type and not a string because it names a closed set: a value
// outside the three is a parse error at the boundary rather than something a
// runtime check has to notice, and a runtime check for an impossible value is
// a sign the type is wrong (#41). The zero value is the default, so
// Policy{} is DefaultPolicy().
type Career int

// Skills is the --auto skills strategy: which table to take a skill from.
// A closed set, for the reason Career is.
type Skills int

// Muster is the --auto mustering out strategy: which table, and what to do
// with a repeated weapon. A closed set, for the reason Career is.
type Muster int

// The career strategies of docs/POLICY.md, in the order that document lists
// them - the first is the default.
const (
	CareerServe Career = iota
	CareerRetire
	CareerOneTerm
)

// The skills strategies of docs/POLICY.md, in its order.
const (
	SkillsAdvanced Skills = iota
	SkillsService
	SkillsPersonal
)

// The mustering out strategies of docs/POLICY.md, in its order.
const (
	MusterCash Muster = iota
	MusterGoods
	MusterSpartan
)

// The three alphabets, for parsing a flag and for listing the choices. They
// are unexported: nothing outside needs to iterate a strategy set, and the
// two things that do need one - the parser and the help - are below.
//
//nolint:gochecknoglobals // three immutable tables, and Go has no const slice.
var (
	careerStrategies = []Career{CareerServe, CareerRetire, CareerOneTerm}
	skillsStrategies = []Skills{SkillsAdvanced, SkillsService, SkillsPersonal}
	musterStrategies = []Muster{MusterCash, MusterGoods, MusterSpartan}
)

func (c Career) String() string {
	switch c {
	case CareerServe:
		return "serve"
	case CareerRetire:
		return "retire"
	case CareerOneTerm:
		return "oneterm"
	}

	return fmt.Sprintf("Career(%d)", int(c))
}

func (s Skills) String() string {
	switch s {
	case SkillsAdvanced:
		return "advanced"
	case SkillsService:
		return "service"
	case SkillsPersonal:
		return "personal"
	}

	return fmt.Sprintf("Skills(%d)", int(s))
}

func (m Muster) String() string {
	switch m {
	case MusterCash:
		return "cash"
	case MusterGoods:
		return "goods"
	case MusterSpartan:
		return "spartan"
	}

	return fmt.Sprintf("Muster(%d)", int(m))
}

// ParsePolicy turns the three strategy flags' words into the policy they name.
//
// This is where a strategy stops being a string: at the boundary, once,
// rather than being carried inward and re-checked wherever it is used. The
// command line is the one thing that parses strategies, and it parses all
// three together every time (#135).
func ParsePolicy(career, skills, muster string) (Policy, error) {
	var (
		policy Policy
		err    error
	)

	policy.Career, err = parsed(career, "career", careerStrategies)
	if err != nil {
		return Policy{}, err
	}

	policy.Skills, err = parsed(skills, "skills", skillsStrategies)
	if err != nil {
		return Policy{}, err
	}

	policy.Muster, err = parsed(muster, "muster", musterStrategies)
	if err != nil {
		return Policy{}, err
	}

	return policy, nil
}

// CareerChoices is the values --career takes, as a reader is shown them.
//
// This and its two siblings are read by the help, and the rejection below
// composes its own list from the same alphabet - so the two cannot come to
// disagree about the order, the separator, or which set they read.
func CareerChoices() string { return choices(careerStrategies) }

// SkillsChoices is the values --skills takes, as a reader is shown them.
func SkillsChoices() string { return choices(skillsStrategies) }

// MusterChoices is the values --muster takes, as a reader is shown them.
func MusterChoices() string { return choices(musterStrategies) }

// DefaultPolicy is what a bare --auto run applies. A record names its
// strategies either way, so no record is silent about the policy that made
// it.
func DefaultPolicy() Policy {
	return Policy{Career: CareerServe, Skills: SkillsAdvanced, Muster: MusterCash}
}

// parsed finds the one strategy of an alphabet that a word spells.
//
// The rejection names the values and not the document. POLICY.md is a
// repository file, and the reader most likely to be told his strategy is
// wrong is the one who typed `go install` and has never seen this tree.
func parsed[T fmt.Stringer](word, flag string, all []T) (T, error) {
	for _, want := range all {
		if word == want.String() {
			return want, nil
		}
	}

	var none T

	return none, fmt.Errorf("%w: --%s %q; want %s",
		errNoSuchStrategy, flag, word, choices(all))
}

// choices joins an alphabet the way a reader is shown it.
func choices[T fmt.Stringer](all []T) string {
	names := make([]string, 0, len(all))

	for _, one := range all {
		names = append(names, one.String())
	}

	return strings.Join(names, ", ")
}

// prefer picks the first of a ranked preference that is on offer, and falls
// back to the first thing offered - which is book order, because that is the
// order the engine builds an offered set in.
//
// The switches below are exhaustive over closed types, so a ranked slice is
// always populated and the fallback means only what it says: nothing
// preferred was offered.
func prefer[T comparable](offered, ranked []T) T {
	for _, want := range ranked {
		if slices.Contains(offered, want) {
			return want
		}
	}

	return offered[0]
}

// Service takes the offer whose throw is likeliest to succeed, ties going to
// the first listed - which is the Prior Service Table's column order (p. 10).
func (p Policy) Service(from []traveller.EnlistmentOffer) (traveller.ServiceName, error) {
	best := from[0]
	bestOdds := odds(best)

	for _, offer := range from[1:] {
		if o := odds(offer); o > bestOdds {
			best, bestOdds = offer, o
		}
	}

	return best.Service, nil
}

// odds is the chance of meeting an offer's target on two dice after its
// modifier. It is computed from the printed target and the printed DM; it is
// not a table of its own and it is not a rule.
func odds(offer traveller.EnlistmentOffer) float64 {
	ways := 0

	const faces = 6

	for first := 1; first <= faces; first++ {
		for second := 1; second <= faces; second++ {
			if offer.Target.Satisfied(first + second + offer.DM) {
				ways++
			}
		}
	}

	return float64(ways) / float64(faces*faces)
}

// SubmitToDraft is answered no only by oneterm, which is what reaches the civilian
// record at all (E001).
func (p Policy) SubmitToDraft() (bool, error) { return p.Career != CareerOneTerm, nil }

// AttemptCommission is always yes: nothing in the rules costs a character a failed
// attempt, and a commission carries a skill eligibility (p. 6).
func (p Policy) AttemptCommission() (bool, error) { return true, nil }

// AttemptPromotion is always yes, for the same reason (p. 6).
func (p Policy) AttemptPromotion() (bool, error) { return true, nil }

// AssumeTitle is always yes: every strategy assumes the title.
func (p Policy) AssumeTitle(traveller.Title) (bool, error) { return true, nil }

// ReenlistIntent ranks the three by career strategy.
func (p Policy) ReenlistIntent(from []traveller.Intent) (traveller.Intent, error) {
	var ranked []traveller.Intent

	switch p.Career {
	case CareerServe:
		ranked = []traveller.Intent{traveller.Continue, traveller.Retire, traveller.Discharge}
	case CareerRetire:
		ranked = []traveller.Intent{traveller.Retire, traveller.Continue, traveller.Discharge}
	case CareerOneTerm:
		ranked = []traveller.Intent{traveller.Discharge, traveller.Retire, traveller.Continue}
	}

	return prefer(from, ranked), nil
}

// SkillTable designates the table the character has trained on least,
// breaking ties by the skills strategy.
//
// taken is parallel to from: how many results he has already had off each
// offered table. It is engine-computed and never recorded, so this weighs
// where he is thin without reaching into him (#34).
//
// Ranking alone designated Advanced Education every time it was offered and
// Personal Development never - ninety times to none over thirty characters -
// so a character generated under the default never raised a characteristic,
// though p. 11's first table is how that is done.
//
// The strategy still decides, and still decides everything on a first
// designation, when every count is zero. advanced remains the default because
// it is the one ranking that reaches the Education 8+ gate the instant it
// opens.
func (p Policy) SkillTable(
	from []traveller.SkillTable, taken []int,
) (traveller.SkillTable, error) {
	var ranked []traveller.SkillTable

	switch p.Skills {
	case SkillsAdvanced:
		ranked = []traveller.SkillTable{
			traveller.AdvancedEducationEight, traveller.AdvancedEducation,
			traveller.ServiceSkills, traveller.PersonalDevelopment,
		}
	case SkillsService:
		ranked = []traveller.SkillTable{
			traveller.ServiceSkills, traveller.AdvancedEducationEight,
			traveller.AdvancedEducation, traveller.PersonalDevelopment,
		}
	case SkillsPersonal:
		ranked = []traveller.SkillTable{
			traveller.PersonalDevelopment, traveller.ServiceSkills,
			traveller.AdvancedEducationEight, traveller.AdvancedEducation,
		}
	}

	return prefer(thinnest(from, taken), ranked), nil
}

// thinnest is the offered tables the character has had fewest results off.
//
// A caller with no counts - a test constructing a policy directly - gets the
// whole offered set back, so the ranking decides alone.
func thinnest(from []traveller.SkillTable, taken []int) []traveller.SkillTable {
	if len(taken) != len(from) {
		return from
	}

	fewest := slices.Min(taken)

	thin := make([]traveller.SkillTable, 0, len(from))

	for i, table := range from {
		if taken[i] == fewest {
			thin = append(thin, table)
		}
	}

	return thin
}

// Weapon draws among the names on the category's printed list.
//
// Taking the first every time arms every auto-generated character from the
// head of the list: thirty of them carried twenty-two Body Pistols and
// seventeen Daggers and nothing else, though every name on both lists was
// offered (#34).
//
// It draws rather than ranks because the book names no basis. Where p. 11
// makes the player designate a table, or p. 21 decides a departure by the
// term count, the page supplies something to decide on and the policy decides
// on it. Here the page supplies a list in printed order, and printed order is
// not a preference: taking the first is as invented as drawing, and drawing
// admits it.
//
// Repeats can still take the same weapon again, which raises its level
// (p. 12) and is the branch that produces a level above 1, by drawing the
// same name twice.
func (p Policy) Weapon(
	_ traveller.WeaponCategory, from []traveller.WeaponName, vary Vary,
) (traveller.WeaponName, error) {
	return from[vary.Among(len(from))], nil
}

// MusterTable is ranked by muster strategy. cash reaches the three-roll cap, which is the only way the
// cap is exercised; goods is what reaches the ships and the dash rows.
func (p Policy) MusterTable(from []traveller.MusterTable) (traveller.MusterTable, error) {
	if p.Muster == MusterCash {
		return prefer(from, []traveller.MusterTable{traveller.TableTwo, traveller.TableOne}), nil
	}

	return prefer(from, []traveller.MusterTable{traveller.TableOne, traveller.TableTwo}), nil
}

// MusterTable1DM is declined only by spartan, which declines both, so that the
// declined branch of each is reachable by a generated golden.
func (p Policy) MusterTable1DM() (bool, error) { return p.Muster != MusterSpartan, nil }

// MusterTable2DM is declined only by spartan, for the same reason (p. 9).
func (p Policy) MusterTable2DM() (bool, error) { return p.Muster != MusterSpartan, nil }

// MusterWeapon converts a repeat into expertise, except under spartan, which
// diversifies - the branch that fills a benefit list with distinct weapons.
func (p Policy) MusterWeapon(
	_ traveller.WeaponCategory, from, received []traveller.WeaponName,
) (traveller.WeaponBenefit, error) {
	if p.Muster != MusterSpartan && len(received) > 0 {
		return traveller.TakeExpertise{Weapon: received[0]}, nil
	}

	if p.Muster == MusterSpartan {
		for _, name := range from {
			if !slices.Contains(received, name) {
				return traveller.TakeWeapon{Weapon: name}, nil
			}
		}
	}

	return traveller.TakeWeapon{Weapon: from[0]}, nil
}
