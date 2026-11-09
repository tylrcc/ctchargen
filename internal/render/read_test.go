package render_test

import (
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/render"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// noteOf is the note of the least character the sheet reads, with change
// applied, so a test can pin one detail without carrying a whole character.
func noteOf(change func(c *chargen.Character)) string {
	character := &chargen.Character{
		Age:        traveller.NewAge(18),
		Enlistment: traveller.Enlisted{Service: traveller.Navy},
	}

	change(character)

	return render.Note(character, false)
}

// headlineOf is the line the sheet leads with, the one every headline case
// is about. Searching the whole note is too loose: "(" would find the
// "(unnamed)" title.
func headlineOf(note string) string {
	for line := range strings.Lines(note) {
		if strings.HasPrefix(line, "UPP ") {
			return line
		}
	}

	return ""
}

// A ship's terms are read off its kind, not inferred from its numbers. A
// Free Trader received and never paid down carries years 0 and paymentYears
// 0, and is owned - not held in constructive possession, which is the scout
// ship's arrangement (p. 23).
//
// Holds: A ship's terms follow its kind, not its numbers.
func TestAShipsTermsFollowItsKind(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		kind        traveller.ShipKind
		want, avoid string
	}{
		"a new Free Trader": {traveller.FreeTrader, "owned free and clear", "constructive possession"},
		"a scout ship":      {traveller.ScoutShip, "held in constructive possession", "owned free and clear"},
	} {
		note := noteOf(func(c *chargen.Character) {
			c.Benefits.Ships = []chargen.Ship{{Kind: tc.kind, Tons: 200}}
		})

		if !strings.Contains(note, tc.want) {
			t.Errorf("%s does not read %q:\n%s", name, tc.want, note)
		}

		if strings.Contains(note, tc.avoid) {
			t.Errorf("%s reads %q, which is the other ship's terms:\n%s", name, tc.avoid, note)
		}
	}
}

// The service record narrates the career a term at a time, as the Jamison
// example does (pp. 24-25): the rank each term conferred and what it granted,
// what the term's skill rolls gave in the order rolled, what aging took, and a
// crisis survived. The grants on entering the service sit on the enlistment
// line, where E005 puts them. A rank's grants are in parentheses and two ranks
// in one term are joined by "and", so "; " only separates a term's parts. A
// term with nothing to tell - the one a survival throw ended - has no line,
// since the departure line says how it ended (#401).
//
// Holds: The service record narrates the career a term at a time.
func TestTheServiceRecordNarratesTheServiceHistory(t *testing.T) {
	t.Parallel()

	skill := func(name string, level int) chargen.Gain {
		return chargen.SkillGain{Skill: traveller.Skill{Name: traveller.SkillName(name), Level: level}}
	}

	note := noteOf(func(c *chargen.Character) {
		c.Terms = 3
		c.Enlistment = traveller.Enlisted{Service: traveller.Army}
		c.Departure = traveller.KilledBySurvivalThrow{}
		c.History = chargen.ServiceHistory{
			OnEntering: []chargen.Gain{skill("Rifle", 1)},
			Terms: []chargen.TermServed{
				{
					Term: 1,
					Raised: []chargen.Raise{
						{How: chargen.Commissioned, Rank: "Lieutenant", Grants: []chargen.Gain{skill("Submachine Gun", 1)}},
						{How: chargen.Promoted, Rank: captain},
					},
					Gained: []chargen.Gain{skill("Tactics", 1), skill("Halberd", 1)},
				},
				{
					Term: 2,
					Raised: []chargen.Raise{{
						How: chargen.Promoted, Rank: "Major",
						Grants: []chargen.Gain{chargen.Change{Characteristic: traveller.SocialStanding, From: 9, To: 10}},
					}},
					Gained: []chargen.Gain{
						chargen.Change{Characteristic: traveller.Education, From: 10, To: 11}, skill("Tactics", 2),
					},
					Aged: []chargen.Change{
						{Characteristic: traveller.Strength, From: 1, To: 0},
						{Characteristic: traveller.Endurance, From: 8, To: 7},
					},
					Recovered: []traveller.Characteristic{traveller.Strength},
				},
				{Term: 3},
			},
		}
	})

	want := "## Service record\n\n" +
		"- Army, enlisted; Rifle-1\n" +
		"- Term 1: commissioned Lieutenant (Submachine Gun-1) and promoted Captain; Tactics-1, Halberd-1\n" +
		"- Term 2: promoted Major (Social Standing +1); Education +1, Tactics-2; " +
		"aging Strength -1, Endurance -1; Strength recovered from a medical crisis\n" +
		"- killed by the survival throw in term 3\n"
	if !strings.Contains(note, want) {
		t.Errorf("the service record does not read:\n%s\nin:\n%s", want, note)
	}
}

