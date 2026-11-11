package chargen_test

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// Words that name no strategy. They are strings and not strategies, which is
// the point of #41: past the parser there is no way to write one.
const (
	noSuchCareer = "dawdle"
	noSuchSkills = "osmosis"
	noSuchMuster = "gold"
)

// Words that do, for the two strategies a case is not about.
const (
	anyCareer = "serve"
	anySkills = "advanced"
	anyMuster = "cash"
)

// Every career answer, checked against what the method's doc says.
func TestCareerStrategies(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		career  chargen.Career
		offered []traveller.Intent
		want    traveller.Intent
		draft   bool
	}{
		"serve continues while it can": {
			chargen.CareerServe,
			[]traveller.Intent{traveller.Continue, traveller.Retire},
			traveller.Continue, true,
		},
		"serve retires when it must": {
			chargen.CareerServe, []traveller.Intent{traveller.Retire}, traveller.Retire, true,
		},
		"retire leaves as soon as leaving is retirement": {
			chargen.CareerRetire,
			[]traveller.Intent{traveller.Continue, traveller.Retire},
			traveller.Retire, true,
		},
		"retire continues while it cannot retire": {
			chargen.CareerRetire,
			[]traveller.Intent{traveller.Continue, traveller.Discharge},
			traveller.Continue, true,
		},
		"oneterm leaves at the first opportunity": {
			chargen.CareerOneTerm,
			[]traveller.Intent{traveller.Continue, traveller.Discharge},
			traveller.Discharge, true,
		},
	} {
		policy := chargen.Policy{Career: tc.career, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash}

		got, err := policy.ReenlistIntent(tc.offered, traveller.Odds{})
		if err != nil || got != tc.want {
			t.Errorf("%s: chose %v (%v), want %v", name, got, err, tc.want)
		}

		submit, err := policy.SubmitToDraft()
		if err != nil || submit != tc.draft {
			t.Errorf("%s: submits to the draft = %v (%v), want %v", name, submit, err, tc.draft)
		}
	}
}

func TestSkillsStrategies(t *testing.T) {
	t.Parallel()

	all := traveller.SkillTables[:]
	withoutTheGate := all[:len(all)-1]

	for name, tc := range map[string]struct {
		strategy chargen.Skills
		offered  []traveller.SkillTable
		want     traveller.SkillTable
	}{
		"advanced takes the fourth table the instant it opens": {
			chargen.SkillsAdvanced, all, traveller.AdvancedEducationEight,
		},
		"advanced falls back when the gate is shut": {
			chargen.SkillsAdvanced, withoutTheGate, traveller.AdvancedEducation,
		},
		"service prefers the service table": {
			chargen.SkillsService, all, traveller.ServiceSkills,
		},
		"personal reaches the one negative result": {
			chargen.SkillsPersonal, all, traveller.PersonalDevelopment,
		},
	} {
		policy := chargen.Policy{Career: chargen.CareerServe, Skills: tc.strategy, Muster: chargen.MusterCash}

		// No results off any table yet: an untrained character, where the
		// ranking decides alone and the answer is what it was before counts
		// existed.
		offers := make([]traveller.SkillTableOffer, 0, len(tc.offered))
		for _, table := range tc.offered {
			offers = append(offers, traveller.SkillTableOffer{Table: table})
		}

		got, err := policy.SkillTable(offers, traveller.SkillRoll{})
		if err != nil || got != tc.want {
			t.Errorf("%s: designated %v (%v), want %v", name, got, err, tc.want)
		}
	}
}

