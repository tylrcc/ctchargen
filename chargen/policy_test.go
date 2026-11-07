package chargen_test

import (
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/chargen"
	"github.com/philoserf/ctchargen/traveller"
)

// Words POLICY.md does not carry. They are strings and not strategies,
// which is the point of #41: past the parser there is no way to write one.
const (
	noSuchCareer = "dawdle"
	noSuchSkills = "osmosis"
	noSuchMuster = "gold"
)

// Words it does carry, for the two strategies a case is not about.
const (
	anyCareer = "serve"
	anySkills = "advanced"
	anyMuster = "cash"
)

// Every row of POLICY.md, checked against what the document says. The gate
// in internal/docsgate holds the row set to the interface; this holds each
// row's answer to what the row promises.
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
			traveller.Discharge, false,
		},
	} {
		policy := chargen.Policy{Career: tc.career, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash}

		got, err := policy.ReenlistIntent(tc.offered)
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

		// No counts: an untrained character, where the ranking decides
		// alone and the answer is what it was before counts existed.
		got, err := policy.SkillTable(tc.offered, nil)
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
		"spartan prefers table 1 and declines both modifiers": {
			chargen.MusterSpartan, traveller.TableOne, false,
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
// received. spartan is the strategy that diversifies instead.
func TestMusterWeaponStrategies(t *testing.T) {
	t.Parallel()

	list := []traveller.WeaponName{dagger, "Blade", "Foil"}
	received := []traveller.WeaponName{dagger}

	cash := chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterCash}

	taken, err := cash.MusterWeapon(traveller.Blade, list, received)
	if err != nil {
		t.Fatalf("cash: %v", err)
	}

	if _, expertise := taken.(traveller.TakeExpertise); !expertise {
		t.Errorf("cash took %T on a repeat, want expertise", taken)
	}

	// A first receipt has nothing to take expertise in.
	first, err := cash.MusterWeapon(traveller.Blade, list, nil)
	if err != nil {
		t.Fatalf("cash, first receipt: %v", err)
	}

	if _, weapon := first.(traveller.TakeWeapon); !weapon {
		t.Errorf("cash took %T on a first receipt, want the weapon", first)
	}

	spartan := chargen.Policy{Career: chargen.CareerServe, Skills: chargen.SkillsAdvanced, Muster: chargen.MusterSpartan}

	diversified, err := spartan.MusterWeapon(traveller.Blade, list, received)
	if err != nil {
		t.Fatalf("spartan: %v", err)
	}

	if got, ok := diversified.(traveller.TakeWeapon); !ok || got.Weapon != "Blade" {
		t.Errorf("spartan took %v, want the first weapon not already received", diversified)
	}
}

// Service ranks by the odds of the throw, ties to book order (p. 10).
func TestServicePicksTheLikeliestEnlistment(t *testing.T) {
	t.Parallel()

	target := func(s string) traveller.Target {
		parsed, err := traveller.ParseTarget(s)
		if err != nil {
			t.Fatalf("parsing %q: %v", s, err)
		}

		return parsed
	}

	policy := chargen.DefaultPolicy()

	chosen, err := policy.Service([]traveller.EnlistmentOffer{
		{Service: traveller.Navy, Target: target("8+")},
		{Service: traveller.Other, Target: target("3+")},
		{Service: traveller.Army, Target: target("5+")},
	})
	if err != nil || chosen != traveller.Other {
		t.Errorf("chose %v (%v), want Other, whose 3+ is the likeliest", chosen, err)
	}

	// A modifier is part of the odds.
	chosen, err = policy.Service([]traveller.EnlistmentOffer{
		{Service: traveller.Navy, Target: target("8+"), DM: 6},
		{Service: traveller.Other, Target: target("3+")},
	})
	if err != nil || chosen != traveller.Navy {
		t.Errorf("chose %v (%v), want Navy, whose 8+ with a +6 is certain", chosen, err)
	}
}

// A word no row of POLICY.md carries never becomes a strategy.
//
// This is the whole of what Policy.Validate and the engine's validate() used
// to do, moved to the one place a string is supposed to become a domain
// value. Past the parser the type carries it, which is why nothing downstream
// checks any more (#41).
//
// Holds: A strategy no row of POLICY.md carries never becomes one.
func TestAWordNoRowCarriesIsNotAStrategy(t *testing.T) {
	t.Parallel()

	_, career := chargen.ParsePolicy(noSuchCareer, anySkills, anyMuster)
	_, skills := chargen.ParsePolicy(anyCareer, noSuchSkills, anyMuster)
	_, muster := chargen.ParsePolicy(anyCareer, anySkills, noSuchMuster)

	for name, err := range map[string]error{
		"career": career, "skills": skills, "muster": muster,
	} {
		if err == nil {
			t.Errorf("%s: a word no row carries was accepted", name)
		}
	}
}

// Every strategy POLICY.md does carry parses, and parses back to its own
// spelling. The round trip is what holds the parser to String: the record
// writes the one and the flag reads the other, and a disagreement would make
// a record name a strategy the flag refuses.
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
		chargen.MusterCash, chargen.MusterGoods, chargen.MusterSpartan,
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
// a binary and no tree, so a message pointing at POLICY.md - which this one
// used to do - names a document he cannot open. The values come from the
// same function the --help text reads, so the two cannot disagree.
func TestTheRejectionNamesTheValuesAndNoDocument(t *testing.T) {
	t.Parallel()

	_, err := chargen.ParsePolicy(anyCareer, anySkills, noSuchMuster)
	if err == nil {
		t.Fatalf("%q was accepted", noSuchMuster)
	}

	for _, want := range []string{
		chargen.MusterCash.String(), chargen.MusterGoods.String(), chargen.MusterSpartan.String(),
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the rejection %q does not offer %q", err, want)
		}
	}

	if strings.Contains(err.Error(), ".md") {
		t.Errorf("the rejection %q names a file the reader may not have", err)
	}
}