// A choice is named on the generation record in the words of the page that
// poses it, with that page, and never by the Decider method's name (#402).
// The labels are typed out here from the pages, not read from the code that
// writes them.
//
// Holds: A choice is named in the words of the page that poses it.
func TestAChoiceIsNamedInThePagesWords(t *testing.T) {
	t.Parallel()

	for point, want := range map[traveller.ChoicePoint]string{
		traveller.ChoiceService:           "service to enlist in (pp. 5, 10)",
		traveller.ChoiceSubmitToDraft:     "submit to the draft (p. 5)",
		traveller.ChoiceAttemptCommission: "attempt a commission (p. 6)",
		traveller.ChoiceAttemptPromotion:  "attempt promotion (p. 6)",
		traveller.ChoiceReenlistIntent:    "leave the service or remain for another term (pp. 6-7, 21)",
		traveller.ChoiceAssumeTitle:       "assume the hereditary title (p. 5)",
		traveller.ChoiceSkillTable:        "acquired skills table to consult (pp. 6, 11)",
		traveller.ChoiceWeapon:            "weapon of expertise, specified immediately (p. 11)",
		traveller.ChoiceMusterTable:       "mustering out table to consult (pp. 7, 9)",
		traveller.ChoiceMusterTable1DM:    "add +1 on Table 1 for rank 5 or 6 (pp. 7, 9)",
		traveller.ChoiceMusterTable2DM:    "DM +1 on Table 2 for gambling expertise (pp. 7, 9)",
		traveller.ChoiceMusterWeapon:      "weapon benefit, or +1 expertise in lieu (p. 22)",
	} {
		policy := chargen.DefaultPolicy()

		note := noteOf(func(c *chargen.Character) {
			c.Policy = &policy
			c.Events = []traveller.Event{traveller.ChoiceEvent{
				Seq: 4, Point: point, Alternatives: []string{"a", "b"}, Chosen: "a",
			}}
		})

		line := "  4. " + want + ": policy chose a from a, b"
		if !strings.Contains(note, line) {
			t.Errorf("%v is not named %q:\n%s", point, line, note)
		}

		if strings.Contains(note, point.String()+":") {
			t.Errorf("%v is named by its method:\n%s", point, note)
		}
	}
}

// The headline says whose service it is, where that is not obvious.
//
// The service record further down has always been honest. The headline is
// the line a referee reads, and a character who was typed as a Marine and
// came back a Navy man had nothing there to explain it.
//
// Everything turns on the service ATTEMPTED, not on who named it. The draft
// can override either decider, and crediting one with an outcome it did not
// pick is worse than saying nothing.
//
// Holds: The headline names the service attempted, not the one the draft gave.
func TestTheHeadlineSaysWhoseServiceItIs(t *testing.T) {
	t.Parallel()

	for name, tc := range headlineCases() {
		note := noteOf(func(character *chargen.Character) {
			character.Enlistment = tc.served

			if tc.forced {
				character.Inputs.Forced = chargen.ForceService(tc.chose)
			} else {
				character.Events = []traveller.Event{traveller.ChoiceEvent{
					Seq: 15, Point: traveller.ChoiceService, Alternatives: []string{tc.chose.String()},
					Chosen: tc.chose.String(),
				}}
			}

			if tc.byPolicy {
				policy := chargen.DefaultPolicy()

				character.Policy = &policy
			}
		})

		lead := headlineOf(note)
		if !strings.Contains(lead, tc.wants) {
			t.Errorf("%s: the headline does not read %q: %q", name, tc.wants, lead)
		}

		if tc.does != "" && strings.Contains(lead, tc.does) {
			t.Errorf("%s: the headline reads %q and should not: %q", name, tc.does, lead)
		}
	}
}

