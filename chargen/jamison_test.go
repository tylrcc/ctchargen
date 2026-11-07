package chargen_test

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// The worked example of Book 1 pp. 23-25, replayed.
//
// FR9 makes the instrument free: the procedure asks through a Decider and
// throws through a Roller, so a narrated pair walks the example's own
// character and the result is compared against the page. It is the one check
// on this engine written by someone other than its author.
//
// Every value below is read from pp. 23-25. The page's own arithmetic is
// inconsistent in four places, and the comment says so at each: a "DM of +7"
// its own sum corrects to +1; a survival "rolls 2 (+2=11)", whose total
// needs a 9; a promotion failure "by two points" that is one; and a "Table
// 7" among four tables. Where a stated roll and a stated total disagree the
// script follows the total, because the total is what the narrated outcome
// turns on.

// errNotNarrated is what a scripted answer returns on a path the example
// does not narrate. The t.Fatal above it stops the test first; this is what
// keeps the signature honest rather than returning a value nobody chose.
var errNotNarrated = errors.New("the example does not narrate this")

// dagger is the first blade on p. 12's list, and what Jamison chooses.
const dagger = "Dagger"

// jamisonThrows is every two-dice throw the engine makes, as totals, in the
// order it makes them. The example narrates the same throws in its own
// order; E002 puts skills before reenlistment, so the reenlistment throw
// sits after that term's skills here and before them on the page.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var jamisonThrows = []int{
	// "generate all six characteristics; the rolls, consecutively, 6, 8, 8,
	// 12, 8, 9" - UPP 688C89.
	6, 8, 8, 12, 8, 9,

	// Enlistment in the Merchants, 7+: "he rolls 5 (+2=7)".
	5,

	// Term 1. Survival "he rolls 11 (+2=13)". Commission 4+: the page prints
	// "DM of +7 allowed for intelligence", which its own sum corrects to +1 -
	// "he rolls 7 (+1=8)" - and the Prior Service Table gives Merchants +1.
	// Promotion 10+: "he rolls 10 (+1=11)". Reenlistment 4+: "he rolls 7".
	11, 7, 10, 7,

	// Term 2. Survival "he rolls 3, which is the lowest possible and still
	// survive (3+2=5)". Promotion "he rolls 12 (+1=13)". Reenlistment "6".
	3, 12, 6,

	// Term 3. Survival: the page prints "he rolls 2 (+2=11)", whose stated
	// total needs a 9. Promotion "he rolls 8 (+1=9)" against the Merchants'
	// 10+, which the page then calls failing "by two points" - it is one.
	// Reenlistment "10".
	9, 8, 10,

	// Term 4. Survival "he rolls 7 (+2=9)". Promotion "he throws 12
	// (+1=13)", making him 1st Officer. Reenlistment "10". Then the aging
	// round the page defers: strength 12, dexterity 8, endurance 9.
	7, 12, 10, 12, 8, 9,

	// Term 5. Survival "he rolls 7 (+2=9)". Promotion "he rolls 11 (+1=12)",
	// making him Captain. Reenlistment "he rolls 3", which fails. Then the
	// second aging round: strength 8, dexterity 6, endurance 11.
	7, 11, 3, 8, 6, 11,
}

// jamisonDice is every one-die roll, in order: the skills tables term by
// term, then the seven mustering out rolls.
//
//nolint:gochecknoglobals // an immutable script, and Go has no const slice.
var jamisonDice = []int{
	1, 5, 2, 5, // term 1: +1 strength, blade combat, vacc suit, electronics
	3, 4, // term 2: +1 endurance, gun combat
	5,    // term 3: electronics
	5, 4, // term 4: blade combat, gun combat
	5, 3, // term 5: pilot, electronics

	// "Jamison elects to make one roll on Table 2 [he rolls 4= CR 20,000]
	// and six rolls on Table 1 [he rolls 5 (+1=6); 6 (+1=7); 2 (+1=3);
	// 6 (+1=7); 6 (+1=7); 6 (+1=7)]".
	4,
	5, 6, 2, 6, 6, 6,
}

// narrated rolls the sequence the page narrates, and refuses to be asked for more
// than it was given: a throw the engine makes and the page does not narrate
// would otherwise be answered with a zero nobody chose.
type narrated struct {
	t      *testing.T
	throws []int
	dice   []int
}

