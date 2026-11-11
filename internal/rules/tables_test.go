package rules_test

import (
	"fmt"
	"testing"

	"github.com/philoserf/ctchargen/internal/rules"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// The rest of the second transcription: the tables that are not a grid of
// six services.

// The last term the Aging Table's Term of Service header row prints.
//
// The second transcription of that one number (p. 9): the header runs 4
// through 14, so 14 is the last term the table names. It is transcribed here
// rather than derived from the bands because it cannot be - the last band
// begins at term 12 and is open-ended, so nothing in the band list says where
// the printed header stops.
//
// It earns its own test because it is E014's stamping condition: past it a
// record declares that the reading of an open-ended last column governed it.
// A wrong number here does not misplay a character - the effects are the same
// either side - it misreports which readings a record rests on, and only this
// would notice.
//
// Holds: The table's last printed term, which decides where E014 governs.
func TestTheAgingTablesLastPrintedTerm(t *testing.T) {
	t.Parallel()

	const printed = 14

	if got := load(t).Aging().LastPrintedTerm(); got != printed {
		t.Errorf("last printed term %d, want %d - p. 9's header row runs 4 through 14", got, printed)
	}
}

// The Aging Table (p. 9). Its three column groups are read off the
// Intelligence row, whose "no effect before age 66" ends where age 66 - the
// term 12 column - begins. Education and Social Standing carry "unaffected
// by aging" across the whole width.
//
// Holds: Read off the table's term row, not its age row.
// Holds: Education and Social Standing are unaffected by aging.
// Holds: Intelligence is unaffected before age 66.
func TestAgingTable(t *testing.T) {
	t.Parallel()

	r := load(t)

	for _, tc := range []struct {
		terms []traveller.Term
		want  []string
	}{
		{
			terms: []traveller.Term{4, 5, 6, 7},
			want: []string{
				"Strength -1 8+", "Dexterity -1 7+", "Endurance -1 8+",
			},
		},
		{
			terms: []traveller.Term{8, 9, 10, 11},
			want: []string{
				"Strength -1 9+", "Dexterity -1 8+", "Endurance -1 9+",
			},
		},
		{
			// The last column is terminal (E014): the Age row's 74+ is the
			// only cell in either header carrying a plus.
			terms: []traveller.Term{12, 13, 14, 15, 40},
			want: []string{
				"Strength -2 9+", "Dexterity -2 9+", "Endurance -2 9+", "Intelligence -1 9+",
			},
		},
	} {
		for _, term := range tc.terms {
			effects := r.Aging().At(term)

			got := make([]string, 0, len(effects))
			for _, e := range effects {
				got = append(got, fmt.Sprintf("%v -%d %v", e.Characteristic, e.Reduction, e.Saving))
			}

			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Errorf("term %d: %v, want %v", term, got, tc.want)
			}
		}
	}

	// No round is run before the fourth term (p. 7).
	for term := traveller.Term(1); term < 4; term++ {
		if got := r.Aging().At(term); len(got) != 0 {
			t.Errorf("term %d: %v, want no aging", term, got)
		}
	}

	// Education and Social Standing are unaffected at every band.
	for term := traveller.Term(4); term <= 40; term++ {
		for _, e := range r.Aging().At(term) {
			if e.Characteristic == traveller.Education || e.Characteristic == traveller.SocialStanding {
				t.Errorf("term %d reduces %v, which the table says is unaffected by aging", term, e.Characteristic)
			}
		}
	}
}

// The medical crisis (pp. 7-8): "A basic saving throw of 8+ applies ... The
// characteristic which was reduced to zero automatically becomes one. The
// character ages (one die equals the number of months in added age)."
//
// Holds: Saving throw 8+, with no modifier during generation.
func TestMedicalCrisis(t *testing.T) {
	t.Parallel()

	crisis := load(t).Aging().Crisis

	if got := crisis.Saving.String(); got != "8+" {
		t.Errorf("crisis saving throw is %s, want 8+", got)
	}

	if crisis.RecoversTo != 1 {
		t.Errorf("crisis recovers to %d, want 1", crisis.RecoversTo)
	}

	if crisis.MonthsDice != 1 {
		t.Errorf("crisis adds %dD months, want 1D", crisis.MonthsDice)
	}
}

