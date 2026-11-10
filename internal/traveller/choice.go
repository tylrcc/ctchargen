package traveller

import "fmt"

// Intent is what a character means to do at the end of a term, once the
// reenlistment throw has been made and was neither a 12 nor a failure
// (pp. 6-7, 21).
//
// Discharge and Retire never both appear: p. 21 makes leaving at the end of
// the fifth term or later retirement by definition, and p. 7 caps voluntary
// service at seven terms.
type Intent int

// The three (pp. 6-7, 21).
const (
	Continue Intent = iota
	Discharge
	Retire
)

// Intents is the three, for iteration. The order is the declaration's, and
// deliberately not a preference: the auto policy gives each strategy its own
// ranking of the three, so no single order here could be the policy's. A
// preference is a rule, and rules are not what these types carry.
var Intents = [...]Intent{Continue, Discharge, Retire}

func (i Intent) String() string {
	switch i {
	case Continue:
		return "continue"
	case Discharge:
		return "discharge"
	case Retire:
		return "retire"
	}

	return fmt.Sprintf("Intent(%d)", int(i))
}

// MusterTable is one of the two Mustering Out Tables (p. 9): "Table 1
// provides travel, education and material benefits; while Table 2 provides
// cash severance pay."
type MusterTable int

// The two (p. 9).
const (
	TableOne MusterTable = iota
	TableTwo
)

// MusterTables is both tables, in the order p. 9 prints them.
var MusterTables = [...]MusterTable{TableOne, TableTwo}

func (t MusterTable) String() string {
	switch t {
	case TableOne:
		return "Table 1, Material Benefits"
	case TableTwo:
		return "Table 2, Cash Allowances"
	}

	return fmt.Sprintf("MusterTable(%d)", int(t))
}

// EnlistmentOffer is one service a character may attempt to enlist in,
// carried to the Decider with everything the choice turns on: the service,
// the enlistment throw it prints, and the cumulative modifier this character
// earns against it (pp. 5, 10).
//
// The DM travels with the offer so that choosing a service stays a pure
// function of what the chooser is handed. A policy that had to reach into
// the character to rank services would be reaching past the question it was
// asked. The throw is an Odds, the shape every question that leads to a
// throw shows it in.
type EnlistmentOffer struct {
	Odds

	Service ServiceName
}

// ChoicePoint names one point at which the procedure asks. There is exactly
// one for each method of the Decider interface, and the value's String is
// that method's name.
//
// The v1 contract (docs/PRD.md as of ec7289f, in the history) called this "a
// rendering of that interface, never a second list kept parallel to it" — and
// an enum is, unavoidably, a second list. What
// makes it honest is a check in chargen's tests, which holds these constants
// and the interface's methods to each other by reflection, so the list cannot
// drift.
type ChoicePoint int

// One per Decider method.
const (
	ChoiceService ChoicePoint = iota
	ChoiceSubmitToDraft
	ChoiceAttemptCommission
	ChoiceAttemptPromotion
	ChoiceReenlistIntent
	ChoiceAssumeTitle
	ChoiceSkillTable
	ChoiceWeapon
	ChoiceMusterTable
	ChoiceMusterTable1DM
	ChoiceMusterTable2DM
	ChoiceMusterWeapon
)

// ChoicePoints is every choice point, for iteration and for the gate.
var ChoicePoints = [...]ChoicePoint{
	ChoiceService, ChoiceSubmitToDraft, ChoiceAttemptCommission, ChoiceAttemptPromotion,
	ChoiceReenlistIntent, ChoiceAssumeTitle, ChoiceSkillTable, ChoiceWeapon,
	ChoiceMusterTable, ChoiceMusterTable1DM, ChoiceMusterTable2DM, ChoiceMusterWeapon,
}

// String is the name of the Decider method this choice point stands for.
func (c ChoicePoint) String() string {
	switch c {
	case ChoiceService:
		return "Service"
	case ChoiceSubmitToDraft:
		return "SubmitToDraft"
	case ChoiceAttemptCommission:
		return "AttemptCommission"
	case ChoiceAttemptPromotion:
		return "AttemptPromotion"
	case ChoiceReenlistIntent:
		return "ReenlistIntent"
	case ChoiceAssumeTitle:
		return "AssumeTitle"
	case ChoiceSkillTable:
		return "SkillTable"
	case ChoiceWeapon:
		return "Weapon"
	case ChoiceMusterTable:
		return "MusterTable"
	case ChoiceMusterTable1DM:
		return "MusterTable1DM"
	case ChoiceMusterTable2DM:
		return "MusterTable2DM"
	case ChoiceMusterWeapon:
		return "MusterWeapon"
	}

	return fmt.Sprintf("ChoicePoint(%d)", int(c))
}

// Odds is a throw a choice leads to, as the chooser is shown it before
// answering: the target the service prints, and the cumulative modifier the
// character earns against it as things stand (pp. 5-6, 10). Each enlistment
// offer carries one, and so do the commission, the promotion and the next
// term's survival.
type Odds struct {
	Target Target
	DM     int
}

// Earned is how many skills rolls a term has given, by what gave them: the
// term itself, a commission and a promotion (p. 6, the Basic Skill
// Eligibility box). A term's rolls are spent before it ends, so this is one
// term's and never a running total.
type Earned struct {
	Term, Commission, Promotion int
}

// Rolls is how many skills rolls the term has given in all.
func (e Earned) Rolls() int { return e.Term + e.Commission + e.Promotion }

// SkillRoll is one skills roll as the chooser is asked it: which of the
// term's rolls it is, counting from 1, and what gave the term its rolls.
type SkillRoll struct {
	N  int
	Of Earned
}

// SkillTableOffer is one skills table a character may designate, carried to
// the Decider with what the choice turns on: the table, how many results the
// character has already had off it, and its six results by die face from
// the character's service column (p. 11).
//
// Taken is engine-computed and never recorded, so a decider can weigh where
// the character is under-trained without reaching into the character, and
// rewording nothing can change one. Results is what a person choosing needs
// to see.
type SkillTableOffer struct {
	Table   SkillTable
	Taken   int
	Results []TableResult
}
