package chargen

import (
	"fmt"
	"slices"
	"strings"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// Policy is the auto run's decider (new --auto, and every batch): it answers
// every question the procedure asks,
// under three named strategies, each over one group of choice points. Career
// (--career) answers Service, SubmitToDraft, AttemptCommission,
// AttemptPromotion, ReenlistIntent and AssumeTitle; Skills (--skills) answers
// SkillTable and Weapon; Muster (--muster) answers MusterTable,
// MusterTable1DM, MusterTable2DM and MusterWeapon. Each method's doc says when
// the engine asks it and what each strategy answers, and those are the whole
// of what --auto decides.
//
// It is total - it answers every method of Decider - and deterministic. Every
// strategy is a pure function of what it is handed. Where a strategy needs to
// know that an option is legal, the engine supplies that by what it puts in
// the offered set, never by the strategy reaching into the character.
//
// Ranked preferences are written best-first. A strategy takes the first of its
// ranking that is on offer; where its ranking is silent, the tie-break is
// first-listed order in Book 1 - the Prior Service Table's column order for
// services (p. 10), the Acquired Skills Table's block order for skill tables
// (p. 11) - which is the order the engine builds an offered set in.
//
// It never returns an error. An offered set it cannot answer from would mean
// the engine offered something the rules do not contain, which is an engine
// defect and fails there rather than being absorbed here. Every offered set
// must be non-empty: the engine never offers an empty one, so nothing here
// checks, and a caller handing a Policy one directly gets the panic of
// indexing it (#131).
//
// It decides nothing a flag already decides - --service forces the enlistment
// attempt and suppresses Service, and --name is not a choice point - and
// nothing after generation: selling a passage back (p. 22), buying
// Travellers' Aid membership for CR 1,000,000 (p. 22) and flying the ship are
// play, and the record carries the assets and stops.
// Interactive mode implements the same Decider and asks the player instead.
type Policy struct {
	Career Career
	Skills Skills
	Muster Muster
}

// PolicyVersion is the version of what --auto answers.
//
// Every auto run's record carries it, beside the ruleset that
// governed the rules and the build that wrote the file. Those two move for
// reasons that have nothing to do with what --auto answers; this moves only
// when an answer does.
//
// It is what a referee reading a record consults to tell which answers made
// it. Without it he can see only what was answered, and `build` cannot tell
// him which policy did - it changes when a comment changes.
//
// Changing what any Policy method answers is a bump, because it changes every
// character --auto makes. A new choice point is not: no record was answered
// differently before it, because the question was never asked. Version 2
// draws the weapon rather than taking the first name; version 3 designates
// the skills table he has trained on least (#34).
const PolicyVersion = 3

// Career is the --auto career strategy: continue, retire, or take one term.
//
// It is a type and not a string because it names a closed set: a value
// outside the three is a parse error at the boundary rather than something a
// runtime check has to notice, and a runtime check for an impossible value is
// a sign the type is wrong (#41).
type Career int

// Skills is the --auto skills strategy: which table to take a skill from.
// A closed set, for the reason Career is.
type Skills int

// Muster is the --auto mustering out strategy: which table, and what to do
// with a repeated weapon. A closed set, for the reason Career is.
type Muster int

// The career strategies, in their listed order.
const (
	CareerServe Career = iota
	CareerRetire
	CareerOneTerm
)

// The skills strategies, in their listed order.
const (
	SkillsAdvanced Skills = iota
	SkillsService
	SkillsPersonal
)

// The mustering out strategies, in their listed order.
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

// DefaultPolicy is what an auto run applies when it names no strategy. The
// command's flag defaults read it rather than naming strategies of their own
// (#367). A
// record names its strategies either way, so no record is silent about the
// policy that made it.
func DefaultPolicy() Policy {
	return Policy{Career: CareerServe, Skills: SkillsAdvanced, Muster: MusterCash}
}

// parsed finds the one strategy of an alphabet that a word spells.
//
// The rejection names the values the flag takes. The reader most likely to be
// told his strategy is wrong is the one who typed `go install` and has never
// seen this tree.
//
// Implements: A word that names no strategy never becomes one [Choice points].
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
	return strings.Join(names(all, T.String), ", ")
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

// Service is asked when no --service flag was given. With --service the
// attempt is forced to the named service and this is not called; the throw is
// still made, and a failed throw still goes to the draft. It is offered one
// EnlistmentOffer per service of the Prior Service Table (p. 10), in book
// order - Navy, Marines, Army, Scouts, Merchants, Other - each carrying the
// printed enlistment target and the cumulative DM this character earns
// against it.
//
// serve, retire and oneterm all take the offer whose throw is likeliest to
// succeed: the 2D probability of meeting its target after its DM. Ties go to
// the first listed, which is the Prior Service Table's column order and
// nothing else.
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

// SubmitToDraft is asked when the enlistment throw failed. The question
// exists at all because the draft is declinable (E001).
//
// serve and retire answer yes. oneterm answers no, and generation ends with
// an 18-year-old civilian; it is the strategy that reaches the civilian record
// at all, and the only one whose golden can.
func (p Policy) SubmitToDraft() (bool, error) { return p.Career != CareerOneTerm, nil }

// AttemptCommission is asked when the service has commissions (not Scouts,
// not Other; pp. 6, 10), the character is not yet commissioned, and it is not
// a draftee's first term (p. 5).
//
// All three strategies answer yes. Nothing in the rules costs a character a
// failed commission throw, and a commission carries a skill eligibility
// (p. 6); a strategy that declined would be modelling a preference the book
// does not price.
func (p Policy) AttemptCommission() (bool, error) { return true, nil }

// AttemptPromotion is asked when the character is commissioned and holds a
// rank below the highest his service prints (E013), once per term (p. 6).
//
// All three strategies answer yes, for the reason AttemptCommission does.
func (p Policy) AttemptPromotion() (bool, error) { return true, nil }

// AssumeTitle is asked when the character survived generation and his final
// Social Standing is 11 or greater - once, at the end of generation (E011).
// The dead are assessed but not asked: assuming a title is an act a dead
// character does not perform. The title is the one Book 3 p. 22 confers at
// that value.
//
// All three strategies answer yes.
func (p Policy) AssumeTitle(traveller.Title) (bool, error) { return true, nil }

// ReenlistIntent is asked when the reenlistment throw was made, was not a 12,
// and succeeded, and more than one intent is legal. A 12 forces another term
// "regardless of his desires" (p. 6), a failed throw ends the service, and at
// the end of term 7 or later the only remaining intent is to leave (E003) and
// the departure is Retire, so none of those asks. After terms 1-4 the engine
// offers Continue and Discharge; after terms 5-6, Continue and Retire.
// Discharge and Retire never coexist: p. 21 makes leaving at the end of the
// 5th term or later retirement by definition ("is considered to have
// retired"), and p. 7 caps voluntary service at seven terms. A Retire from
// the Scouts or Other is a retirement that pays nothing (p. 21).
//
// Ranked best-first:
//
//   - serve: Continue, Retire, Discharge. It takes the full seven terms
//     whenever the throws allow.
//   - retire: Retire, Continue, Discharge. It leaves the moment leaving counts
//     as retirement, the end of term 5 (p. 21), whether or not the service
//     pays a pension.
//   - oneterm: Discharge, Retire, Continue. It leaves at the first
//     opportunity.
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

// SkillTable is asked once per skill eligibility, before the die is rolled
// (p. 11: "must specify the table being consulted prior to the die throw").
// It is offered Personal Development, Service Skills, Advanced Education, and
// Advanced Education (Education 8+) - the fourth only while the character's
// Education stands at 8 or greater at the moment of the designation.
//
// It designates the table the character has trained on least, and the skills
// strategy decides among those, ranked best-first:
//
//   - advanced: Advanced Education (8+), Advanced Education, Service Skills,
//     Personal Development.
//   - service: Service Skills, Advanced Education (8+), Advanced Education,
//     Personal Development.
//   - personal: Personal Development, Service Skills, Advanced Education (8+),
//     Advanced Education.
//
// taken is parallel to from: how many results he has already had off each
// offered table. It is engine-computed and never recorded, so this weighs
// where he is thin without reaching into him (#34).
//
// Version 2 followed the ranking alone, so advanced designated Advanced
// Education every time it was offered and Personal Development never - ninety
// times to none over thirty characters - and a character generated under the
// default never raised a characteristic, though p. 11's first table is how
// that is done. A first designation, when every count is zero, is still
// exactly what the ranking says.
//
// Where Weapon admits that the page gives no basis, this one does not need
// to: the page gives a basis, in that a character trains where he is thin,
// and the strategy still says which way he leans.
//
// advanced is the default because it is the one ranking that makes the
// Education 8+ gate visible in a default run: it takes the fourth table the
// instant it opens. The gate never closes again once open - p. 9's Aging
// Table prints "unaffected by aging" across Education, and the procedure's one
// negative table result is Other's -1 Social (p. 11) - so no strategy needs an
// answer for an Education that falls back below 8. personal is the ranking
// that reaches that one negative result.
//
// Implements: The skills table designated is the one he has trained on least
// [1:11].
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
// taken is parallel to from, one count per offered table, as the engine
// passes it (#370).
func thinnest(from []traveller.SkillTable, taken []int) []traveller.SkillTable {
	fewest := slices.Min(taken)

	thin := make([]traveller.SkillTable, 0, len(from))

	for i, table := range from {
		if taken[i] == fewest {
			thin = append(thin, table)
		}
	}

	return thin
}

// Weapon is asked when a skills-table result is Blade Cbt or Gun Cbt,
// immediately (pp. 11-13). It is offered the blades and polearms list (p. 12)
// or the guns list (p. 13), whichever the category names, in printed order.
//
// All three strategies draw a name from the list, uniformly. The draw comes
// from the run's own dice, so a fixture that scripts its draws still walks the
// same path; it is the only variation any strategy takes.
//
// It draws rather than ranks because the book names no basis. Where p. 11
// makes the player designate a table, or p. 21 decides a departure by the
// term count, the page supplies something to decide on and the policy decides
// on it. Here the page supplies a list in printed order, and printed order is
// not a preference: taking the first is as invented as drawing, and drawing
// admits it. Version 1 took the first every time, and thirty auto-generated
// characters carried twenty-two Body Pistols and seventeen Daggers and nothing
// else, though every name on both lists was offered (#34). No strategy reached
// past it, so no flag could change it either.
//
// A later combat result draws again, and a draw that names a weapon he
// already has raises its level (p. 12: "Additional acquisitions of expertise
// in the same weapon increase the present level by one"); that is the branch
// that produces a level above 1 from the skills tables. Diversifying on
// purpose is spartan's MusterWeapon.
//
// Implements: The weapon for a Blade or Gun result is drawn, not always the
// first printed [1:11–13].
func (p Policy) Weapon(
	_ traveller.WeaponCategory, from []traveller.WeaponName, vary Vary,
) (traveller.WeaponName, error) {
	return from[vary.Among(len(from))], nil
}

// MusterTable is asked once per benefit roll, before the die (p. 9: "The
// player must designate the table before the die is rolled"). Table 2 is
// offered only while fewer than three rolls have been taken on it (p. 9: "A
// maximum of three rolls on table 2 are allowed per character"); once the cap
// is reached Table 1 is the only option and this is not called.
//
// cash ranks Table 2 over Table 1; goods and spartan rank Table 1 over
// Table 2. cash reaches the three-roll cap in any run with three or more
// rolls, which is the only way the cap is exercised; goods is what reaches the
// ship benefits, Travellers' Aid, and Table 1's dash rows.
func (p Policy) MusterTable(from []traveller.MusterTable) (traveller.MusterTable, error) {
	if p.Muster == MusterCash {
		return prefer(from, []traveller.MusterTable{traveller.TableTwo, traveller.TableOne}), nil
	}

	return prefer(from, []traveller.MusterTable{traveller.TableOne, traveller.TableTwo}), nil
}

// MusterTable1DM is asked when the character's final rank is 5 or 6 (p. 9:
// "Characters with rank 5 or 6 may add +1 to their rolls on this table") and
// the roll is on Table 1.
//
// cash and goods answer yes; spartan answers no. The declined branch is
// reached by the golden navy-spartan-declines: spartan prefers Table 1, so a
// spartan character who reaches rank 5 is asked and refuses.
func (p Policy) MusterTable1DM() (bool, error) { return p.Muster != MusterSpartan, nil }

// MusterTable2DM is asked when the character has Gambling skill (p. 9:
// "Individuals with gambling expertise are allowed a DM of +1 on table 2") and
// the roll is on Table 2.
//
// cash and goods answer yes; spartan answers no. On an auto run only cash is
// ever asked: it is the only strategy that rolls on Table 2, because goods
// and spartan both prefer Table 1 and Table 1 is always on offer. So --auto
// reaches only cash's yes, and goods' yes and spartan's no are answers no
// generated character is given, held by TestMusterStrategies alone. The
// declined branch is reachable only from outside the policy, where a person
// is asked and answers no
// (cmd/ctchargen.TestInteractiveOffersTheGamblingModifier).
func (p Policy) MusterTable2DM() (bool, error) { return p.Muster != MusterSpartan, nil }

// MusterWeapon is asked when a Table 1 roll is a weapon benefit (p. 9's Blade
// and Gun rows). from is the category's printed list, read column-major (settled by the
// page; see rules.Rules.Weapons). received is the weapons
// of this category the character has already taken as benefits - narrower
// twice over than the weapons he holds: p. 22 says "Expertise may only be
// taken in a weapon received as a benefit", which excludes weapons from
// skills-table results, and offers the expertise "in lieu of receiving a
// second or subsequent weapon of exactly the same type", which keeps it inside
// the benefit's own category. It is offered TakeWeapon on any name in the
// list, and TakeExpertise on a name in received once received is non-empty.
//
//   - cash and goods: TakeExpertise on the first name in received when it is
//     offered; otherwise TakeWeapon on the first name in the printed list. They
//     convert repeats into expertise, the branch p. 22's worked sentence
//     describes.
//   - spartan: TakeWeapon on the first name in the printed list not already in
//     received, or the first name in the list if every one is. It diversifies,
//     the branch that fills a benefit list with distinct weapons.
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