// The weapon lists (pp. 12-13), read as the boxes are set: the left column,
// then the right.
func TestWeaponLists(t *testing.T) {
	t.Parallel()

	r := load(t)

	for _, tc := range []struct {
		category    traveller.WeaponCategory
		left, right []string
	}{
		{
			category: traveller.Blade,
			left:     []string{"Dagger", "Blade", "Foil", "Cutlass", "Sword", "Broadsword"},
			right:    []string{"Spear", "Halberd", "Pike", "Cudgel", "Bayonet"},
		},
		{
			category: traveller.Gun,
			left:     []string{"Body Pistol", "Automatic Pistol", "Revolver", "Carbine", "Rifle"},
			right:    []string{"Laser Carbine", "Laser Rifle", "Automatic Rifle", "Submachine Gun", "Shotgun"},
		},
	} {
		printed := append(append([]string{}, tc.left...), tc.right...)

		want := make([]traveller.WeaponName, 0, len(printed))
		for _, name := range printed {
			want = append(want, traveller.WeaponName(name))
		}

		got := r.Weapons(tc.category)
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%v:\n got %v\nwant %v", tc.category, got, want)
		}
	}
}

// A caller cannot write through a weapon list into the ruleset.
//
// Load is once per process, so a write through the ruleset's own list changed
// what every later character was offered (#137).
func TestAWeaponListIsTheCallersOwn(t *testing.T) {
	t.Parallel()

	for _, category := range []traveller.WeaponCategory{traveller.Blade, traveller.Gun} {
		given := load(t).Weapons(category)
		if len(given) == 0 {
			t.Fatalf("%v: no printed list, so this checks nothing", category)
		}

		printed := given[0]

		given[0] = "Lightsaber"

		again := load(t).Weapons(category)
		if len(again) == 0 || again[0] != printed {
			t.Errorf("%v: a write through one list reached the ruleset: now %v, want %q first",
				category, again, printed)
		}
	}
}

// Nothing else the ruleset hands out writes through to it either: a service's
// ranks and its throws' DMs, and an aging band's effects, are each a slice,
// and each comes back as the caller's own (#137).
func TestNothingTheRulesHandOutIsShared(t *testing.T) {
	t.Parallel()

	navy := load(t).Service(traveller.Navy)
	commission, _ := navy.Commission()
	last := load(t).Aging().LastPrintedTerm()
	effects := load(t).Aging().At(last)

	if len(navy.Ranks) == 0 || len(navy.Enlistment.DMs) == 0 || len(commission.DMs) == 0 || len(effects) == 0 {
		t.Fatal("the Navy prints no ranks or DMs, or the last band no effects, so this checks nothing")
	}

	rank, enlistDM, commissionDM, effect := navy.Ranks[0], navy.Enlistment.DMs[0], commission.DMs[0], effects[0]

	navy.Ranks[0] = "Grand Admiral"
	navy.Enlistment.DMs[0].Amount = 99
	commission.DMs[0].Amount = 99
	effects[0].Reduction = 99

	again := load(t).Service(traveller.Navy)
	againCommission, _ := again.Commission()
	againEffects := load(t).Aging().At(last)

	if len(again.Ranks) == 0 || again.Ranks[0] != rank {
		t.Errorf("a write through a service's ranks reached the ruleset: %v", again.Ranks)
	}

	if len(again.Enlistment.DMs) == 0 || again.Enlistment.DMs[0] != enlistDM {
		t.Errorf("a write through an enlistment DM reached the ruleset: %v", again.Enlistment.DMs)
	}

	if len(againCommission.DMs) == 0 || againCommission.DMs[0] != commissionDM {
		t.Errorf("a write through a commission DM reached the ruleset: %v", againCommission.DMs)
	}

	if len(againEffects) == 0 || againEffects[0] != effect {
		t.Errorf("a write through an aging effect reached the ruleset: %v", againEffects)
	}
}