// Among is never called: the worked example drives its own decider, which
// names each weapon the book names rather than choosing one. If this ever
// fires, the replay has stopped following p. 25 and started guessing.
func (s *narrated) Among(int) int {
	s.t.Fatalf("the worked example drew among alternatives; p. 25 names them")

	return 0
}

func (s *narrated) Die() int {
	s.t.Helper()

	if len(s.dice) == 0 {
		s.t.Fatal("the engine asked for a one-die roll the example does not narrate")
	}

	next := s.dice[0]

	s.dice = s.dice[1:]

	return next
}

func (s *narrated) TwoDice() (int, int) {
	s.t.Helper()

	if len(s.throws) == 0 {
		s.t.Fatal("the engine asked for a two-dice throw the example does not narrate")
	}

	total := s.throws[0]

	s.throws = s.throws[1:]

	first := min(6, total-1)

	return first, total - first
}

// decisions answers the choice points the example narrates. Its queues are
// what p. 24 records Jamison designating and choosing.
type decisions struct {
	t       *testing.T
	tables  []traveller.SkillTable
	weapons []traveller.WeaponName
	muster  []traveller.MusterTable
}

func (d *decisions) Service([]traveller.EnlistmentOffer) (traveller.ServiceName, error) {
	d.t.Fatal("the service was forced; the example does not choose one")

	return 0, nil
}

func (d *decisions) SubmitToDraft() (bool, error) {
	d.t.Fatal("Jamison enlisted; the example never reaches the draft")

	return false, nil
}

func (d *decisions) AttemptCommission() (bool, error) { return true, nil }
func (d *decisions) AttemptPromotion() (bool, error)  { return true, nil }

func (d *decisions) SkillTable([]traveller.SkillTable, []int) (traveller.SkillTable, error) {
	d.t.Helper()

	if len(d.tables) == 0 {
		d.t.Fatal("the engine designated more skills tables than the example does")
	}

	next := d.tables[0]

	d.tables = d.tables[1:]

	return next, nil
}

func (d *decisions) Weapon(
	traveller.WeaponCategory, []traveller.WeaponName, chargen.Vary,
) (traveller.WeaponName, error) {
	d.t.Helper()

	if len(d.weapons) == 0 {
		d.t.Fatal("the engine asked for more weapons than the example names")
	}

	next := d.weapons[0]

	d.weapons = d.weapons[1:]

	return next, nil
}

func (d *decisions) ReenlistIntent([]traveller.Intent) (traveller.Intent, error) {
	return traveller.Continue, nil
}

func (d *decisions) MusterTable([]traveller.MusterTable) (traveller.MusterTable, error) {
	d.t.Helper()

	if len(d.muster) == 0 {
		d.t.Fatal("the engine took more benefit rolls than the example does")
	}

	next := d.muster[0]

	d.muster = d.muster[1:]

	return next, nil
}

// "Characters with rank 5 or 6 may add +1 to their rolls on this table"
// (p. 9), and the example adds it to all six.
func (d *decisions) MusterTable1DM() (bool, error) { return true, nil }

func (d *decisions) MusterTable2DM() (bool, error) {
	d.t.Fatal("Jamison has no gambling expertise, so p. 9 offers him no modifier on Table 2")

	return false, nil
}

func (d *decisions) MusterWeapon(
	traveller.WeaponCategory, []traveller.WeaponName, []traveller.WeaponName,
) (traveller.WeaponBenefit, error) {
	d.t.Fatal("no weapon row comes up; the six rolls reach rows 6, 7, 3, 7, 7 and 7")

	return nil, errNotNarrated
}

func (d *decisions) AssumeTitle(traveller.Title) (bool, error) {
	d.t.Fatal("Jamison's social standing is 9, which confers no title (Book 3 p. 22)")

	return false, nil
}