// headlineCase is a service attempted, how it was named, and the service
// served: what the headline should read, and what it should not.
type headlineCase struct {
	served      traveller.Enlistment
	forced      bool
	chose       traveller.ServiceName
	byPolicy    bool
	wants, does string
}

func headlineCases() map[string]headlineCase {
	const unchanged = "Navy, "

	return map[string]headlineCase{
		"asked for one service and got another": {
			served: traveller.Drafted{Service: traveller.Navy}, forced: true, chose: traveller.Marines,
			wants: "Navy (drafted after the Marines refused him)",
		},
		"asked for a service and got it": {
			served: traveller.Enlisted{Service: traveller.Navy}, forced: true, chose: traveller.Navy,
			wants: unchanged, does: "refused",
		},
		"asked for nothing and the policy picked": {
			served: traveller.Enlisted{Service: traveller.Navy}, chose: traveller.Navy, byPolicy: true,
			wants: "Navy (service chosen by the policy)",
		},
		// A player who named it himself is told nothing he does not know.
		"asked for nothing and the player picked": {
			served: traveller.Enlisted{Service: traveller.Navy}, chose: traveller.Navy,
			wants: unchanged, does: "chosen by",
		},
		// The draft overrides the attempt. Saying "chosen by the policy"
		// over a draft's outcome credits the policy with a choice it never
		// made, in the case this exists to expose.
		"the policy picked one and the draft gave another": {
			served: traveller.Drafted{Service: traveller.Marines}, chose: traveller.Navy, byPolicy: true,
			wants: "Marines (drafted after the Navy refused him)", does: "chosen by the policy",
		},
		// And after a draft the player does not know either: he chose Navy.
		"the player picked one and the draft gave another": {
			served: traveller.Drafted{Service: traveller.Marines}, chose: traveller.Navy,
			wants: "Marines (drafted after the Navy refused him)",
		},
	}
}

// One term is one term.
//
// "1 term" is a substring of "1 terms", so the singular case is pinned by what
// must NOT appear. Asserting only the positive would pass on the bug.
//
// Holds: A one-term character reads as one term.
func TestOneTermIsSingular(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		terms       int
		wants, does string
	}{
		"one":  {1, "1 term", "1 terms"},
		"two":  {2, "2 terms", ""},
		"none": {0, "0 terms", ""},
	} {
		lead := headlineOf(noteOf(func(c *chargen.Character) { c.Terms = tc.terms }))

		if !strings.Contains(lead, tc.wants) {
			t.Errorf("%s: the headline does not read %q: %q", name, tc.wants, lead)
		}

		if tc.does != "" && strings.Contains(lead, tc.does) {
			t.Errorf("%s: the headline reads %q: %q", name, tc.does, lead)
		}
	}
}

// The errata are on the generation record and not on the sheet.
//
// On the sheet a player reads they are codes he cannot expand; beside the
// outcome they governed they are how a reading is checked against the page,
// which is what the generation record is for.
//
// Holds: The errata are on the generation record and not the sheet.
func TestTheErrataAreOnTheGenerationRecordAndNotTheSheet(t *testing.T) {
	t.Parallel()

	note := noteOf(func(c *chargen.Character) {
		c.Errata = []traveller.Erratum{traveller.E001}
		c.Events = []traveller.Event{traveller.OutcomeEvent{
			Seq: 1, Description: "drafted", Errata: []traveller.Erratum{traveller.E001},
		}}
	})

	sheet, record, found := strings.Cut(note, "## Generation record")
	if !found {
		t.Fatalf("the note has no generation record:\n%s", note)
	}

	if strings.Contains(sheet, "E001") {
		t.Errorf("the sheet carries an erratum id:\n%s", sheet)
	}

	if !strings.Contains(record, "E001") {
		t.Errorf("the generation record lost the erratum that governed an outcome:\n%s", record)
	}
}