func TestMusterStrategies(t *testing.T) {
	t.Parallel()

	both := traveller.MusterTables[:]

	for name, tc := range map[string]struct {
		strategy chargen.Muster
		want     traveller.MusterTable
		takesDMs bool
	}{
		"cash prefers table 2, which is what reaches the three-roll cap": {
			chargen.MusterCash, traveller.TableTwo, true,
		},
		"goods prefers table 1, which is what reaches the ships": {
			chargen.MusterGoods, traveller.TableOne, true,
		},
		"arsenal prefers table 1, and takes both modifiers as the others do": {
			chargen.MusterArsenal, traveller.TableOne, true,
		},
	} {
		policy := chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: tc.strategy}

		got, err := policy.MusterTable(both)
		if err != nil || got != tc.want {
			t.Errorf("%s: designated %v (%v), want %v", name, got, err, tc.want)
		}

		one, _ := policy.MusterTable1DM()
		two, _ := policy.MusterTable2DM()

		if one != tc.takesDMs || two != tc.takesDMs {
			t.Errorf("%s: takes the modifiers = %v and %v, want %v", name, one, two, tc.takesDMs)
		}
	}
}

// P. 22 offers expertise "in lieu of receiving a second or subsequent weapon
// of exactly the same type", so it can only be taken in one already
// received. cash and goods take it on a repeat; arsenal diversifies instead.
// A weapon taken is drawn, never the first printed (#570): each case's draw
// names a weapon the first printed is not, and says how many it drew among.
//
// Holds: The weapon for a weapon benefit is drawn, not always the first
// printed.
func TestMusterWeaponStrategies(t *testing.T) {
	t.Parallel()

	const blade, foil traveller.WeaponName = "Blade", "Foil"

	list := []traveller.WeaponName{dagger, blade, foil}

	for name, tc := range map[string]struct {
		strategy chargen.Muster
		received []traveller.WeaponName
		draws    []int
		want     traveller.WeaponBenefit
		among    []int // the size of each draw the policy made
	}{
		"cash draws a first receipt": {
			chargen.MusterCash, nil, []int{2}, traveller.TakeWeapon{Weapon: foil}, []int{3},
		},
		"goods draws a first receipt": {
			chargen.MusterGoods, nil, []int{1}, traveller.TakeWeapon{Weapon: blade}, []int{3},
		},
		"cash takes expertise on a repeat, and draws nothing": {
			chargen.MusterCash, []traveller.WeaponName{dagger}, nil, traveller.TakeExpertise{Weapon: dagger}, nil,
		},
		"goods takes expertise on a repeat, and draws nothing": {
			chargen.MusterGoods, []traveller.WeaponName{foil}, nil, traveller.TakeExpertise{Weapon: foil}, nil,
		},
		"arsenal draws a first receipt": {
			chargen.MusterArsenal, nil, []int{2}, traveller.TakeWeapon{Weapon: foil}, []int{3},
		},
		"arsenal draws among the weapons not yet received": {
			chargen.MusterArsenal, []traveller.WeaponName{dagger}, []int{1}, traveller.TakeWeapon{Weapon: foil}, []int{2},
		},
		"arsenal draws from the whole list once it has received every one": {
			chargen.MusterArsenal, list, []int{1}, traveller.TakeWeapon{Weapon: blade}, []int{3},
		},
	} {
		policy := chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: tc.strategy}
		vary := &drawn{draws: tc.draws}

		got, err := policy.MusterWeapon(traveller.Blade, list, tc.received, vary)
		if err != nil || got != tc.want {
			t.Errorf("%s: took %v (%v), want %v", name, got, err, tc.want)
		}

		if !slices.Equal(vary.among, tc.among) {
			t.Errorf("%s: drew among %v, want %v", name, vary.among, tc.among)
		}
	}
}