func replayJamison(t *testing.T) *chargen.Character {
	t.Helper()

	inputs := chargen.Inputs{
		Name: "Jamison", Service: traveller.Merchants, Forced: true,
		Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash,
	}

	rolls := &narrated{t: t, throws: slices.Clone(jamisonThrows), dice: slices.Clone(jamisonDice)}

	chosen := &decisions{
		t: t,
		// P. 24 calls the four Acquired Skills tables 1 through 4, and in
		// term 4 mislabels the first as "Table 7". The results it states -
		// +1 strength, blade combat, +1 endurance - are the Personal
		// Development column either way.
		tables: []traveller.SkillTable{
			traveller.PersonalDevelopment, traveller.PersonalDevelopment,
			traveller.ServiceSkills, traveller.ServiceSkills,
			traveller.PersonalDevelopment, traveller.ServiceSkills,
			traveller.ServiceSkills,
			traveller.PersonalDevelopment, traveller.ServiceSkills,
			traveller.AdvancedEducationEight, traveller.AdvancedEducation,
		},
		weapons: []traveller.WeaponName{dagger, "Body Pistol", "Cutlass", "Submachine Gun"},
		muster: []traveller.MusterTable{
			traveller.TableTwo,
			traveller.TableOne, traveller.TableOne, traveller.TableOne,
			traveller.TableOne, traveller.TableOne, traveller.TableOne,
		},
	}

	character, err := chargen.Generate(inputs, chosen, chargen.WithRoller(rolls))
	if err != nil {
		t.Fatalf("replaying the example: %v", err)
	}

	if len(rolls.throws) != 0 || len(rolls.dice) != 0 {
		t.Errorf("the example narrates %d throws and %d dice the engine never asked for",
			len(rolls.throws), len(rolls.dice))
	}

	return character
}

// P. 25: "Captain Jamison is now a 38 year old retired merchant captain, UPP
// 779C99."
//
// Holds: The worked example reproduces.
func TestTheWorkedExampleReproduces(t *testing.T) {
	t.Parallel()

	jamison := replayJamison(t)

	if got := jamison.Profile.UPP(); got != "779C99" {
		t.Errorf("UPP %s, want 779C99", got)
	}

	if got := jamison.Age.Years(); got != 38 {
		t.Errorf("aged %v, want 38", jamison.Age)
	}

	if jamison.Age.Months() != 0 {
		t.Errorf("aged %v; no medical crisis occurs, so there are no months", jamison.Age)
	}

	if jamison.Terms != 5 {
		t.Errorf("served %d terms, want 5", jamison.Terms)
	}

	service, served := jamison.ServedIn()
	if !served || service != traveller.Merchants || jamison.RankName != "Captain" {
		t.Errorf("%v %s (served %v), want Merchants Captain", service, jamison.RankName, served)
	}

	// "of rank 5 on the scale of ranks".
	if jamison.Rank != 5 {
		t.Errorf("rank %d, want 5", jamison.Rank)
	}

	// The service dismissed him, but p. 21 makes leaving at the end of the
	// fifth term or later retirement however it came about.
	if _, retired := jamison.Departure.(traveller.Retired); !retired {
		t.Errorf("departed as %T, want retired", jamison.Departure)
	}
}

// The skills inset on p. 25.
//
// Holds: Its skills, as its inset lists them.
func TestTheWorkedExamplesSkills(t *testing.T) {
	t.Parallel()

	want := []string{
		"Body Pistol-1", "Cutlass-1", dagger + "-1", "Electronic-3",
		"Pilot-2", "Submachine Gun-1", "Vacc Suit-1",
	}

	got := make([]string, 0, len(want))
	for _, skill := range replayJamison(t).Skills {
		got = append(got, skill.String())
	}

	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Errorf("skills\n got %v\nwant %v", got, want)
	}
}