// A note ends by saying what generated it and what to keep: the ruleset its
// page cites are to, the build that wrote it where there is one, and who
// answered its choices - the policy with its strategies, or the player - and
// then that the generation record is the only way back (ADR 0001, ADR 0002).
//
// Holds: A note names the ruleset, the build and who answered it.
// Holds: A note names the policy that answered it.
func TestANoteEndsWithWhatWroteItAndWhatToKeep(t *testing.T) {
	t.Parallel()

	const (
		ruleset = "Generated under Classic Traveller, Books 1-3 (FFE reprint of the 1977 text)"
		keep    = "\n\nKeep this note whole: its generation record is the only way back to this character.\n"
	)

	policy := chargen.DefaultPolicy()

	for name, tc := range map[string]struct {
		change func(c *chargen.Character)
		ends   string
	}{
		"a guided run": {
			func(*chargen.Character) {},
			ruleset + "; every choice answered by the player." + keep,
		},
		"an auto run, built": {
			func(c *chargen.Character) { c.Policy, c.Build = &policy, "v1.2.3" },
			ruleset + " by ctchargen v1.2.3; every choice answered by the policy, version 3: " +
				"career serve, skills advanced, muster cash." + keep,
		},
	} {
		note := noteOf(tc.change)
		if !strings.HasSuffix(note, "---\n\n"+tc.ends) {
			t.Errorf("%s: the note does not end with\n%s\nin:\n%s", name, tc.ends, note)
		}
	}
}

// A roll has nothing to meet, so its line reads the dice and the row they
// land on and claims no success: no target, no "made" (#50). A throw beside it
// reads its target and its outcome.
//
// Holds: A roll with nothing to meet is not recorded as a throw that
// succeeded.
func TestARollClaimsNothingMet(t *testing.T) {
	t.Parallel()

	fivePlus, err := traveller.ParseTarget("5+")
	if err != nil {
		t.Fatal(err)
	}

	note := noteOf(func(c *chargen.Character) {
		c.Events = []traveller.Event{
			traveller.RollEvent{Seq: 1, Step: "Table 2, Cash Allowances", Dice: []int{4}, DM: 1},
			traveller.ThrowEvent{Seq: 2, Step: "survival", Dice: []int{3, 4}, Target: fivePlus, Succeeded: true},
		}
	})

	for _, want := range []string{
		"  1. Table 2, Cash Allowances: rolled 4 +1 = 5\n",
		"  2. survival: rolled 3+4 = 7 against 5+, made\n",
	} {
		if !strings.Contains(note, want) {
			t.Errorf("the note does not read %q:\n%s", want, note)
		}
	}
}

// The generation record sits under one heading, its steps as bold lines
// rather than headings of their own, so a note's outline shows the sheet's
// sections and one entry for the record (#422). Each step keeps its number
// (#249).
//
// Holds: The generation record is one heading, its steps bold lines.
func TestTheGenerationRecordIsOneHeading(t *testing.T) {
	t.Parallel()

	note := noteOf(func(c *chargen.Character) {
		c.Events = []traveller.Event{
			traveller.StepEvent{Seq: 1, Step: "characteristics", Pages: "p. 4"},
			traveller.OutcomeEvent{Seq: 2, Description: "UPP 777777 at age 18"},
			traveller.StepEvent{Seq: 3, Step: "enlistment", Pages: "pp. 5, 10"},
		}
	})

	_, record, _ := strings.Cut(note, "## Generation record\n")

	for line := range strings.Lines(record) {
		if strings.HasPrefix(line, "#") {
			t.Errorf("the generation record carries a heading of its own: %q", line)
		}
	}

	for _, want := range []string{
		"\n**1. characteristics (p. 4)**\n\n",
		"  2. UPP 777777 at age 18\n",
		"\n**3. enlistment (pp. 5, 10)**\n\n",
	} {
		if !strings.Contains(record, want) {
			t.Errorf("the generation record does not read %q:\n%s", want, record)
		}
	}
}

// captain is the rank the tests below confer, by the Army's and the Navy's
// name for it.
const captain = "Captain"

// pilot is the one skill the tests below give a Scout or a merchant.
const pilot = "Pilot"