// The draft is a question, and a person may answer no: generation ends there,
// with an 18-year-old civilian who served no term, stamped with E001. No
// strategy declines (#566), so a guided run is what reaches it.
//
// The walk is played past its script rather than strictly: an engine that
// went on after a declined draft would otherwise fail as a script run dry,
// which says nothing of the draft. Past the script the dice are any dice, and
// the outcome is judged first; the script is held to its end after.
//
// Holds: The draft is offered, not compelled.
// Holds: Declining ends generation with a civilian.
func TestDecliningTheDraftEndsWithACivilian(t *testing.T) {
	t.Parallel()

	roller := &pastTheScript{dice: declinedTheDraft.dice, rest: rand.New(rand.NewPCG(566, 1))}

	character, err := chargen.Guided(declinedTheDraft.inputs(),
		refusing{Policy: declinedTheDraft.policy, at: traveller.ChoiceSubmitToDraft}, chargen.WithRoller(roller))
	if err != nil {
		t.Fatal(err)
	}

	asked := answersAt(character, traveller.ChoiceSubmitToDraft)
	if !slices.Equal(asked, []string{"no"}) {
		t.Fatalf("the draft was answered %v, want one no", asked)
	}

	if _, declined := character.Enlistment().(traveller.DeclinedTheDraft); !declined {
		t.Errorf("the draft was declined, and the enlistment is %v; want the draft declined", character.Enlistment())
	}

	if service, served := character.ServedIn(); served || character.Terms() != 0 || character.Age().Years() != 18 {
		t.Errorf("the draft was declined, and generation went on: served %v (%v) for %d terms to age %v; "+
			"want no service, no term, age 18", service, served, character.Terms(), character.Age())
	}

	if !slices.Contains(character.Errata, traveller.E001) {
		t.Errorf("the draft was declined, and E001 is not stamped: %v", character.Errata)
	}

	if roller.past > 0 || roller.read != len(declinedTheDraft.dice) {
		t.Errorf("%s read %d of its %d dice and asked for %d past them",
			declinedTheDraft.name, roller.read, len(declinedTheDraft.dice), roller.past)
	}
}

// pastTheScript plays a script's dice and, past their end, hands out any
// dice and draws at all, counting them rather than failing.
type pastTheScript struct {
	dice       []int
	read, past int
	rest       *rand.Rand
}

func (p *pastTheScript) Die() int {
	if p.read < len(p.dice) {
		p.read++

		return p.dice[p.read-1]
	}

	p.past++

	return p.rest.IntN(6) + 1
}

func (p *pastTheScript) Among(n int) int {
	p.past++

	return p.rest.IntN(n)
}

// A rank of 5 or 6 allows +1 on Table 1, and a person may decline it: the
// roll is then read at its dice alone (p. 9: "may add"). Every strategy takes
// it (#570), so a guided run is what reaches the refusal.
//
// Holds: The +1 on Table 1 may be declined.
func TestTheTableOneModifierMayBeDeclined(t *testing.T) {
	t.Parallel()

	character := declinedTheTableOneModifier.walk(t)

	asked := answersAt(character, traveller.ChoiceMusterTable1DM)
	if len(asked) == 0 {
		t.Fatalf("%s was never offered the +1, so this checks nothing", declinedTheTableOneModifier.name)
	}

	declined := false

	for _, event := range character.Events {
		switch e := event.(type) {
		case traveller.ChoiceEvent:
			declined = e.Point == traveller.ChoiceMusterTable1DM && e.Chosen == "no"
		case traveller.RollEvent:
			if declined && e.DM != 0 {
				t.Errorf("the +1 was declined, and the roll after it was read at %v %+d", e.Dice, e.DM)
			}

			declined = false
		}
	}

	if slices.ContainsFunc(asked, func(answer string) bool { return answer != "no" }) {
		t.Errorf("the +1 was answered %v, want no every time", asked)
	}
}

// answersAt is every answer the record gives at one choice point, in order.
func answersAt(character *chargen.Character, point traveller.ChoicePoint) []string {
	var chosen []string

	for _, event := range character.Events {
		if choice, ok := event.(traveller.ChoiceEvent); ok && choice.Point == point {
			chosen = append(chosen, choice.Chosen)
		}
	}

	return chosen
}

// A word that names no strategy never becomes one.
//
// This is the whole of what Policy.Validate and the engine's validate() used
// to do, moved to the one place a string is supposed to become a domain
// value. Past the parser the type carries it, which is why nothing downstream
// checks any more (#41).
//
// Holds: A word that names no strategy never becomes one.
func TestAWordThatNamesNoStrategyIsNotOne(t *testing.T) {
	t.Parallel()

	_, career := chargen.ParsePolicy(noSuchCareer, anySkills, anyMuster)
	_, skills := chargen.ParsePolicy(anyCareer, noSuchSkills, anyMuster)
	_, muster := chargen.ParsePolicy(anyCareer, anySkills, noSuchMuster)

	for name, err := range map[string]error{
		"career": career, "skills": skills, "muster": muster,
	} {
		if err == nil {
			t.Errorf("%s: a word that names no strategy was accepted", name)
		}
	}
}