// P. 25: "He owns a Type A merchant ship (30 years old) and he owes 10 years
// (120 months) of payments before he will have clear title. ... He has a
// retirement income of CR 4,000 yearly, and has already collected the first
// year's benefit, which, when added to his other monies, gives him a balance
// of CR 24,000."
//
// Holds: Its ship, cash and pension.
func TestTheWorkedExamplesPossessions(t *testing.T) {
	t.Parallel()

	jamison := replayJamison(t)

	if len(jamison.Benefits.Ships) != 1 {
		t.Fatalf("owns %d ships, want one", len(jamison.Benefits.Ships))
	}

	ship := jamison.Benefits.Ships[0]
	if ship.Kind != traveller.FreeTrader || ship.Years != 30 || ship.PaymentYears != 10 {
		t.Errorf("owns a %v, %d years old, with %d years of payments; want a Free Trader, 30 and 10",
			ship.Kind, ship.Years, ship.PaymentYears)
	}

	// Book 2 p. 19: the free trader uses the type 200 hull.
	if ship.Tons != 200 {
		t.Errorf("the ship is %d tons, want 200", ship.Tons)
	}

	if jamison.Benefits.Cash != 20000 {
		t.Errorf("holds %v, want CR 20000", jamison.Benefits.Cash)
	}

	if jamison.Pension != 4000 {
		t.Errorf("draws %v a year, want CR 4000", jamison.Pension)
	}

	// The balance the page reports is the cash plus the first year drawn.
	if want := jamison.Benefits.Cash + jamison.Pension; want != 24000 {
		t.Errorf("cash and the first year's pension come to %v, want CR 24000", want)
	}
}

// The three places the engine and the page part company, each asserted
// rather than merely noted. A departure that is only described is a
// departure nothing would notice closing.
//
// Holds: A printed table governs over the example's stated result.
// Holds: The example's per-term order is not followed.
// Holds: The example's batched aging is not followed.
func TestTheWorkedExamplesDepartures(t *testing.T) {
	t.Parallel()

	jamison := replayJamison(t)

	// E015. The page's six Table 1 rolls yield the same +1 Education and the
	// same four merchant ships either way; the whole visible difference is
	// this one passage. P. 25 says "one middle passage, worth about CR
	// 8,000", but a roll of 3 reaches Merchant row 3, which prints +1 Educ,
	// and no row of that column prints a Middle Passage at all.
	if len(jamison.Benefits.Passages) != 1 {
		t.Fatalf("holds %d passages, want one", len(jamison.Benefits.Passages))
	}

	if got := jamison.Benefits.Passages[0]; got != traveller.LowPassage {
		t.Errorf("holds a %v; Table 1's Merchant row 6 prints Low Psg, and the table governs (E015)", got)
	}

	// E002. The example narrates reenlistment before that term's skills; the
	// exposition's order governs, so the record throws for reenlistment
	// after them. The order is visible in the log and nowhere else.
	var reenlistedBeforeSkills bool

	seenSkillsThisTerm := false

	// Both kinds, because the two halves of this comparison are now
	// different events: a skills table is rolled against nothing and is a
	// RollEvent, while reenlistment is thrown against a target (#50). A walk
	// over throws alone sees the reenlistment and not the skills, and
	// concludes the order is wrong.
	for _, event := range jamison.Events {
		step, rolled := stepOf(event)
		if !rolled {
			continue
		}

		switch step {
		case "Personal Development Table", "Service Skills Table",
			"Advanced Education Table", "Advanced Education Table (education 8+)":
			seenSkillsThisTerm = true
		case "reenlistment":
			if !seenSkillsThisTerm {
				reenlistedBeforeSkills = true
			}

			seenSkillsThisTerm = false
		default:
		}
	}

	if reenlistedBeforeSkills {
		t.Error("a term threw for reenlistment before its skills; E002 puts the exposition's order first")
	}

	// E006. The example resolves both aging rounds at mustering out and says
	// so; the engine runs each at its own term end, which is where the log
	// must show them.
	agingBeforeMuster := 0

	for _, event := range jamison.Events {
		step, isStep := event.(traveller.StepEvent)
		if !isStep {
			continue
		}

		if step.Step == "aging, end of term 4" || step.Step == "aging, end of term 5" {
			agingBeforeMuster++
		}
	}

	if agingBeforeMuster != 2 {
		t.Errorf("found %d aging rounds at term ends, want 2: the page batches them, E006 does not",
			agingBeforeMuster)
	}
}

// stepOf is the step of any event that put dice on the table, whether it had
// a target to meet or not.
//
// The two are separate cases since #50, and a test that walks one kind sees
// half the dice - which is what the first run of that change reported here
// and in the Table 2 cap.
func stepOf(event traveller.Event) (string, bool) {
	switch e := event.(type) {
	case traveller.ThrowEvent:
		return e.Step, true
	case traveller.RollEvent:
		return e.Step, true
	default:
		return "", false
	}
}