// The Rank and Service Skills box (p. 23). Rank 0 means the grant is by
// virtue of the service itself, which E005 reads as granted once on
// entering it.
func TestRankAndServiceSkills(t *testing.T) {
	t.Parallel()

	r := load(t)

	onEntering := map[traveller.ServiceName][]string{
		traveller.Navy:      nil,
		traveller.Marines:   {"Cutlass"},
		traveller.Army:      {"Rifle"},
		traveller.Scouts:    {"Pilot"},
		traveller.Merchants: nil,
		traveller.Other:     nil,
	}
	for service, want := range onEntering {
		grants := r.GrantsOnEntering(service)

		got := make([]string, 0, len(grants))
		for _, result := range grants {
			got = append(got, describe(t, result))
		}

		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%v on entering: %v, want %v", service, got, want)
		}
	}

	byRank := []struct {
		service traveller.ServiceName
		rank    traveller.Rank
		want    []string
	}{
		{traveller.Navy, 5, []string{"+1 Social Standing"}},
		{traveller.Navy, 6, []string{"+1 Social Standing"}},
		{traveller.Marines, 1, []string{"Revolver"}},
		{traveller.Army, 1, []string{"Submachine Gun"}},
		{traveller.Merchants, 4, []string{"Pilot"}},
		{traveller.Navy, 1, nil},
		{traveller.Merchants, 5, nil},
	}
	for _, tc := range byRank {
		var got []string

		for _, result := range r.GrantsAtRank(tc.service, tc.rank) {
			got = append(got, describe(t, result))
		}

		if fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("%v rank %d: %v, want %v", tc.service, tc.rank, got, tc.want)
		}
	}
}

// Annual Retirement Pay (p. 21), and who draws it.
//
// Holds: Retirement pay from term 5, not for Scouts or Other.
// Holds: The pension table says whether it prices a term at all.
func TestRetirementPay(t *testing.T) {
	t.Parallel()

	r := load(t)

	for terms, want := range map[int]traveller.Credits{
		5: 4000, 6: 6000, 7: 8000, 8: 10000,
		// "Service beyond 8 terms adds CR 2000 per additional term."
		9: 12000, 10: 14000, 12: 18000,
	} {
		got, tabled := r.Retirement().Pay(terms)
		if !tabled || got != want {
			t.Errorf("%d terms: %v (tabled %v), want %v", terms, got, tabled, want)
		}
	}

	// A term the table does not print. P. 21 begins its pension at five
	// terms and says nothing below that, so Pay says so rather than reading
	// a missing key and returning a real-looking zero (#52). The guard used
	// to live in the caller alone, where nothing held it.
	for _, terms := range []int{0, 1, 4} {
		if pay, tabled := r.Retirement().Pay(terms); tabled {
			t.Errorf("%d terms is priced at %v, and the table starts at five", terms, pay)
		}
	}

	// "Retirement pay is not available to characters serving in the Scout or
	// the Other service."
	for _, name := range traveller.ServiceNames {
		want := name != traveller.Scouts && name != traveller.Other
		if got := r.Service(name).PaysPension; got != want {
			t.Errorf("%v pays a pension: %v, want %v", name, got, want)
		}
	}
}

// The Basic Skill Eligibility box (p. 6) and the Education 8+ gate (p. 11).
//
// Holds: A commission or promotion confers one skill eligibility.
// Holds: Two eligibilities for the initial term, one thereafter.
// Holds: The fourth table needs Education 8+.
func TestEligibility(t *testing.T) {
	t.Parallel()

	r := load(t)

	want := rules.Eligibility{InitialTerm: 2, PerSubsequentTerm: 1, OnCommission: 1, OnPromotion: 1}
	if r.Eligibility() != want {
		t.Errorf("eligibility %+v, want %+v", r.Eligibility(), want)
	}

	if r.Education().Table != traveller.AdvancedEducationEight {
		t.Errorf("the education gate opens %v", r.Education().Table)
	}

	for education, open := range map[int]bool{6: false, 7: false, 8: true, 9: true, 15: true} {
		var p traveller.Profile

		p[traveller.Education] = education
		if got := r.Education().Open(p); got != open {
			t.Errorf("education %d opens the fourth table: %v, want %v", education, got, open)
		}
	}
}

