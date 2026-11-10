package rules_test

import (
	"fmt"
	"testing"

	"github.com/philoserf/ctchargen/internal/rules"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// The second transcription.
//
// Every numeric table of Book 1 pp. 4-25 is written down twice: once in
// transcription.go, and once here. The reprint's embedded font substitutes glyphs
// that look like data - on p. 9 the dash cells of Mustering Out Table 1
// extract as the digit 4, and "Travellers' Aid" extracts as Travellers9 -
// so a table can only be trusted if two readings of the page agree.
//
// The two copies are deliberately traversed in different directions. The
// first transcription is row-major, as the pages print them: one row of the table,
// six services across. Everything below is column-major: one service, down
// its whole column. A slip made walking a row is unlikely to be repeated
// walking a column, which is the only independence a single reading pass can
// honestly offer.
//
// Names here are the ones E012 normalizes to, because that is what the lift
// produces. Applying a documented reading by hand is not copying the first
// transcription; a mis-transcribed cell still shows up as a mismatch.

func load(t *testing.T) *rules.Rules {
	t.Helper()

	r, err := rules.Load()
	if err != nil {
		t.Fatalf("the tables do not lift: %v", err)
	}

	return r
}

// resultWordsFold renders a skills-table cell. It folds, so a case added to
// the sum stops this file compiling too.
type resultWordsFold struct{ text string }

func (d *resultWordsFold) Alteration(c traveller.Characteristic, delta int) error {
	d.text = fmt.Sprintf("%+d %v", delta, c)

	return nil
}

func (d *resultWordsFold) Skill(n traveller.SkillName) error {
	d.text = string(n)

	return nil
}

func (d *resultWordsFold) WeaponPick(c traveller.WeaponCategory) error {
	d.text = "weapon " + c.String()

	return nil
}

func describe(t *testing.T, result traveller.TableResult) string {
	t.Helper()

	var d resultWordsFold

	err := result.Fold(&d)
	if err != nil {
		t.Fatalf("describing %v: %v", result, err)
	}

	return d.text
}

// benefitWordsFold renders a Mustering Out Table 1 cell, the same way.
type benefitWordsFold struct{ text string }

func (b *benefitWordsFold) Passage(p traveller.PassageClass) error {
	b.text = p.String()

	return nil
}

func (b *benefitWordsFold) Alteration(c traveller.Characteristic, delta int) error {
	b.text = fmt.Sprintf("%+d %v", delta, c)

	return nil
}

func (b *benefitWordsFold) WeaponPick(c traveller.WeaponCategory) error {
	b.text = "weapon " + c.String()

	return nil
}

func (b *benefitWordsFold) TravellersAid() error {
	b.text = "Travellers' Aid"

	return nil
}

func (b *benefitWordsFold) Ship(k traveller.ShipKind) error {
	b.text = k.String()

	return nil
}

func (b *benefitWordsFold) Nothing() error {
	b.text = "nothing"

	return nil
}

func describeBenefit(t *testing.T, row traveller.BenefitRow) string {
	t.Helper()

	var b benefitWordsFold

	err := row.Fold(&b)
	if err != nil {
		t.Fatalf("describing %v: %v", row, err)
	}

	return b.text
}

// column is one service's whole column, read top to bottom.
type column struct {
	enlistment string
	enlistDMs  []string
	draft      int
	survival   string
	survivalDM []string
	commission string
	commDM     []string
	promotion  string
	promoDM    []string
	reenlist   string
	ranks      []string
	skills     [4][6]string
	benefits   [7]string
	cash       [7]traveller.Credits
}

// The Prior Service Table and the Table of Ranks (p. 10), the Acquired
// Skills Table (p. 11) and both Mustering Out Tables (p. 9), one column at a
// time.
var columns = map[traveller.ServiceName]column{
	traveller.Navy: {
		enlistment: "8+", enlistDMs: []string{"+1 Intelligence 8+", "+2 Education 9+"},
		draft:    1,
		survival: "5+", survivalDM: []string{"+2 Intelligence 7+"},
		commission: "10+", commDM: []string{"+1 Social Standing 9+"},
		promotion: "8+", promoDM: []string{"+1 Education 8+"},
		reenlist: "6+",
		ranks:    []string{"Ensign", "Lieutenant", "Lt Cmdr", "Commander", "Captain", "Admiral"},
		skills: [4][6]string{
			{"+1 Strength", "+1 Dexterity", "+1 Endurance", "+1 Social Standing", "+1 Intelligence", "+1 Education"},
			{"Ship's Boat", "Vacc Suit", "Forward Observer", "weapon Blade Combat", "weapon Gun Combat", "Gunnery"},
			{"Vacc Suit", "Mechanical", "Electronic", "Engineer", "Gunnery", "Jack of all Trades"},
			{"Medical", "Navigation", "Engineer", "Computer", "Pilot", "Administration"},
		},
		benefits: [7]string{
			"Low Passage", "+1 Intelligence", "+2 Education", "weapon Blade Combat",
			"Travellers' Aid", "High Passage", "+2 Social Standing",
		},
		cash: [7]traveller.Credits{1000, 5000, 5000, 10000, 20000, 50000, 50000},
	},
	traveller.Marines: {
		enlistment: "9+", enlistDMs: []string{"+1 Intelligence 8+", "+2 Strength 8+"},
		draft:    2,
		survival: "6+", survivalDM: []string{"+2 Endurance 8+"},
		commission: "9+", commDM: []string{"+1 Education 7+"},
		promotion: "9+", promoDM: []string{"+1 Social Standing 8+"},
		reenlist: "6+",
		ranks:    []string{"Lieutenant", "Captain", "Force Cmdr", "Lt Colonel", "Colonel", "Brigadier"},
		skills: [4][6]string{
			{"+1 Strength", "+1 Dexterity", "+1 Endurance", "Gambling", "Brawling", "weapon Blade Combat"},
			{"ATV", "Vacc Suit", "weapon Blade Combat", "weapon Blade Combat", "weapon Gun Combat", "weapon Gun Combat"},
			{"ATV", "Mechanical", "Electronic", "Tactics", "weapon Blade Combat", "weapon Gun Combat"},
			{"Medical", "Tactics", "Tactics", "Computer", "Leader", "Administration"},
		},
		benefits: [7]string{
			"Low Passage", "+2 Intelligence", "+1 Education", "weapon Blade Combat",
			"Travellers' Aid", "High Passage", "+2 Social Standing",
		},
		cash: [7]traveller.Credits{2000, 5000, 5000, 10000, 20000, 30000, 40000},
	},
	traveller.Army: {
		enlistment: "5+", enlistDMs: []string{"+1 Dexterity 6+", "+2 Endurance 5+"},
		draft:    3,
		survival: "5+", survivalDM: []string{"+2 Education 6+"},
		commission: "5+", commDM: []string{"+1 Endurance 7+"},
		promotion: "6+", promoDM: []string{"+1 Education 7+"},
		reenlist: "7+",
		ranks:    []string{"Lieutenant", "Captain", "Major", "Lt Colonel", "Colonel", "General"},
		skills: [4][6]string{
			{"+1 Strength", "+1 Dexterity", "+1 Endurance", "Gambling", "Brawling", "+1 Education"},
			{"ATV", "Air/Raft", "Forward Observer", "weapon Blade Combat", "weapon Gun Combat", "weapon Gun Combat"},
			{"ATV", "Mechanical", "Electronic", "Tactics", "weapon Blade Combat", "weapon Gun Combat"},
			{"Medical", "Tactics", "Tactics", "Computer", "Leader", "Administration"},
		},
		benefits: [7]string{
			"Low Passage", "+1 Intelligence", "+2 Education", "weapon Gun Combat",
			"High Passage", "Middle Passage", "+1 Social Standing",
		},
		cash: [7]traveller.Credits{2000, 5000, 10000, 10000, 10000, 20000, 30000},
	},
	traveller.Scouts: {
		enlistment: "7+", enlistDMs: []string{"+1 Intelligence 6+", "+2 Strength 8+"},
		draft:    4,
		survival: "7+", survivalDM: []string{"+2 Endurance 9+"},
		reenlist: "3+",
		skills: [4][6]string{
			{"+1 Strength", "+1 Dexterity", "+1 Endurance", "weapon Gun Combat", "+1 Intelligence", "+1 Education"},
			{"Air/Raft", "Vacc Suit", "Navigation", "Mechanical", "Electronic", "Jack of all Trades"},
			{"Air/Raft", "Mechanical", "Electronic", "Jack of all Trades", "Gunnery", "Medical"},
			{"Medical", "Navigation", "Engineer", "Computer", "Pilot", "Jack of all Trades"},
		},
		benefits: [7]string{
			"Low Passage", "+2 Intelligence", "+2 Education", "weapon Blade Combat",
			"weapon Gun Combat", "Scout ship, Type S", "nothing",
		},
		cash: [7]traveller.Credits{20000, 20000, 30000, 30000, 50000, 50000, 50000},
	},
	traveller.Merchants: {
		enlistment: "7+", enlistDMs: []string{"+1 Strength 7+", "+2 Intelligence 6+"},
		draft:    5,
		survival: "5+", survivalDM: []string{"+2 Intelligence 7+"},
		commission: "4+", commDM: []string{"+1 Intelligence 6+"},
		promotion: "10+", promoDM: []string{"+1 Intelligence 9+"},
		reenlist: "4+",
		ranks:    []string{"4th Officer", "3rd Officer", "2nd Officer", "1st Officer", "Captain"},
		skills: [4][6]string{
			{"+1 Strength", "+1 Dexterity", "+1 Endurance", "+1 Strength", "weapon Blade Combat", "Bribery"},
			{"Steward", "Vacc Suit", "weapon Blade Combat", "weapon Gun Combat", "Electronic", "Jack of all Trades"},
			{"Streetwise", "Mechanical", "Electronic", "Navigation", "Gunnery", "Medical"},
			{"Medical", "Navigation", "Engineer", "Computer", "Pilot", "Administration"},
		},
		benefits: [7]string{
			"Low Passage", "+1 Intelligence", "+1 Education", "weapon Gun Combat",
			"weapon Blade Combat", "Low Passage", "Free Trader, Type A",
		},
		cash: [7]traveller.Credits{1000, 5000, 10000, 20000, 20000, 40000, 40000},
	},
	traveller.Other: {
		enlistment: "3+", enlistDMs: nil,
		draft:    6,
		survival: "5+", survivalDM: []string{"+2 Intelligence 9+"},
		reenlist: "5+",
		skills: [4][6]string{
			{"+1 Strength", "+1 Dexterity", "+1 Endurance", "weapon Blade Combat", "Brawling", "-1 Social Standing"},
			{"Forgery", "Gambling", "Brawling", "weapon Blade Combat", "weapon Gun Combat", "Bribery"},
			{"Streetwise", "Mechanical", "Electronic", "Gambling", "Brawling", "Forgery"},
			{"Medical", "Forgery", "Electronic", "Computer", "Streetwise", "Jack of all Trades"},
		},
		benefits: [7]string{
			"Low Passage", "+1 Intelligence", "+1 Education", "weapon Gun Combat",
			"High Passage", "nothing", "nothing",
		},
		cash: [7]traveller.Credits{1000, 5000, 10000, 10000, 10000, 50000, 100000},
	},
}

func describeDMs(dms []rules.DM) []string {
	if len(dms) == 0 {
		return nil
	}

	out := make([]string, 0, len(dms))
	for _, dm := range dms {
		out = append(out, fmt.Sprintf("%+d %v %v", dm.Amount, dm.Characteristic, dm.Threshold))
	}

	return out
}

func TestPriorServiceTable(t *testing.T) {
	t.Parallel()

	r := load(t)

	for name, want := range columns {
		service := r.Service(name)

		check := func(what, got, expected string) {
			if got != expected {
				t.Errorf("%v %s: %s, want %s", name, what, got, expected)
			}
		}
		check("enlistment", service.Enlistment.Target.String(), want.enlistment)
		check("survival", service.Survival.Target.String(), want.survival)
		check("reenlist", service.Reenlist.String(), want.reenlist)

		if service.Draft != want.draft {
			t.Errorf("%v draft: %d, want %d", name, service.Draft, want.draft)
		}

		checkDMs := func(what string, dms []rules.DM, expected []string) {
			got := describeDMs(dms)
			if fmt.Sprint(got) != fmt.Sprint(expected) {
				t.Errorf("%v %s DMs: %v, want %v", name, what, got, expected)
			}
		}
		checkDMs("enlistment", service.Enlistment.DMs, want.enlistDMs)
		checkDMs("survival", service.Survival.DMs, want.survivalDM)

		commission, commissions := service.Commission()
		promotion, _ := service.Promotion()

		if commissions != (want.commission != "") {
			t.Errorf("%v commissions: %v, want %v", name, commissions, want.commission != "")
		}

		if commissions {
			check("commission", commission.Target.String(), want.commission)
			check("promotion", promotion.Target.String(), want.promotion)
			checkDMs("commission", commission.DMs, want.commDM)
			checkDMs("promotion", promotion.DMs, want.promoDM)
		}
	}
}

// Holds: Ranks and their names, per the Table of Ranks.
func TestTableOfRanks(t *testing.T) {
	t.Parallel()

	r := load(t)

	for name, want := range columns {
		service := r.Service(name)

		if got := int(service.MaxRank()); got != len(want.ranks) {
			t.Errorf("%v: %d ranks, want %d", name, got, len(want.ranks))

			continue
		}

		for i, title := range want.ranks {
			got, ok := service.RankName(traveller.Rank(i + 1))
			if !ok || got != title {
				t.Errorf("%v rank %d: %q, want %q", name, i+1, got, title)
			}
		}

		// Rank 0 is "not commissioned" and the table prints no title for
		// it, nor for anything past the column's end.
		if _, ok := service.RankName(0); ok {
			t.Errorf("%v: rank 0 has a title", name)
		}

		if _, ok := service.RankName(service.MaxRank() + 1); ok {
			t.Errorf("%v: a rank past the table has a title", name)
		}
	}
}

func TestAcquiredSkillsTable(t *testing.T) {
	t.Parallel()

	r := load(t)

	for name, want := range columns {
		service := r.Service(name)

		for i, table := range traveller.SkillTables {
			for die := 1; die <= rules.Faces; die++ {
				result, err := service.Result(table, die)
				if err != nil {
					t.Fatalf("%v %v %d: %v", name, table, die, err)
				}

				if got := describe(t, result); got != want.skills[i][die-1] {
					t.Errorf("%v, %v, roll %d: %q, want %q", name, table, die, got, want.skills[i][die-1])
				}
			}
		}
	}
}

func TestMusteringOutTables(t *testing.T) {
	t.Parallel()

	r := load(t)

	for name, want := range columns {
		service := r.Service(name)
		for row := 1; row <= rules.MusterRows; row++ {
			benefit, cash, err := service.Row(row)
			if err != nil {
				t.Fatalf("%v row %d: %v", name, row, err)
			}

			if got := describeBenefit(t, benefit); got != want.benefits[row-1] {
				t.Errorf("%v, table 1, row %d: %q, want %q", name, row, got, want.benefits[row-1])
			}

			if cash != want.cash[row-1] {
				t.Errorf("%v, table 2, row %d: %v, want %v", name, row, cash, want.cash[row-1])
			}
		}
	}
}

// The three cells that are the font trap: under text extraction each comes
// out as the digit 4, which would give a Scout a seventh benefit he does not
// have and give Other two benefits at rows 6 and 7.
//
// Holds: The dash rows deliver nothing.
func TestTheDashCellsAreNothing(t *testing.T) {
	t.Parallel()

	r := load(t)

	dashes := []struct {
		service traveller.ServiceName
		row     int
	}{
		{traveller.Scouts, 7},
		{traveller.Other, 6},
		{traveller.Other, 7},
	}

	found := 0

	for _, name := range traveller.ServiceNames {
		for row := 1; row <= rules.MusterRows; row++ {
			benefit, _, err := r.Service(name).Row(row)
			if err != nil {
				t.Fatal(err)
			}

			if _, isNothing := benefit.(traveller.NoBenefit); isNothing {
				found++
			}
		}
	}

	if found != len(dashes) {
		t.Errorf("Table 1 has %d dash cells, want exactly %d", found, len(dashes))
	}

	for _, d := range dashes {
		benefit, _, err := r.Service(d.service).Row(d.row)
		if err != nil {
			t.Fatal(err)
		}

		if _, isNothing := benefit.(traveller.NoBenefit); !isNothing {
			t.Errorf("%v row %d is %v, want nothing", d.service, d.row, describeBenefit(t, benefit))
		}
	}
}