// profile is a Profile from six values, in the order p. 4 rolls them.
func profile(values ...int) traveller.Profile {
	var p traveller.Profile

	copy(p[:], values)

	return p
}

// A note opens with properties a vault can query (#399): flat keys in the
// sheet's own words, single values because a character serves one service
// (Book 1 p. 5), `status` saying outright whether he lived (#418), skills in
// the book's notation, and the name as an alias when one was given. A key
// with nothing to say is absent rather than blank, but for name. The UPP is
// quoted, since six digits would otherwise read as a number.
//
// Holds: A note opens with properties a vault can query.
func TestANoteOpensWithItsProperties(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]noteCase{
		"a named officer who took his title": {
			func(c *chargen.Character) {
				c.Inputs.Name = "Kira Vance"
				c.Profile = profile(7, 9, 3, 7, 4, 11)
				c.Age = traveller.NewAge(26)
				c.Terms = 2
				c.Enlistment = traveller.Enlisted{Service: traveller.Army}
				c.RankName = captain
				c.Departure = traveller.ForcedOut{}
				c.Skills = []traveller.Skill{{Name: "Brawling", Level: 1}, {Name: "Rifle", Level: 2}}
				c.Benefits.Cash = 30000
				c.Nobility = chargen.Nobility{Eligible: true, Title: traveller.Knight, Assumed: true}
			},
			"---\nname: \"Kira Vance\"\naliases:\n  - \"Kira Vance\"\nupp: \"79374B\"\n" +
				"strength: 7\ndexterity: 9\nendurance: 3\nintelligence: 7\neducation: 4\nsocial: 11\n" +
				"age: 26\nservice: \"Army\"\nrank: \"Captain\"\nterms: 2\nstatus: \"alive\"\ndeparture: \"forced out\"\n" +
				"skills:\n  - \"Brawling-1\"\n  - \"Rifle-2\"\ncredits: 30000\npension: 0\nnobility: \"knight/dame\"\n---\n",
		},
		"an unnamed civilian": {
			func(c *chargen.Character) {
				c.Profile = profile(7, 7, 7, 7, 7, 7)
				c.Enlistment = traveller.DeclinedTheDraft{}
			},
			"---\nname:\nupp: \"777777\"\n" +
				"strength: 7\ndexterity: 7\nendurance: 7\nintelligence: 7\neducation: 7\nsocial: 7\n" +
				"age: 18\nterms: 0\nstatus: \"alive\"\nskills: []\ncredits: 0\npension: 0\n---\n",
		},
		// What a referee types is escaped, not trusted: quotes, a colon, a
		// hash, a backslash and a letter outside ASCII all read back as typed.
		"a name a referee typed": {
			func(c *chargen.Character) {
				c.Inputs.Name = `O'Brien: "Red" #1 \ Mår`
				c.Profile = profile(7, 7, 7, 7, 7, 7)
				c.Enlistment = traveller.DeclinedTheDraft{}
			},
			"---\nname: \"O'Brien: \\\"Red\\\" #1 \\\\ Mår\"\n" +
				"aliases:\n  - \"O'Brien: \\\"Red\\\" #1 \\\\ Mår\"\n",
		},
		"a dead Scout": {
			func(c *chargen.Character) {
				c.Profile = profile(6, 6, 6, 6, 6, 6)
				c.Age = traveller.NewAge(30)
				c.Terms = 3
				c.Enlistment = traveller.Enlisted{Service: traveller.Scouts}
				c.Departure = traveller.KilledBySurvivalThrow{}
				c.Skills = []traveller.Skill{{Name: pilot, Level: 1}}
			},
			"---\nname:\nupp: \"666666\"\n" +
				"strength: 6\ndexterity: 6\nendurance: 6\nintelligence: 6\neducation: 6\nsocial: 6\n" +
				"age: 30\nservice: \"Scouts\"\nterms: 3\nstatus: \"dead\"\ndeparture: \"killed by the survival throw\"\n" +
				"skills:\n  - \"Pilot-1\"\ncredits: 0\npension: 0\n---\n",
		},
	} {
		note := noteOf(tc.change)
		if !strings.HasPrefix(note, tc.want) {
			got, _, _ := strings.Cut(note, "\n# ")
			t.Errorf("%s: the note opens\n%s\nwant\n%s", name, got, tc.want)
		}
	}
}