// The mustering out notes (pp. 7, 9).
//
// Holds: One roll per term, plus rank extras.
// Holds: The +1 with gambling on Table 2.
func TestMusteringOutNotes(t *testing.T) {
	t.Parallel()

	r := load(t)

	// "One benefit roll is allowed for each term of service served ... rank
	// 1 or 2 is allowed one extra roll, rank 3 or higher is allowed two
	// extra rolls." The page's own example: an uncommissioned character who
	// has served 4 terms is eligible for 4 benefits.
	for _, tc := range []struct {
		terms int
		rank  traveller.Rank
		want  int
	}{
		{4, 0, 4},
		{4, 1, 5},
		{4, 2, 5},
		{4, 3, 6},
		{4, 6, 6},
		{1, 0, 1},
		// Jamison musters out at 5 terms and rank 5, entitled to two extra
		// rolls, "in addition to the 5 rolls (for 5 terms of service)".
		{5, 5, 7},
	} {
		got := r.Muster().Rolls(tc.terms, tc.rank)
		if got.Rolls() != tc.want {
			t.Errorf("%d terms at rank %d: %d rolls, want %d", tc.terms, tc.rank, got.Rolls(), tc.want)
		}

		// One per term, and whatever is over that the rank earned, so the
		// record's account of the rolls is the page's.
		if got.ForTerms != tc.terms || got.ForRank != tc.want-tc.terms || got.Rank != tc.rank {
			t.Errorf("%d terms at rank %d: %+v, want %d for the terms and %d for rank %d",
				tc.terms, tc.rank, got, tc.terms, tc.want-tc.terms, tc.rank)
		}
	}

	if r.Muster().Table1DMFromRank5or6 != 1 || r.Muster().Table2DMFromGambling != 1 {
		t.Errorf("the two optional DMs are %d and %d, want 1 and 1",
			r.Muster().Table1DMFromRank5or6, r.Muster().Table2DMFromGambling)
	}

	// The skill that earns the second of those, transcribed a second time
	// (#47). It is data because it names which skill, and it has to be
	// spelled as the skills transcription lifts it after E012's normalization - a
	// mismatch would not fail, it would silently stop the modifier applying.
	if r.Muster().Table2ModifierFrom != "Gambling" {
		t.Errorf("the table 2 modifier is earned by %q, want Gambling", r.Muster().Table2ModifierFrom)
	}
}

// The two starships a mustering out benefit delivers (Book 2 pp. 18-19).
// The Scout/Courier "using the type 100 hull"; the Free Trader "using the
// type 200 hull".
//
// Holds: The ships the two benefits name: Type S, Type A.
func TestShipHulls(t *testing.T) {
	t.Parallel()

	r := load(t)

	for kind, want := range map[traveller.ShipKind]int{
		traveller.ScoutShip:  100,
		traveller.FreeTrader: 200,
	} {
		if got := r.Hull(kind); got != want {
			t.Errorf("%v is built on a type %d hull, want %d", kind, got, want)
		}
	}
}

// The Nobility table (Book 3 p. 22).
//
// Holds: The five ranks of nobility.
func TestNobility(t *testing.T) {
	t.Parallel()

	r := load(t)

	for social, want := range map[int]traveller.Title{
		11: traveller.Knight, 12: traveller.Baron, 13: traveller.Marquis,
		14: traveller.Count, 15: traveller.Duke,
	} {
		got, ok := r.TitleFor(social)
		if !ok || got != want {
			t.Errorf("social standing %d: %v (%v), want %v", social, got, ok, want)
		}
	}

	for _, social := range []int{1, 7, 10} {
		if _, ok := r.TitleFor(social); ok {
			t.Errorf("social standing %d confers a title; nobility begins at 11", social)
		}
	}
}

