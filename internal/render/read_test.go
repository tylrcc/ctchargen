package render_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/render"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// noteOf is the note of the least character the sheet reads, with change
// applied, so a test can pin one detail without carrying a whole character.
// He served, so he has a departure, as generation gives every one who did
// (#445). A change that sets the record sets the career with it, since the
// career is read from the record (#441).
func noteOf(change func(c *chargen.Character)) string {
	character := &chargen.Character{Events: recorded(aDischargedSailor())}

	change(character)

	return render.Note(character)
}

// headlineOf is the line the sheet leads with, the one every headline case
// is about. Searching the whole note is too loose: "(" would find a
// parenthesis anywhere in the note.
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
			c.Events = recorded(aDischargedSailor(),
				traveller.ShipReceivedFact{Ship: traveller.Ship{Kind: tc.kind, Tons: 200}})
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

	skill := func(name string, level int) traveller.Gain {
		return traveller.SkillGain{Skill: traveller.Skill{Name: traveller.SkillName(name), Level: level}}
	}

	note := noteOf(func(c *chargen.Character) {
		c.Events = recorded(
			enlisted(traveller.Army),
			applied(skill("Rifle", 1), traveller.OnEntering),
			traveller.Term(1),
			traveller.RaisedFact{How: traveller.Commissioned, Rank: 1, Name: "Lieutenant"},
			applied(skill("Submachine Gun", 1), traveller.AtRank),
			traveller.RaisedFact{How: traveller.Promoted, Rank: 2, Name: captain},
			applied(skill("Tactics", 1), traveller.InTerm),
			applied(skill("Halberd", 1), traveller.InTerm),
			traveller.Term(2),
			traveller.RaisedFact{How: traveller.Promoted, Rank: 3, Name: "Major"},
			applied(traveller.Change{Characteristic: traveller.SocialStanding, From: 9, To: 10, Delta: 1}, traveller.AtRank),
			applied(traveller.Change{Characteristic: traveller.Education, From: 10, To: 11, Delta: 1}, traveller.InTerm),
			applied(skill("Tactics", 2), traveller.InTerm),
			traveller.AgedFact{Change: traveller.Change{Characteristic: traveller.Strength, From: 1, To: 0, Delta: -1}},
			traveller.AgedFact{Change: traveller.Change{Characteristic: traveller.Endurance, From: 8, To: 7, Delta: -1}},
			traveller.RecoveredFact{Characteristic: traveller.Strength, To: 1, Age: traveller.NewAge(26).PlusMonths(3)},
			traveller.Term(3),
			traveller.DiedFact{Term: 3, Age: traveller.NewAge(30).PlusMonths(3)},
		)
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

// A characteristic a mustering-out benefit raised is on the sheet's service
// record, as every other gain is, after the line that says how the service
// ended, since mustering out comes after it (#490). It was in the record and
// the UPP, and nowhere the sheet said where it came from.
//
// Holds: A characteristic raised at mustering out is on the service record.
func TestAMusteringOutGainIsOnTheServiceRecord(t *testing.T) {
	t.Parallel()

	note := noteOf(func(c *chargen.Character) {
		c.Events = recorded(
			enlisted(traveller.Navy),
			traveller.Term(1),
			traveller.DischargedFact{Age: traveller.NewAge(22)},
			applied(traveller.Change{Characteristic: traveller.Education, From: 9, To: 10, Delta: 1},
				traveller.AtMusteringOut),
			applied(traveller.Change{Characteristic: traveller.Intelligence, From: 7, To: 8, Delta: 1},
				traveller.AtMusteringOut),
		)
	})

	want := "- mustered out with Education +1, Intelligence +1\n"
	if !strings.Contains(note, want) {
		t.Errorf("the service record does not read %q:\n%s", want, note)
	}
}

// A throw's DM names the characteristics it is earned by, with the thresholds
// p. 10 prints for them, so a referee can check it against the page (#487);
// and an outcome names the step it came from as a step, since "(from 2)" read
// as a characteristic that had been 2.
//
// Holds: A throw's DM names what earned it.
func TestAThrowsDMNamesWhatEarnedIt(t *testing.T) {
	t.Parallel()

	target := func(printed string) traveller.Target {
		parsed, err := traveller.ParseTarget(printed)
		if err != nil {
			t.Fatal(err)
		}

		return parsed
	}

	note := noteOf(func(c *chargen.Character) {
		c.Events = append(recorded(aDischargedSailor()...),
			traveller.ThrowEvent{
				Seq: 3, Step: "enlistment, Army", Dice: []int{3, 4}, DM: 3,
				Target: target("5+"), Succeeded: true,
				DMFrom: []traveller.Prerequisite{
					{Characteristic: traveller.Dexterity, Threshold: target("6+")},
					{Characteristic: traveller.Endurance, Threshold: target("5+")},
				},
			},
			traveller.OutcomeEvent{Seq: 4, Because: 3, Description: "enlisted in the Army"},
		)
	})

	for _, want := range []string{
		"enlistment, Army: rolled 3+4 +3 (Dexterity 6+, Endurance 5+) = 10 against 5+, made",
		"enlisted in the Army (from step 3)",
	} {
		if !strings.Contains(note, want) {
			t.Errorf("the record does not read %q:\n%s", want, note)
		}
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
			c.Events = recorded(traveller.ChoiceEvent{
				Seq: 4, Point: point, Alternatives: []string{"a", "b"}, Chosen: "a",
			}, aDischargedSailor())
		})

		line := "\n4. " + want + ": policy chose a from a, b"
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
			// The enlistment throw's outcome is where the service attempted is
			// read from, whoever named it (#397); the draft's, where he served.
			character.Events = recorded(tc.entered, traveller.DischargedFact{Age: traveller.NewAge(18)})

			if tc.forced {
				character.Inputs.Forced = chargen.ForceService(tc.chose)
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

// headlineCase is a service attempted, how it was named, and the outcomes
// that entered him in a service: what the headline should read, and what it
// should not.
type headlineCase struct {
	entered     []traveller.Fact
	forced      bool
	chose       traveller.ServiceName
	byPolicy    bool
	wants, does string
}

func headlineCases() map[string]headlineCase {
	const unchanged = "Navy, "

	return map[string]headlineCase{
		"asked for one service and got another": {
			entered: draftedAfter(traveller.Marines, traveller.Navy), forced: true, chose: traveller.Marines,
			wants: "Navy (drafted after the Marines refused enlistment)",
		},
		"asked for a service and got it": {
			entered: []traveller.Fact{enlisted(traveller.Navy)}, forced: true, chose: traveller.Navy,
			wants: unchanged, does: "refused",
		},
		"asked for nothing and the policy picked": {
			entered: []traveller.Fact{enlisted(traveller.Navy)}, chose: traveller.Navy, byPolicy: true,
			wants: "Navy (service chosen by the policy)",
		},
		// A player who named it himself is told nothing he does not know.
		"asked for nothing and the player picked": {
			entered: []traveller.Fact{enlisted(traveller.Navy)}, chose: traveller.Navy,
			wants: unchanged, does: "chosen by",
		},
		// The draft overrides the attempt. Saying "chosen by the policy"
		// over a draft's outcome credits the policy with a choice it never
		// made, in the case this exists to expose.
		"the policy picked one and the draft gave another": {
			entered: draftedAfter(traveller.Navy, traveller.Marines), chose: traveller.Navy, byPolicy: true,
			wants: "Marines (drafted after the Navy refused enlistment)", does: "chosen by the policy",
		},
		// And after a draft the player does not know either: he chose Navy.
		"the player picked one and the draft gave another": {
			entered: draftedAfter(traveller.Navy, traveller.Marines), chose: traveller.Navy,
			wants: "Marines (drafted after the Navy refused enlistment)",
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
		lead := headlineOf(noteOf(func(c *chargen.Character) {
			c.Events = recorded(enlisted(traveller.Navy), opened(tc.terms),
				traveller.DischargedFact{After: traveller.Term(tc.terms), Age: traveller.NewAge(18)})
		}))

		if !strings.Contains(lead, tc.wants) {
			t.Errorf("%s: the headline does not read %q: %q", name, tc.wants, lead)
		}

		if tc.does != "" && strings.Contains(lead, tc.does) {
			t.Errorf("%s: the headline reads %q: %q", name, tc.does, lead)
		}
	}
}

// A note names no erratum, on the sheet or in the generation record, even
// for a character every reading governed. The readings are the authority's
// record, not the referee's (ADR 0007): an id beside an outcome was a code the
// playtest's referee was never told what to do with (#460, #488).
//
// Holds: A note names no erratum.
func TestANoteNamesNoErratum(t *testing.T) {
	t.Parallel()

	note := noteOf(func(c *chargen.Character) { c.Errata = traveller.Errata[:] })

	for _, erratum := range traveller.Errata {
		if strings.Contains(note, erratum.String()) {
			t.Errorf("the note names %v:\n%s", erratum, note)
		}
	}
}

// recordLine is a numbered line of the generation record.
var recordLine = regexp.MustCompile(`^ *[0-9]+\. `)

// A note ends with its generation record. Nothing follows it: the ruleset is
// the same on every note and is the README's to say, the build is the
// ctchargen property, and who answered each choice is on that choice's line
// (#525, and the footer dropped on #474's PR).
//
// Holds: A note ends with its generation record.
func TestANoteEndsWithItsGenerationRecord(t *testing.T) {
	t.Parallel()

	policy := chargen.DefaultPolicy()

	for name, change := range map[string]func(c *chargen.Character){
		"a guided run":       func(*chargen.Character) {},
		"an auto run, built": func(c *chargen.Character) { c.Policy, c.Build = &policy, "v1.2.3" },
	} {
		note := noteOf(change)

		_, record, found := strings.Cut(note, "## Generation record\n")
		if !found {
			t.Fatalf("%s: no generation record:\n%s", name, note)
		}

		if strings.Contains(record, "\n---\n") || strings.Contains(record, "Generated under") {
			t.Errorf("%s: something follows the generation record:\n%s", name, record)
		}

		lines := strings.Split(strings.TrimSuffix(note, "\n"), "\n")
		if last := lines[len(lines)-1]; !recordLine.MatchString(last) {
			t.Errorf("%s: the note ends %q, not on a record line", name, last)
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
		c.Events = recorded(
			traveller.RollEvent{Seq: 1, Step: "Table 2, Cash Allowances", Dice: []int{4}, DM: 1},
			traveller.ThrowEvent{Seq: 2, Step: "survival", Dice: []int{3, 4}, Target: fivePlus, Succeeded: true},
			aDischargedSailor(),
		)
	})

	for _, want := range []string{
		"\n1. Table 2, Cash Allowances: rolled 4 +1 = 5\n",
		"\n2. survival: rolled 3+4 = 7 against 5+, made\n",
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
		c.Events = recorded(
			traveller.StepEvent{Seq: 1, Step: "characteristics", Pages: "p. 4"},
			traveller.OutcomeEvent{Seq: 2, Description: "UPP 777777 at age 18"},
			traveller.StepEvent{Seq: 3, Step: "enlistment", Pages: "pp. 5, 10"},
			aDischargedSailor(),
		)
	})

	_, record, _ := strings.Cut(note, "## Generation record\n")

	for line := range strings.Lines(record) {
		if strings.HasPrefix(line, "#") {
			t.Errorf("the generation record carries a heading of its own: %q", line)
		}
	}

	for _, want := range []string{
		"\n**1. characteristics (p. 4)**\n\n",
		"\n2. UPP 777777 at age 18\n",
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

// rolled is the six characteristics as p. 4 rolls them, in that order, which
// is what a note's characteristics are read from.
func rolled(values ...int) []traveller.Fact {
	facts := make([]traveller.Fact, 0, len(values))

	for i, value := range values {
		facts = append(facts, traveller.RolledFact{Characteristic: traveller.Characteristics[i], Value: value})
	}

	return facts
}

// skilled is each skill granted on entering, at the level it reached, which
// is what a note's skills are read from. On entering is the source the
// service history files without a term opened.
func skilled(skills ...traveller.Skill) []traveller.Fact {
	facts := make([]traveller.Fact, 0, len(skills))

	for _, skill := range skills {
		facts = append(facts, applied(traveller.SkillGain{Skill: skill}, traveller.OnEntering))
	}

	return facts
}

// A note opens with properties a vault can query (#399), in the shape decided
// on #474: flat keys in the sheet's own words, single values because a
// character serves one service (Book 1 p. 5), `status` saying outright whether
// the character lived (#418), skills as names alone, possessions one item
// each, and the build's version. Every key is on every note: empty, or [] for
// a list, where there is nothing to say (#480, #477). The UPP is quoted, since
// six digits would otherwise read as a number.
//
// Holds: A note opens with properties a vault can query.
func TestANoteOpensWithItsProperties(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]noteCase{
		"a named officer who took the title, built": {
			func(c *chargen.Character) {
				c.Inputs.Name = "Kira Vance"
				c.Build = "v1.2.3"
				c.Events = recorded(rolled(7, 9, 3, 7, 4, 11),
					skilled(traveller.Skill{Name: "Brawling", Level: 1}, traveller.Skill{Name: "Rifle", Level: 2}),
					traveller.CashFact{Cash: 30000, Total: 30000},
					traveller.TitledFact{SocialStanding: 11, Title: traveller.Knight, Assumed: true},
					enlisted(traveller.Army), opened(2),
					traveller.RaisedFact{How: traveller.Commissioned, Rank: 2, Name: captain},
					traveller.DeniedReenlistmentFact{After: 2, Age: traveller.NewAge(26)})
			},
			"---\nname: \"Kira Vance\"\naliases:\n  - \"Kira Vance\"\nupp: \"79374B\"\n" +
				"strength: 7\ndexterity: 9\nendurance: 3\nintelligence: 7\neducation: 4\nsocial: 11\n" +
				"age: 26\nservice: \"Army\"\nrank: \"Captain\"\nterms: 2\nstatus: \"alive\"\n" +
				"departure: \"denied reenlistment\"\n" +
				"skills:\n  - \"Brawling\"\n  - \"Rifle\"\ncredits: 30000\npension: 0\nnobility: \"knight/dame\"\n" +
				"possessions: []\nctchargen: \"v1.2.3\"\n---\n",
		},
		"an unnamed civilian": {
			func(c *chargen.Character) {
				c.Events = recorded(rolled(7, 7, 7, 7, 7, 7), civilian())
			},
			"---\nname:\naliases: []\nupp: \"777777\"\n" +
				"strength: 7\ndexterity: 7\nendurance: 7\nintelligence: 7\neducation: 7\nsocial: 7\n" +
				"age: 18\nservice:\nrank:\nterms: 0\nstatus: \"alive\"\ndeparture:\nskills: []\ncredits: 0\npension: 0\n" +
				"nobility:\npossessions: []\nctchargen:\n---\n",
		},
		// What a referee types is escaped, not trusted: quotes, a colon, a
		// hash, a backslash and a letter outside ASCII all read back as typed.
		"a name a referee typed": {
			func(c *chargen.Character) {
				c.Inputs.Name = `O'Brien: "Red" #1 \ Mår`
				c.Events = recorded(rolled(7, 7, 7, 7, 7, 7), civilian())
			},
			"---\nname: \"O'Brien: \\\"Red\\\" #1 \\\\ Mår\"\n" +
				"aliases:\n  - \"O'Brien: \\\"Red\\\" #1 \\\\ Mår\"\n",
		},
		"a dead Scout": {
			func(c *chargen.Character) {
				c.Events = recorded(rolled(6, 6, 6, 6, 6, 6), skilled(traveller.Skill{Name: pilot, Level: 1}),
					enlisted(traveller.Scouts), opened(3), traveller.DiedFact{Term: 3, Age: traveller.NewAge(30)})
			},
			"---\nname:\naliases: []\nupp: \"666666\"\n" +
				"strength: 6\ndexterity: 6\nendurance: 6\nintelligence: 6\neducation: 6\nsocial: 6\n" +
				"age: 30\nservice: \"Scouts\"\nrank:\nterms: 3\nstatus: \"dead\"\n" +
				"departure: \"killed by the survival throw\"\n" +
				"skills:\n  - \"Pilot\"\ncredits: 0\npension: 0\nnobility:\npossessions: []\nctchargen:\n---\n",
		},
	} {
		note := noteOf(tc.change)
		if !strings.HasPrefix(note, tc.want) {
			got, _, _ := strings.Cut(note, "\n# ")
			t.Errorf("%s: the note opens\n%s\nwant\n%s", name, got, tc.want)
		}
	}
}

// Each possession is one item: two passages are two, and a Free Trader
// received again is still the one ship (#475).
//
// Holds: Possessions are one item each.
func TestPossessionsAreOneItemEach(t *testing.T) {
	t.Parallel()

	note := noteOf(func(c *chargen.Character) {
		c.Events = recorded(rolled(7, 7, 7, 7, 7, 7), enlisted(traveller.Merchants), opened(5),
			traveller.RetiredFact{After: 5, Age: traveller.NewAge(38)},
			traveller.PassageFact{Class: traveller.LowPassage},
			traveller.PassageFact{Class: traveller.LowPassage},
			traveller.ShipReceivedFact{Ship: traveller.Ship{Kind: traveller.FreeTrader, Tons: 200, PaymentYears: 40}},
			traveller.FreeTraderAgainFact{Ship: traveller.Ship{
				Kind: traveller.FreeTrader, Tons: 200, Years: 10, PaymentYears: 30,
			}})
	})

	const want = "\npossessions:\n  - \"Low Passage\"\n  - \"Low Passage\"\n  - \"Free Trader\"\nctchargen:\n---\n"

	if !strings.Contains(note, want) {
		t.Errorf("the properties do not read %q:\n%s", want, note)
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

		// The summary is the paragraph between the heading line and the UPP
		// line, whatever the heading is.
		_, afterHeading, _ := strings.Cut(note, "\n# ")
		_, body, _ := strings.Cut(afterHeading, "\n\n")

		if got, _, _ := strings.Cut(body, "\n\nUPP "); got != tc.want {
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
				c.Events = recorded(enlisted(traveller.Army), opened(2),
					traveller.RaisedFact{How: traveller.Commissioned, Rank: 2, Name: captain},
					traveller.DeniedReenlistmentFact{After: 2, Age: traveller.NewAge(26)},
					skilled(
						traveller.Skill{Name: "Brawling", Level: 1},
						traveller.Skill{Name: "Medical", Level: 1},
						traveller.Skill{Name: "Rifle", Level: 2},
						traveller.Skill{Name: "Tactics", Level: 1},
					), traveller.CashFact{Cash: 30000, Total: 30000})
			},
			"Army Captain, 2 terms, denied reenlistment at 26. Known for Rifle-2. " +
				"Musters out with CR 30000.",
		},
		"a merchant with a ship": {
			func(c *chargen.Character) {
				c.Events = recorded(enlisted(traveller.Merchants), opened(7),
					traveller.RetiredFact{After: 7, Age: traveller.NewAge(46)},
					skilled(traveller.Skill{Name: pilot, Level: 1}),
					traveller.TravellersAidFact{},
					traveller.ShipReceivedFact{Ship: traveller.Ship{Kind: traveller.FreeTrader, Tons: 200}})
			},
			"Merchants, 7 terms, retired at 46. " +
				"Musters out with Travellers' Aid Society membership and a Free Trader, Type A.",
		},
		"a civilian": {
			func(c *chargen.Character) { c.Events = recorded(civilian()) },
			"A civilian who declined the draft at 18.",
		},
		"a Scout with his ship": {
			func(c *chargen.Character) {
				c.Events = recorded(enlisted(traveller.Scouts), opened(7),
					traveller.RetiredFact{After: 7, Age: traveller.NewAge(46)},
					skilled(traveller.Skill{Name: pilot, Level: 2}, traveller.Skill{Name: "Electronic", Level: 3}),
					traveller.ShipReceivedFact{Ship: traveller.Ship{Kind: traveller.ScoutShip, Tons: 100}})
			},
			"Scouts, 7 terms, retired at 46. Known for Electronic-3 and Pilot-2. " +
				"Musters out with a Scout ship, Type S, held in constructive possession.",
		},
		"a dead Scout": {
			func(c *chargen.Character) {
				c.Events = recorded(enlisted(traveller.Scouts), opened(3),
					traveller.DiedFact{Term: 3, Age: traveller.NewAge(30)},
					skilled(traveller.Skill{Name: pilot, Level: 1}))
			},
			"Scouts, killed by the survival throw in term 3, aged 30.",
		},
	}
}

// A note is headed by its character's name, and an unnamed one by his title -
// the words its batch file is named by, less the position - so an unnamed
// note reads sensibly and is named in a vault by setting its name property
// (#406).
//
// Holds: An unnamed note is headed by its title.
func TestAnUnnamedNoteIsHeadedByItsTitle(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]noteCase{
		"an unnamed officer": {
			func(c *chargen.Character) {
				c.Events = recorded(rolled(9, 3, 8, 7, 8, 8), enlisted(traveller.Army), opened(1),
					traveller.RaisedFact{How: traveller.Commissioned, Rank: 2, Name: captain},
					traveller.DeniedReenlistmentFact{After: 1, Age: traveller.NewAge(22)})
			},
			"\n# Army Captain 938788\n",
		},
		"an unnamed civilian": {
			func(c *chargen.Character) {
				c.Events = recorded(rolled(8, 7, 6, 6, 5, 11), civilian())
			},
			"\n# Civilian 87665B\n",
		},
		"a named one": {
			func(c *chargen.Character) {
				c.Inputs.Name = "Kira Vance"
				c.Events = recorded(civilian())
			},
			"\n# Kira Vance\n",
		},
	} {
		note := noteOf(tc.change)
		if !strings.Contains(note, tc.want) || strings.Contains(note, "(unnamed)") {
			heading, _, _ := strings.Cut(note[strings.Index(note, "\n# ")+1:], "\n")
			t.Errorf("%s is headed %q, want %q", name, heading, strings.TrimSpace(tc.want))
		}
	}
}

// A change the floor or the ceiling held says so (#409): the table's change
// as it printed it, then where the characteristic stayed - what the
// generation record's "-1, 1 to 1" says. An unclamped change is the delta alone.
//
// Holds: A clamped change says the table's change and where it was held.
func TestAClampedChangeSaysWhereItWasHeld(t *testing.T) {
	t.Parallel()

	note := noteOf(func(c *chargen.Character) {
		c.Events = recorded(
			enlisted(traveller.Other),
			traveller.Term(1),
			applied(traveller.Change{Characteristic: traveller.SocialStanding, From: 1, To: 1, Delta: -1}, traveller.InTerm),
			applied(traveller.Change{Characteristic: traveller.Education, From: 10, To: 11, Delta: 1}, traveller.InTerm),
			traveller.AgedFact{Change: traveller.Change{Characteristic: traveller.Strength, From: 1, To: 0, Delta: -2}},
			traveller.DischargedFact{After: 1, Age: traveller.NewAge(22)},
		)
	})

	want := "- Term 1: Social Standing -1 (held at 1), Education +1; aging Strength -2 (held at 0)\n"
	if !strings.Contains(note, want) {
		t.Errorf("the service record does not read %q:\n%s", want, note)
	}
}

// recorded is a record built from the terms it opens and the facts its outcomes
// carry, in order, which is what a note's sheet, career and service record
// are read from. A list of facts or of entries is taken in its order, and an
// event is taken as it is.
func recorded(entries ...any) []traveller.Event {
	events := make([]traveller.Event, 0, len(entries))

	for _, entry := range entries {
		seq := len(events) + 1

		switch entry := entry.(type) {
		case traveller.Term:
			events = append(events, traveller.StepEvent{
				Seq: seq, Step: fmt.Sprintf("term %d", entry), Pages: "pp. 5-7", Opens: entry,
			})
		case traveller.Fact:
			events = append(events, traveller.OutcomeEvent{Seq: seq, Description: entry.String(), Fact: entry})
		case []traveller.Fact:
			for _, fact := range entry {
				events = append(events, traveller.OutcomeEvent{
					Seq: len(events) + 1, Description: fact.String(), Fact: fact,
				})
			}
		case []any:
			for _, event := range recorded(entry...) {
				events = append(events, renumbered(event, len(events)+1))
			}
		case traveller.Event:
			events = append(events, entry)
		}
	}

	return events
}

// applied is a result applied from a source.
func applied(gain traveller.Gain, source traveller.Source) traveller.AppliedFact {
	return traveller.AppliedFact{Gain: gain, Source: source}
}

// renumbered is an event recorded on its own, numbered where it lands.
func renumbered(event traveller.Event, seq int) traveller.Event {
	switch event := event.(type) {
	case traveller.StepEvent:
		event.Seq = seq

		return event
	case traveller.OutcomeEvent:
		event.Seq = seq

		return event
	}

	return event
}

// enlisted is the enlistment throw that accepted him into service.
func enlisted(service traveller.ServiceName) traveller.EnlistedFact {
	return traveller.EnlistedFact{Service: service, Accepted: true}
}

// draftedAfter is the throw that refused enlistment, and the draft that
// followed.
func draftedAfter(refused, into traveller.ServiceName) []traveller.Fact {
	return []traveller.Fact{traveller.EnlistedFact{Service: refused}, traveller.DraftedFact{Service: into}}
}

// opened is the steps that open terms 1 to n, which is what a note's term
// count is read from.
func opened(n int) []any {
	terms := make([]any, 0, n)

	for term := 1; term <= n; term++ {
		terms = append(terms, traveller.Term(term))
	}

	return terms
}

// aDischargedSailor is the least career the sheet reads: a Navy enlistment,
// and a discharge at the age he started at.
func aDischargedSailor() []any {
	return []any{enlisted(traveller.Navy), traveller.DischargedFact{Age: traveller.NewAge(18)}}
}

// civilian is the age every character starts at, and the draft declined.
func civilian() []any {
	return []any{traveller.StartedFact{Age: traveller.NewAge(18)}, traveller.DeclinedFact{}}
}