// A note's heading is followed by a paragraph a referee can read aloud
// (#425): the service and rank, terms, how he left and at what age, his two
// or three best skills, and what he mustered out with that matters - credits,
// a ship, Travellers' Aid. The dead are told how and when they died (#418);
// a civilian, that he declined the draft. The words are typed out here, not
// read from the code that writes them.
//
// Holds: A note opens with a summary a referee can read aloud.
func TestANoteOpensWithASummary(t *testing.T) {
	t.Parallel()

	for name, tc := range summaryCases() {
		note := noteOf(tc.change)

		heading := "\n# (unnamed)\n\n"
		if !strings.Contains(note, heading+tc.want+"\n\nUPP ") {
			_, after, _ := strings.Cut(note, heading)
			got, _, _ := strings.Cut(after, "\nUPP ")
			t.Errorf("%s: the summary reads %q, want %q", name, got, tc.want)
		}
	}
}

// noteCase is a character, as a change to the least one, and what its note
// should read.
type noteCase struct {
	change func(c *chargen.Character)
	want   string
}

// summaryCases are an officer, a merchant with a ship, a civilian and a dead
// Scout, each with the summary its note should open with.
func summaryCases() map[string]noteCase {
	return map[string]noteCase{
		"an officer": {
			func(c *chargen.Character) {
				c.Age = traveller.NewAge(26)
				c.Terms = 2
				c.Enlistment = traveller.Enlisted{Service: traveller.Army}
				c.RankName = captain
				c.Departure = traveller.ForcedOut{}
				c.Skills = []traveller.Skill{
					{Name: "Brawling", Level: 1},
					{Name: "Medical", Level: 1},
					{Name: "Rifle", Level: 2},
					{Name: "Tactics", Level: 1},
				}
				c.Benefits.Cash = 30000
			},
			"Army Captain, 2 terms, forced out at 26. Known for Rifle-2. " +
				"Musters out with CR 30000.",
		},
		"a merchant with a ship": {
			func(c *chargen.Character) {
				c.Age = traveller.NewAge(46)
				c.Terms = 7
				c.Enlistment = traveller.Enlisted{Service: traveller.Merchants}
				c.Departure = traveller.Retired{}
				c.Skills = []traveller.Skill{{Name: pilot, Level: 1}}
				c.Benefits.TravellersAid = true
				c.Benefits.Ships = []chargen.Ship{{Kind: traveller.FreeTrader, Tons: 200}}
			},
			"Merchants, 7 terms, retired at 46. " +
				"Musters out with Travellers' Aid Society membership and a Free Trader, Type A.",
		},
		"a civilian": {
			func(c *chargen.Character) { c.Enlistment = traveller.DeclinedTheDraft{} },
			"A civilian who declined the draft at 18.",
		},
		"a Scout with his ship": {
			func(c *chargen.Character) {
				c.Age = traveller.NewAge(46)
				c.Terms = 7
				c.Enlistment = traveller.Enlisted{Service: traveller.Scouts}
				c.Departure = traveller.Retired{}
				c.Skills = []traveller.Skill{{Name: pilot, Level: 2}, {Name: "Electronic", Level: 3}}
				c.Benefits.Ships = []chargen.Ship{{Kind: traveller.ScoutShip, Tons: 100}}
			},
			"Scouts, 7 terms, retired at 46. Known for Electronic-3 and Pilot-2. " +
				"Musters out with a Scout ship, Type S, held in constructive possession.",
		},
		"a dead Scout": {
			func(c *chargen.Character) {
				c.Age = traveller.NewAge(30)
				c.Terms = 3
				c.Enlistment = traveller.Enlisted{Service: traveller.Scouts}
				c.Departure = traveller.KilledBySurvivalThrow{}
				c.Skills = []traveller.Skill{{Name: pilot, Level: 1}}
			},
			"Scouts, killed by the survival throw in term 3, aged 30.",
		},
	}
}