// Every strategy parses, and parses back to its own spelling. The round trip
// is what holds the parser to String: the record writes the one and the flag
// reads the other, and a disagreement would make a record name a strategy the
// flag refuses.
func TestEveryStrategyParsesBackToItsSpelling(t *testing.T) {
	t.Parallel()

	for _, want := range []chargen.Career{
		chargen.CareerServe, chargen.CareerRetire, chargen.CareerOneTerm,
	} {
		got, err := chargen.ParsePolicy(want.String(), anySkills, anyMuster)
		if err != nil || got.Career != want {
			t.Errorf("career %q parsed to %v (%v)", want, got, err)
		}
	}

	for _, want := range []chargen.Skills{
		chargen.SkillsAdvanced, chargen.SkillsService, chargen.SkillsPersonal,
	} {
		got, err := chargen.ParsePolicy(anyCareer, want.String(), anyMuster)
		if err != nil || got.Skills != want {
			t.Errorf("skills %q parsed to %v (%v)", want, got, err)
		}
	}

	for _, want := range []chargen.Muster{
		chargen.MusterCash, chargen.MusterGoods, chargen.MusterArsenal,
	} {
		got, err := chargen.ParsePolicy(anyCareer, anySkills, want.String())
		if err != nil || got.Muster != want {
			t.Errorf("muster %q parsed to %v (%v)", want, got, err)
		}
	}
}

// The rejection tells the reader what to type, and names no file.
//
// The reader most likely to see it is the one who typed `go install`: he has
// a binary and no tree, so a message pointing at a repository document -
// which this one used to do - names a document he cannot open. The values
// come from the same function the --help text reads, so the two cannot
// disagree.
func TestTheRejectionNamesTheValuesAndNoDocument(t *testing.T) {
	t.Parallel()

	_, err := chargen.ParsePolicy(anyCareer, anySkills, noSuchMuster)
	if err == nil {
		t.Fatalf("%q was accepted", noSuchMuster)
	}

	for _, want := range []string{
		chargen.MusterCash.String(), chargen.MusterGoods.String(), chargen.MusterArsenal.String(),
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the rejection %q does not offer %q", err, want)
		}
	}

	if strings.Contains(err.Error(), ".md") {
		t.Errorf("the rejection %q names a file the reader may not have", err)
	}
}

// drawn is a Vary that answers each draw from a script, and keeps how many
// alternatives each draw was among.
type drawn struct {
	draws []int
	among []int
}

func (d *drawn) Among(n int) int {
	d.among = append(d.among, n)

	next := d.draws[0]

	d.draws = d.draws[1:]

	return next
}

// Service draws the service from the offer, uniformly, with the run's own
// dice: every one of the six is reachable, each by its place in book order
// (p. 10), whatever its odds (#420). The best-odds rule made an army.
//
// Holds: The service is drawn, so every service is reachable.
func TestServiceIsDrawnFromEveryService(t *testing.T) {
	t.Parallel()

	offers := make([]traveller.EnlistmentOffer, 0, len(traveller.ServiceNames))
	for _, service := range traveller.ServiceNames {
		offers = append(offers, traveller.EnlistmentOffer{Service: service})
	}

	want := []traveller.ServiceName{
		traveller.Navy, traveller.Marines, traveller.Army,
		traveller.Scouts, traveller.Merchants, traveller.Other,
	}

	for draw, service := range want {
		for _, policy := range []chargen.Policy{
			chargen.DefaultPolicy(),
			{Career: chargen.CareerOneTerm, Skills: chargen.SkillsPersonal, Muster: chargen.MusterGoods},
		} {
			chosen, err := policy.Service(offers, &drawn{draws: []int{draw}})
			if err != nil || chosen != service {
				t.Errorf("%v drew %d and chose %v (%v), want %v", policy, draw, chosen, err, service)
			}
		}
	}
}