// P. 10's own worked example of a cumulative DM: "the enlistment throw
// required for the Navy is 8+; DM of +1 allowed for intelligence of 8 or
// greater, and DM of +2 is allowed for education of 9 or greater. Assuming a
// character has intelligence of 6 and education of 10 ... he would be
// allowed a DM of +2 (for his education)." (p. 5).
//
// Holds: Cumulative characteristic DMs.
func TestModifierIsCumulative(t *testing.T) {
	t.Parallel()

	r := load(t)
	navy := r.Service(traveller.Navy).Enlistment

	profile := func(intelligence, education int) traveller.Profile {
		var p traveller.Profile

		p[traveller.Intelligence] = intelligence
		p[traveller.Education] = education

		return p
	}

	for _, tc := range []struct {
		name                    string
		intelligence, education int
		want                    int
	}{
		{"the page's own example", 6, 10, 2},
		{"neither threshold", 6, 8, 0},
		{"intelligence only", 8, 8, 1},
		{"both, cumulative", 8, 9, 3},
	} {
		if got := navy.Modifier(profile(tc.intelligence, tc.education)); got != tc.want {
			t.Errorf("%s: DM %+d, want %+d", tc.name, got, tc.want)
		}
	}

	// Jamison enlists in the Merchants: 7+, "DM of +2 allowed for his
	// intelligence of greater than 6", his strength of 6 earning nothing
	// (p. 24).
	var jamison traveller.Profile

	copy(jamison[:], []int{6, 8, 8, 12, 8, 9})

	if got := r.Service(traveller.Merchants).Enlistment.Modifier(jamison); got != 2 {
		t.Errorf("Jamison's enlistment DM is %+d, want +2", got)
	}
}

// The draft is one die, and each service prints its own number (p. 5).
//
// Holds: One die, entering the service with that draft number.
func TestDraft(t *testing.T) {
	t.Parallel()

	r := load(t)

	for die, want := range map[int]traveller.ServiceName{
		1: traveller.Navy, 2: traveller.Marines, 3: traveller.Army,
		4: traveller.Scouts, 5: traveller.Merchants, 6: traveller.Other,
	} {
		got, err := r.Draft(die)
		if err != nil || got != want {
			t.Errorf("draft %d: %v (%v), want %v", die, got, err, want)
		}
	}

	for _, die := range []int{0, 7} {
		_, err := r.Draft(die)
		if err == nil {
			t.Errorf("draft %d named a service; the draft is one die", die)
		}
	}
}

// E012's normalizations, applied by the lift.
//
// Holds: Names normalized to their description headings.
func TestNormalization(t *testing.T) {
	t.Parallel()

	r := load(t)

	for printed, want := range map[string]traveller.SkillName{
		"Fwd Obsv":    "Forward Observer",
		"Engnrng":     "Engineer",
		"Jack-o-T":    "Jack of all Trades",
		"Admin":       "Administration",
		"Electronics": "Electronic",
		"Rifl3":       "Rifle",
		"SMG":         "Submachine Gun",
		// A name already spelled as its description heading passes through.
		"Navigation": "Navigation",
		"Gunnery":    "Gunnery",
	} {
		if got := r.Normalize(printed); got != want {
			t.Errorf("%q normalizes to %q, want %q", printed, got, want)
		}
	}
}

// A roll off the end of a table is a defect, not a missing value.
func TestRollsOffTheEndAreRefused(t *testing.T) {
	t.Parallel()

	navy := load(t).Service(traveller.Navy)

	for _, die := range []int{0, 7} {
		_, err := navy.Result(traveller.ServiceSkills, die)
		if err == nil {
			t.Errorf("skills table accepted a roll of %d", die)
		}
	}

	for _, row := range []int{0, 8} {
		_, _, err := navy.Row(row)
		if err == nil {
			t.Errorf("mustering out accepted row %d", row)
		}
	}
}
