package chargen_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// The service history is the service a term at a time: the grants on
// entering it (E005), each commission or promotion with what p. 23 granted at
// its rank, what each term's skill rolls gave, what aging took, and any
// medical crisis survived (#401).
//
// The literals are typed from each fixture's generation record, which is what the
// history has to agree with, and not from the code that fills it. Three
// fixtures between them reach every part: an Army career for the grant on
// entering and a rank grant that is a skill, a Navy career for one that is a
// characteristic, and an Other career for a crisis survived.
//
// Holds: The service history records each term's rank, gains and aging.
func TestTheServiceHistoryRecordsEachTerm(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]chargen.ServiceHistory{
		"army-serve":            armyServeHistory(),
		"navy-spartan-declines": navySpartanDeclinesHistory(),
		"other-crisis-survived": otherCrisisSurvivedHistory(),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := byName(t, name).generate(t).History
			if !reflect.DeepEqual(got, want) {
				t.Errorf("the service history:\n got %s\nwant %s", historyText(got), historyText(want))
			}
		})
	}
}

// armyServeHistory is army-serve's generation record: Rifle-1 on entering, a
// commission in the first term that grants Submachine Gun-1, a promotion in
// each term to General, and none thrown for in the seventh (E013).
func armyServeHistory() chargen.ServiceHistory {
	return chargen.ServiceHistory{
		OnEntering: gains(skill("Rifle", 1)),
		Terms: []chargen.TermServed{
			{
				Term: 1,
				Raised: []chargen.Raise{{
					How: chargen.Commissioned, Rank: "Lieutenant", Grants: gains(skill("Submachine Gun", 1)),
				}},
				Gained: gains(skill("Tactics", 1), skill("Halberd", 1), skill("Submachine Gun", 2)),
			},
			{Term: 2, Raised: promoted("Captain"), Gained: gains(skill("Gambling", 1), skill("Administration", 1))},
			{Term: 3, Raised: promoted("Major"), Gained: gains(skill("Electronic", 1), skill("ATV", 1))},
			{
				Term: 4, Raised: promoted("Lt Colonel"),
				Gained: gains(change(traveller.Education, 10, 11), skill("Tactics", 2)),
				Aged:   []chargen.Change{change(traveller.Endurance, 8, 7)},
			},
			{
				Term: 5, Raised: promoted("Colonel"),
				Gained: gains(skill("Electronic", 2), skill("ATV", 2)),
				Aged:   []chargen.Change{change(traveller.Endurance, 7, 6)},
			},
			{
				Term: 6, Raised: promoted("General"),
				Gained: gains(change(traveller.Education, 11, 12), skill("Tactics", 3)),
				Aged:   []chargen.Change{change(traveller.Strength, 10, 9)},
			},
			{
				Term: 7, Gained: gains(skill("Mechanical", 1)),
				Aged: []chargen.Change{change(traveller.Endurance, 6, 5)},
			},
		},
	}
}

// navySpartanDeclinesHistory is navy-spartan-declines' generation record: nothing on
// entering the Navy, a commission in the second term, and a promotion to
// Captain in the seventh whose p. 23 grant is Social Standing +1.
func navySpartanDeclinesHistory() chargen.ServiceHistory {
	return chargen.ServiceHistory{
		Terms: []chargen.TermServed{
			{Term: 1, Gained: gains(skill("Navigation", 1), skill("Vacc Suit", 1))},
			{
				Term: 2, Raised: []chargen.Raise{{How: chargen.Commissioned, Rank: "Ensign"}},
				Gained: gains(skill("Ship's Boat", 1), change(traveller.Endurance, 8, 9)),
			},
			{
				Term: 3, Raised: promoted("Lieutenant"),
				Gained: gains(skill("Navigation", 2), skill("Jack of all Trades", 1)),
			},
			{
				Term: 4, Raised: promoted("Lt Cmdr"),
				Gained: gains(skill("Forward Observer", 1), change(traveller.SocialStanding, 12, 13)),
				Aged:   []chargen.Change{change(traveller.Strength, 10, 9), change(traveller.Endurance, 9, 8)},
			},
			{
				Term: 5, Raised: promoted("Commander"),
				Gained: gains(skill("Pilot", 1), skill("Vacc Suit", 2)),
				Aged:   []chargen.Change{change(traveller.Strength, 9, 8)},
			},
			{
				Term: 6, Gained: gains(skill("Ship's Boat", 2)),
				Aged: []chargen.Change{change(traveller.Endurance, 8, 7)},
			},
			{
				Term: 7,
				Raised: []chargen.Raise{{
					How: chargen.Promoted, Rank: "Captain", Grants: gains(change(traveller.SocialStanding, 13, 14)),
				}},
				Gained: gains(change(traveller.Education, 10, 11), skill("Engineer", 1)),
				Aged:   []chargen.Change{change(traveller.Strength, 8, 7), change(traveller.Endurance, 7, 6)},
			},
		},
	}
}

// otherCrisisSurvivedHistory is other-crisis-survived's generation record: no ranks,
// as the Other print none, and Strength taken to zero in the seventh term's
// aging round and recovered from the crisis.
func otherCrisisSurvivedHistory() chargen.ServiceHistory {
	return chargen.ServiceHistory{
		Terms: []chargen.TermServed{
			{Term: 1, Gained: gains(skill("Electronic", 1), skill("Brawling", 1))},
			{Term: 2, Gained: gains(skill("Automatic Rifle", 1))},
			{Term: 3, Gained: gains(skill("Brawling", 2))},
			{
				Term: 4, Gained: gains(skill("Electronic", 2)),
				Aged: []chargen.Change{change(traveller.Strength, 3, 2), change(traveller.Endurance, 10, 9)},
			},
			{
				Term: 5, Gained: gains(skill("Mechanical", 1)),
				Aged: []chargen.Change{change(traveller.Strength, 2, 1), change(traveller.Endurance, 9, 8)},
			},
			{
				Term: 6, Gained: gains(skill("Forgery", 1)),
				Aged: []chargen.Change{change(traveller.Dexterity, 10, 9), change(traveller.Endurance, 8, 7)},
			},
			{
				Term: 7, Gained: gains(change(traveller.SocialStanding, 10, 9)),
				Aged:      []chargen.Change{change(traveller.Strength, 1, 0), change(traveller.Endurance, 7, 6)},
				Recovered: []traveller.Characteristic{traveller.Strength},
			},
		},
	}
}

func skill(name string, level int) chargen.SkillGain {
	return chargen.SkillGain{Skill: traveller.Skill{Name: traveller.SkillName(name), Level: level}}
}

func change(characteristic traveller.Characteristic, from, to int) chargen.Change {
	return chargen.Change{Characteristic: characteristic, From: from, To: to}
}

func promoted(rank string) []chargen.Raise {
	return []chargen.Raise{{How: chargen.Promoted, Rank: rank}}
}

// gains is a list of gains in the order given, which is the order the
// generation record received them in.
func gains(given ...chargen.Gain) []chargen.Gain { return given }

// historyText is a service history a term to a line, so a failure shows
// which term differs.
func historyText(history chargen.ServiceHistory) string {
	var text strings.Builder

	fmt.Fprintf(&text, "\n  entering %+v", history.OnEntering)

	for _, term := range history.Terms {
		fmt.Fprintf(&text, "\n  %+v", term)
	}

	return text.String()
}
