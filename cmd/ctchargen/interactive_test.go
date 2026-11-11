package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The prompt loop, driven from a scripted reader so the whole of it is
// covered without a terminal.

// answers supplies the same answer as often as it is asked for, which walks
// a character by always taking the first option offered.
func answers(choice string, times int) io.Reader {
	return strings.NewReader(strings.Repeat(choice+"\n", times))
}

// unusableFirst puts three answers the prompt cannot use in front of such a
// walk: a word, a number below the range, and one above it. Both tests of
// the refusal drive the same three, so they are written once - a walk that
// answered differently would be testing something else.
func unusableFirst() io.Reader {
	return io.MultiReader(strings.NewReader("banana\n0\n99\n"), answers("1", 200))
}

// A bare new walks the procedure, asking at every choice point and showing
// what happened between the questions.
//
// Holds: Every question the player is asked is one the engine asked.
func TestInteractiveWalksACharacter(t *testing.T) {
	t.Parallel()

	note, shown := mustRun(t, []string{cmdNew, flagService, other}, answers("1", 200),
		withDice(otherWalked.roller(t)))

	for _, want := range []string{
		// The procedure, shown as it runs. The step lines carry their
		// sequence number, which is what stops the numbers appearing to
		// skip past them.
		"\n1. characteristics (p. 4)\n",
		"Strength: rolled",
		"\n18. term 1 (pp. 5-7)\n",
		"survival: rolled",
		// A question, with the alternatives the engine offered.
		"Which skills table? Roll 1 of 2",
		"1) Personal Development Table",
	} {
		if !strings.Contains(shown, want) {
			t.Errorf("the run never showed %q", want)
		}
	}

	// And the note at the end, on the channel the player did not watch.
	if !strings.HasPrefix(note, "---\nname:") || !strings.Contains(note, "UPP ") {
		t.Errorf("the run did not end with the note:\n%s", note)
	}

	// A choice is echoed back, and in one line.
	//
	// It used to be shown as nothing at all, on the reasoning that the
	// player answered it a line earlier and repeating it pushes the useful
	// lines off the screen. That reasoning still holds against the
	// generation record's own form - "submit to the draft (p. 5): player chose yes from yes, no"
	// - which names the point, the decider and every alternative he can
	// still see above him. What it does not survive is the cost: a choice
	// holds a sequence number, and showing nothing left the numbers with
	// holes in them that read as lost events.
	//
	// So the echo is the number and the answer, and nothing else.
	if strings.Contains(shown, "player chose") {
		t.Error("the live view used the generation record's form, which repeats what he can see")
	}

	// Named against the answer, not against the words around it. An echo
	// reading "you chose " with nothing after it keeps the numbering whole
	// and says nothing - which is what happens the moment the event stops
	// carrying what was chosen, a drift the fold cannot catch because the
	// case still compiles. This holds what the player is shown.
	if !strings.Contains(shown, "you chose Personal Development Table") {
		t.Error("the echo does not name what was chosen")
	}
}

// An answer it cannot read is asked again rather than guessed at.
//
// Holds: An unreadable answer is asked again, never guessed at.
func TestInteractiveReAsksWhatItCannotRead(t *testing.T) {
	t.Parallel()

	out := mustRunTogether(t, []string{cmdNew, flagService, other}, unusableFirst(),
		withDice(otherWalked.roller(t)))

	shown := out

	if strings.Count(shown, "answer with one of the numbers shown") != 3 {
		t.Errorf("three unusable answers drew %d re-prompts, want 3",
			strings.Count(shown, "answer with one of the numbers shown"))
	}

	// The complaint names what was typed, so a reader can see which of the
	// answers was refused rather than inferring it.
	for _, given := range []string{`"banana"`, `"0"`, `"99"`} {
		if !strings.Contains(shown, given+" is refused") {
			t.Errorf("the complaint does not name %s", given)
		}
	}

	if !strings.Contains(shown, "UPP ") {
		t.Error("the run did not finish after the answers became usable")
	}
}

// A bad answer does not re-print the menu.
//
// Re-printing it is what buried the complaint: the message landed on the
// prompt row and six lines of options followed, so the tool read as though it
// had said nothing. The question is asked once and the prompt comes back on
// its own.
//
// Driven without --service, because the enlistment menu is the one question
// asked exactly once in a run - counting it is meaningless for a question the
// procedure puts every term.
func TestABadAnswerDoesNotReprintTheMenu(t *testing.T) {
	t.Parallel()

	out := mustRunTogether(t, []string{cmdNew}, unusableFirst())

	shown := out

	const question = "Which service will you try to enlist in?"

	if asked := strings.Count(shown, question); asked != 1 {
		t.Errorf("three bad answers printed the menu %d times, want 1", asked)
	}

	if refused := strings.Count(shown, " is refused; "); refused != 3 {
		t.Errorf("three bad answers drew %d complaints, want 3", refused)
	}
}

// A question says how to answer it before any answer is refused, and a
// refusal says what it wants (#461). The prompt carries the range; a word and
// an empty line each draw the same nudge, and only a typed word is quoted
// back, since an empty line has nothing to quote.
//
// Held at the enlistment menu, the first question a bare new asks; every
// question is put through the one prompt loop, so this is every question's
// wording.
//
// Holds: A question shows the numbers it accepts before it is answered.
func TestAQuestionSaysHowToAnswerIt(t *testing.T) {
	t.Parallel()

	shown := mustRunTogether(t, []string{cmdNew},
		io.MultiReader(strings.NewReader("army\n\n"), answers("1", 300)))

	const menu = "  6) Other (enlistment 3+, DM +0)\nanswer 1-6> "

	for _, want := range []string{
		// The range, at the prompt, before anything was typed.
		menu,
		// A word is quoted back, and the nudge says what is wanted.
		menu + `  "army" is refused; answer with one of the numbers shown, 1-6` + "\nanswer 1-6> ",
		// An empty line is given the nudge alone.
		"\nanswer 1-6>   answer with one of the numbers shown, 1-6\nanswer 1-6> ",
	} {
		if !strings.Contains(shown, want) {
			t.Errorf("the session never showed %q:\n%s", want, shown)
		}
	}

	for _, gone := range []string{`""`, "one of them", "\n> "} {
		if strings.Contains(shown, gone) {
			t.Errorf("the session still shows %q:\n%s", gone, shown)
		}
	}
}

// The numbers a player watches do not skip.
//
// They appeared to - 17, 18, then 20 - because the headings and the questions
// hold sequence numbers and rendered unnumbered. Nothing was ever missing, but
// a gap in a record whose whole purpose is auditability costs a reader
// trust he should not have to spend.
func TestTheNumbersAPlayerWatchesDoNotSkip(t *testing.T) {
	t.Parallel()

	_, out := mustRun(t, []string{cmdNew, flagService, other}, answers("1", 200),
		withDice(otherWalked.roller(t)))

	numbered := regexp.MustCompile(`(?m)^\s*(\d+)\. `)

	lines := numbered.FindAllStringSubmatch(out, -1)
	seen := make([]int, 0, len(lines))

	for _, m := range lines {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("reading %q: %v", m[1], err)
		}

		seen = append(seen, n)
	}

	if len(seen) < 2 {
		t.Fatalf("found %d numbered lines in\n%s", len(seen), out)
	}

	for i := 1; i < len(seen); i++ {
		if seen[i] != seen[i-1]+1 {
			t.Errorf("the sequence jumps from %d to %d", seen[i-1], seen[i])
		}
	}
}

// A skills table has one name, and the menu, the echo and the record all use
// it (#486). The fourth table's menu line once read "Education 8+ table (the
// second Advanced Education table)" where the record read "Advanced
// Education Table (education 8+)": two names for one table, which a reader
// matching the question to the note had to reconcile. The record's agrees
// with p. 11, so it is the one kept; the six results shown beside each name
// are what tells the two Advanced Education tables apart.
func TestASkillsTableHasOneName(t *testing.T) {
	t.Parallel()

	// Answering 4 takes the fourth table at the drafted walk's first skills
	// question, whose Education already opens it.
	in := io.MultiReader(strings.NewReader("1\n4\n"), answers("1", 300))

	note, shown := mustRun(t, []string{cmdNew, flagService, navy}, in,
		withDice(draftedWalked.roller(t)))

	const name = "Advanced Education Table (education 8+)"

	for where, text := range map[string]string{
		"the menu": "  4) " + name + ": ",
		"the echo": "you chose " + name + "\n",
	} {
		if !strings.Contains(shown, text) {
			t.Errorf("%s does not name the table %q:\n%s", where, name, shown)
		}
	}

	if !strings.Contains(note, "player chose "+name+" from ") {
		t.Errorf("the record does not name the table %q:\n%s", name, note)
	}

	if strings.Contains(shown, "Education 8+ table") {
		t.Errorf("the menu still has a second name for the table:\n%s", shown)
	}
}

// The input ending is an error, not a default. A character finished by a
// decision nobody made is not the character anybody asked for.
//
// Holds: The input ending is an error, not a default.
func TestInteractiveRefusesToInventAnAnswer(t *testing.T) {
	t.Parallel()

	for name, script := range map[string]io.Reader{
		"nothing at all":  strings.NewReader(""),
		"nil reader":      nil,
		"answers run out": answers("1", 2),
	} {
		// A walk that lives to be asked: random dice can kill a character in
		// his first survival throw, before any question is put. Partly,
		// because the input ends before the walk does.
		err := run([]string{cmdNew, flagService, other}, script, io.Discard, io.Discard,
			withDice(otherWalked.partly(t)))

		switch {
		case err == nil:
			t.Errorf("%s: the run finished without being answered", name)
		case !strings.Contains(err.Error(), "the input ended"):
			t.Errorf("%s: error %q does not say the input ended", name, err)
		}
	}
}

// The observer shows the procedure. Without it the player would answer
// blind, which is most of what a guided run is for.
//
// Holds: The procedure is shown between the questions.
func TestInteractiveShowsWhatAutoDoesNot(t *testing.T) {
	t.Parallel()

	_, asked := mustRun(t, []string{cmdNew, flagService, other}, answers("1", 200),
		withDice(otherWalked.roller(t)))

	_, automatic := mustRun(t, []string{cmdNew, flagAuto, flagService, other}, nil,
		withDice(otherByPolicy.roller(t)))

	if strings.Contains(automatic, "survival: rolled") {
		t.Error("--auto showed the procedure; nobody is reading, so it passes no observer")
	}

	if !strings.Contains(asked, "survival: rolled") {
		t.Error("the guided run did not show the survival throw")
	}
}

// The yes-or-no points, which the walk through Other never reaches: it is
// never rejected, holds no rank, and ends nobody's noble.
//
// The drafted walk is rejected by the Navy, drafted into the Marines, and
// serves seven terms to a rank and a title. Each prompt below is asserted
// rather than merely walked past, so a change to what the procedure asks
// fails here instead of quietly leaving these branches unasked.
//
// Holds: The alternatives offered are exactly the engine's.
func TestInteractiveAsksTheYesOrNoPoints(t *testing.T) {
	t.Parallel()

	out := mustRunTogether(t, []string{cmdNew, flagService, navy}, answers("1", 300),
		withDice(draftedWalked.roller(t)))

	shown := out

	for _, want := range []string{
		"Rejected. Submit to the draft?",
		"Attempt a commission this term?",
		"Attempt a promotion this term?",
		"Your rank allows +1 on this table. Take it? (p. 9)",
		"Assume the title? (p. 5; Book 3 p. 22)",
		// Both alternatives are offered at each of them.
		"1) yes",
		"2) no",
	} {
		if !strings.Contains(shown, want) {
			t.Errorf("the run never asked %q", want)
		}
	}
}

// A question that leads to a throw shows the throw before it is answered
// (#482): the commission and the promotion their target and DM, and what a
// failure costs, for which p. 6 names none; the end of a term the survival
// throw the next term makes. Each is the Marines' as the drafted walk reaches
// it, read from the transcription: a mutation to the p. 10 cell moves the
// prompt with it.
//
// Holds: A throw's question shows its target and DM before the answer.
func TestAThrowsQuestionShowsTheThrow(t *testing.T) {
	t.Parallel()

	shown := mustRunTogether(t, []string{cmdNew, flagService, navy}, answers("1", 300),
		withDice(draftedWalked.roller(t)))

	for _, want := range []string{
		"\nAttempt a commission this term? (commission 9+, DM +1 for Education 7+; p. 6 names no cost for a failure)\n" +
			"  1) yes\n  2) no\nanswer 1-2> ",
		"\nAttempt a promotion this term? (promotion 9+, DM +1 for Social Standing 8+; p. 6 names no cost for a failure)\n" +
			"  1) yes\n  2) no\nanswer 1-2> ",
		"\nThe term is over. What now? (pp. 6-7, 21)\n" +
			"  1) continue (next term's survival 6+, DM +2 for Endurance 8+, before any aging)\n" +
			"  2) discharge\nanswer 1-2> ",
	} {
		if !strings.Contains(shown, want) {
			t.Errorf("the session never asked\n%s", want)
		}
	}
}

// The enlistment menu shows each service's target and the DM the character
// earns against it, built from the same Odds the commission and survival
// questions are handed; its wording is its own. Driven by the drafted walk's
// dice without --service: answering 1 attempts the Navy, the walk's own
// path, and the menu asks without a die.
//
// The DMs are worked by hand from the walk's characteristics, Strength 7,
// Dexterity 6, Endurance 11, Intelligence 7, Education 11, against the p. 10
// enlistment DMs: the Navy +2 for Education 9+; the Marines none; the Army +1
// for Dexterity 6+ and +2 for Endurance 5+; the Scouts +1 for Intelligence
// 6+; the Merchants +1 for Strength 7+ and +2 for Intelligence 6+.
//
// Holds: A throw's question shows its target and DM before the answer.
func TestTheEnlistmentMenuShowsEachThrow(t *testing.T) {
	t.Parallel()

	shown := mustRunTogether(t, []string{cmdNew}, answers("1", 300), withDice(draftedWalked.roller(t)))

	const want = "" +
		"  1) Navy (enlistment 8+, DM +2 for Education 9+)\n" +
		"  2) Marines (enlistment 9+, DM +0)\n" +
		"  3) Army (enlistment 5+, DM +3 for Dexterity 6+, Endurance 5+)\n" +
		"  4) Scouts (enlistment 7+, DM +1 for Intelligence 6+)\n" +
		"  5) Merchants (enlistment 7+, DM +3 for Strength 7+, Intelligence 6+)\n" +
		"  6) Other (enlistment 3+, DM +0)\n"

	if !strings.Contains(shown, want) {
		t.Errorf("the enlistment menu is not\n%s\nin\n%s", want, shown)
	}
}

// Declining the draft ends the generation, and the question says so before
// it is answered (#483). The answer and its outcome are unchanged. The
// consequence is E001's reading, so the page cite stays on the question and
// is not put after the consequence as though p. 5 stated it.
//
// Holds: The draft question says that declining ends generation.
func TestTheDraftQuestionSaysWhatDecliningCosts(t *testing.T) {
	t.Parallel()

	shown := mustRunTogether(t, []string{cmdNew, flagService, navy}, answers("1", 300),
		withDice(draftedWalked.roller(t)))

	const want = "\nRejected. Submit to the draft? (p. 5) " +
		"Declining ends the generation with a character who never served.\n"

	if !strings.Contains(shown, want) {
		t.Errorf("the session never asked %q:\n%s", want, shown)
	}
}

// A person may answer no to the draft, and the note is then a civilian's: no
// strategy declines (#566), so this is the one way a civilian is made.
func TestDecliningTheDraftWritesACivilian(t *testing.T) {
	t.Parallel()

	note, _ := mustRun(t, []string{cmdNew, flagService, navy}, answers("2", 10),
		withDice(navyDeclined.roller(t)))

	for _, want := range []string{
		"# Civilian 87665B",
		"submit to the draft (p. 5): player chose no from yes, no",
	} {
		if !regexp.MustCompile(`(?m)^(\d+\. )?` + regexp.QuoteMeta(want) + `$`).MatchString(note) {
			t.Errorf("the note has no line %q:\n%s", want, note)
		}
	}
}

// The skills question says which roll of how many it is and what gave the
// term that many, and each table offered shows its six results (#486). Held
// at the drafted walk's third term, which is commissioned and promoted, so
// all three sources of a roll are named; and at its first, where the term's
// two are all there is.
//
// Each table is named once, as the record names it: the fourth is "Advanced
// Education Table (education 8+)", which p. 11 prints as "Advanced Education
// Table (allowed only for characters with education 8+)".
//
// Holds: The skills question says which roll of how many, and why.
// Holds: A skills table offered shows its six results.
func TestTheSkillsQuestionSaysWhichRollAndWhatEachTableHolds(t *testing.T) {
	t.Parallel()

	shown := mustRunTogether(t, []string{cmdNew, flagService, navy}, answers("1", 300),
		withDice(draftedWalked.roller(t)))

	//nolint:dupword // the Marines' column of p. 11 prints Tactics twice, and Blade and Gun Cbt
	const marines = "" +
		"  1) Personal Development Table: +1 Strength, +1 Dexterity, +1 Endurance, Gambling, Brawling, Blade Combat\n" +
		"  2) Service Skills Table: ATV, Vacc Suit, Blade Combat, Blade Combat, Gun Combat, Gun Combat\n" +
		"  3) Advanced Education Table: ATV, Mechanical, Electronic, Tactics, Blade Combat, Gun Combat\n" +
		"  4) Advanced Education Table (education 8+): Medical, Tactics, Tactics, Computer, Leader, Administration\n" +
		"answer 1-4> "

	for _, want := range []string{
		"\nWhich skills table? Roll 1 of 2: 2 for the term. It is designated before the die (p. 11)\n" + marines,
		"\nWhich skills table? Roll 2 of 2: 2 for the term. It is designated before the die (p. 11)\n",
		"\nWhich skills table? Roll 1 of 3: 1 for the term, 1 for the commission, 1 for the promotion. " +
			"It is designated before the die (p. 11)\n" + marines,
		"\nWhich skills table? Roll 3 of 3: 1 for the term, 1 for the commission, 1 for the promotion. ",
	} {
		if !strings.Contains(shown, want) {
			t.Errorf("the session never asked\n%s\nin\n%s", want, shown)
		}
	}
}

// The gambling modifier is offered only to someone who has the expertise, so
// reaching it takes a walk that draws the skill: the gambler, taking the
// Service Skills Table every time. Answering 2 there declines it, which no
// strategy does (#570), so this is the one way the refusal is reached: the
// record says the player chose no, and the roll after it is read at its die.
//
// Holds: The gambling modifier is offered only to one who has the expertise.
// Holds: The +1 with gambling on Table 2 may be declined.
func TestInteractiveOffersTheGamblingModifier(t *testing.T) {
	t.Parallel()

	var note, asked strings.Builder

	err := run([]string{cmdNew, flagService, other}, answers("2", 300), &note, &asked,
		withDice(gamblerWalked.roller(t)))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(asked.String(), "Your gambling expertise allows +1 on this table") {
		t.Error("a character with Gambling was never offered the modifier p. 9 gives him")
	}

	declined := regexp.MustCompile(`(?m)^\d+\. DM \+1 on Table 2 for gambling expertise \(pp\. 7, 9\): ` +
		`player chose no from yes, no\n\d+\. Table 2, Cash Allowances: rolled \d$`)
	if !declined.MatchString(note.String()) {
		t.Errorf("the note does not record the modifier declined and the roll read without it:\n%s", note.String())
	}
}

// A player nobody can be shown anything is not asked anything either. The
// failure is remembered and reported at the next question, rather than being
// swallowed once per line for the length of a career.
//
// Holds: A player nobody can be shown anything is not asked.
func TestInteractiveStopsWhenItCannotBeRead(t *testing.T) {
	t.Parallel()

	// Partly: the run stops at the first question nobody can be shown.
	err := run([]string{cmdNew, flagService, other},
		answers("1", 300), io.Discard, &failsAfter{}, withDice(otherWalked.partly(t)))

	switch {
	case err == nil:
		t.Fatal("the run finished while nobody could see the questions")
	case !errors.Is(err, errWriteFailed):
		t.Errorf("error %q does not carry the write failure that caused it", err)
	}
}

// The questions do not go where the note goes.
//
// A guided run writes its note to stdout, so `ctchargen new > kira.md` has to
// save a note and not a conversation. The two channels are separate
// arguments for that reason, and this is what holds them apart.
//
// Holds: The questions go to stderr, not into the note.
// Holds: A guided run's choices are recorded as the player's, not the policy's.
func TestTheQuestionsStayOutOfTheNote(t *testing.T) {
	t.Parallel()

	note, asked := mustRun(t, []string{cmdNew, flagService, other}, answers("1", 300),
		withDice(otherWalked.roller(t)))

	if !strings.Contains(asked, "Which skills table?") {
		t.Error("the questions did not go to the channel meant for them")
	}

	if strings.Contains(note, "Which skills table?") || !strings.HasPrefix(note, "---\nname:") {
		t.Errorf("the data channel does not hold a note on its own:\n%s", note)
	}

	// And the note names who answered. A guided run's choices were the
	// player's: a note that called them the policy's would misdescribe how
	// the character was made - the one thing the record exists to say.
	if !strings.Contains(note, ": player chose ") {
		t.Fatal("the note holds no choice the player made, so it asserts nothing about who answered")
	}

	if strings.Contains(note, ": policy ") {
		t.Errorf("a guided note credits the policy with a choice:\n%s", note)
	}
}

// A session cut short says what was answered and that nothing was written.
//
// The half-built character is lost: it has no note, nothing carries its
// dice, and there is no way back to the question it stopped on (ADR 0001).
// The message is honest about the loss rather than silent, and it offers
// nothing the tool cannot honour.
//
// Holds: A stopped session says what was answered and that nothing was written.
func TestAStoppedSessionSaysWhatWasLost(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct{ typed, wants string }{
		"after some answers": {typed: "1\n2\n1\n", wants: "Answered 1, 2, 1;"},
		"before any answer":  {typed: "", wants: "Nothing was answered"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var asking strings.Builder

			// Partly: the session is stopped short, which is the point.
			err := run([]string{cmdNew, flagService, other},
				strings.NewReader(tc.typed), io.Discard, &asking, withDice(otherWalked.partly(t)))
			if !errors.Is(err, errNoAnswer) {
				t.Fatalf("stopping short gave %v, want the input ending", err)
			}

			for _, want := range []string{tc.wants, "nothing was written"} {
				if !strings.Contains(asking.String(), want) {
					t.Errorf("the message does not say %q:\n%s", want, asking.String())
				}
			}

			for _, gone := range []string{"--seed", "--answers", "Re-run", "re-run", "same command"} {
				if strings.Contains(asking.String(), gone) {
					t.Errorf("the message offers a way back in, %q, that nothing honours:\n%s",
						gone, asking.String())
				}
			}
		})
	}
}

// A read that fails is not the input ending.
//
// bufio.Scanner returns false for both, and only one of them is the reader
// running out. A line past its 64KB bound comes back the same way, and
// blaming the input for ending sends the reader looking in the wrong place.
//
// Holds: A read that fails is not the input ending.
func TestAFailedReadIsNotTheInputEnding(t *testing.T) {
	t.Parallel()

	// One answer that reads, then a line longer than bufio's default
	// buffer, which Scan refuses.
	huge := "1\n" + strings.Repeat("9", 128*1024) + "\n"

	var asking strings.Builder

	// Partly: the run stops at the line it cannot read.
	err := run([]string{cmdNew, flagService, other},
		strings.NewReader(huge), io.Discard, &asking, withDice(otherWalked.partly(t)))

	switch {
	case err == nil:
		t.Fatal("a line too long to read was accepted")
	case errors.Is(err, errNoAnswer):
		t.Errorf("a failed read was reported as the input ending: %v", err)
	case !strings.Contains(err.Error(), "reading the answer"):
		t.Errorf("error %q does not say the read failed", err)
	}

	// The stop the operator did not choose is the one that most needs telling
	// what it cost, so the loss is reported whichever way the answers ran out.
	for _, want := range []string{"Answered 1;", "nothing was written"} {
		if !strings.Contains(asking.String(), want) {
			t.Errorf("a failed read was not told what was lost, %q:\n%s", want, asking.String())
		}
	}
}

// Each line of the weapon-benefit menu records the benefit it says (#247).
//
// The engine's gate checks only that an answer is among those offered, and a
// plain Dagger for a Dagger already held is offered too, so an expertise line
// that carried TakeWeapon would pass it. Only a walk that answers past the
// weapon lines shows what the expertise line means: navyRearmed's second
// Blade Combat menu, answered here, and every other question answered 1.
//
// The menu says "+1 expertise in the Dagger" and the record says "expertise
// in Dagger": two wordings for two readers, and this holds each to its own.
//
// Holds: Each weapon-benefit line records the benefit it says.
func TestEachWeaponLineRecordsWhatItSays(t *testing.T) {
	t.Parallel()

	// askedFirst is how many questions navyRearmed puts before its second
	// Blade Combat menu, and expertiseLine is what only that menu prints: the
	// first had received nothing to offer expertise in.
	const (
		askedFirst    = 29
		expertiseLine = "12) +1 expertise in the Dagger"
	)

	for name, tc := range map[string]struct {
		answer string
		line   string
		want   armed
	}{
		"the expertise line": {
			answer: "12", line: expertiseLine,
			want: armed{weapons: []string{dagger}, daggerSkill: true, chosen: "expertise in Dagger"},
		},
		"a weapon line": {
			answer: "4", line: "4) Cutlass",
			want: armed{weapons: []string{dagger, "Cutlass"}, daggerSkill: false, chosen: "Cutlass"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			in := io.MultiReader(answers("1", askedFirst), strings.NewReader(tc.answer+"\n"), answers("1", 50))

			record, asking := mustRun(t, []string{cmdNew, flagService, navy}, in,
				withDice(navyRearmed.roller(t)))

			if !strings.Contains(asking, expertiseLine) {
				t.Fatalf("navyRearmed never reached a second Blade Combat menu: %q was never offered",
					expertiseLine)
			}

			got := armedBy(t, record)

			if !slices.Equal(got.weapons, tc.want.weapons) {
				t.Errorf("answering %q recorded the weapons %v, want %v", tc.line, got.weapons, tc.want.weapons)
			}

			if got.daggerSkill != tc.want.daggerSkill {
				t.Errorf("answering %q left Dagger-1 among the skills = %v, want %v",
					tc.line, got.daggerSkill, tc.want.daggerSkill)
			}

			if got.chosen != tc.want.chosen {
				t.Errorf("answering %q recorded the choice %q, want %q", tc.line, got.chosen, tc.want.chosen)
			}
		})
	}
}

// dagger is the weapon navyRearmed takes at its first Blade Combat benefit.
const dagger = "Dagger"

// armed is what a note says of the weapon benefits it took: the weapons,
// whether Dagger-1 is among the skills, and the last weapon-benefit choice
// in the record's own wording.
type armed struct {
	weapons     []string
	daggerSkill bool
	chosen      string
}

// armedBy reads the note: the weapons among its benefits, whether
// Dagger-1 is among its skills, and the last weapon-benefit choice on its
// generation record, in the record's own wording.
func armedBy(t *testing.T, note string) armed {
	t.Helper()

	const weaponBenefit = "weapon benefit, or +1 expertise in lieu (p. 22): player chose "

	var found armed

	section := ""

	for line := range strings.Lines(note) {
		line = strings.TrimRight(line, "\n")

		if heading, ok := strings.CutPrefix(line, "## "); ok {
			section = heading

			continue
		}

		item, isItem := strings.CutPrefix(line, "- ")

		switch {
		case section == "Skills" && item == dagger+"-1":
			found.daggerSkill = true
		case section == "Benefits" && isItem && isWeapon(item):
			found.weapons = append(found.weapons, item)
		case strings.Contains(line, weaponBenefit):
			_, chosen, _ := strings.Cut(line, weaponBenefit)

			found.chosen, _, _ = strings.Cut(chosen, " from ")
		}
	}

	return found
}

// isWeapon is whether a line of the sheet's benefits is a weapon received as
// a benefit: the sheet lists those by name alone, where every other line is
// money, a passage, a membership, a pension or a ship.
func isWeapon(benefit string) bool {
	for _, other := range []string{"CR ", "Passage", "Travellers' Aid", "retirement pay", " tons"} {
		if strings.Contains(benefit, other) {
			return false
		}
	}

	return true
}

// The live view prints a step as a plain numbered line, without the note's
// heading marks or any bold: a terminal does not render them, and `###` or
// `**` would reach the player as marks (#422).
//
// Holds: The live view marks a step with no markdown.
func TestTheLiveViewMarksNothing(t *testing.T) {
	t.Parallel()

	_, shown := mustRun(t, []string{cmdNew, flagService, other}, answers("1", 200),
		withDice(otherWalked.roller(t)))

	for _, mark := range []string{"###", "**"} {
		if strings.Contains(shown, mark) {
			t.Errorf("the live view printed %q:\n%s", mark, shown)
		}
	}
}

// unread is an input the test fails on reading: a session refused before its
// first question reads no answer.
type unread struct{ t *testing.T }

func (u unread) Read([]byte) (int, error) {
	u.t.Error("a refused session read an answer")

	return 0, io.EOF
}

// A guided `new -o` onto a path that is already there is refused before the
// first question (#424), rather than after the whole session, when the
// player's answers were lost with it. A file and a directory are both
// refused, and the file is left as it was.
//
// Holds: A guided session onto an existing -o is refused before it asks.
func TestAGuidedSessionOntoAnExistingPathIsRefusedFirst(t *testing.T) {
	t.Parallel()

	for name, plant := range map[string]func(path string) error{
		"a file":      func(path string) error { return os.WriteFile(path, []byte("keep\n"), 0o600) },
		"a directory": func(path string) error { return os.Mkdir(path, 0o700) },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "kira.md")

			err := plant(path)
			if err != nil {
				t.Fatal(err)
			}

			var asked strings.Builder

			err = run([]string{cmdNew, flagService, other, flagOutput, path}, unread{t}, io.Discard, &asked)
			if err == nil || !strings.Contains(err.Error(), path) {
				t.Errorf("a session onto %s said %v, want it refused naming the path", name, err)
			}

			if asked.Len() != 0 {
				t.Errorf("a refused session asked:\n%s", asked.String())
			}

			if name == "a file" {
				kept, readErr := os.ReadFile(path)
				if readErr != nil || string(kept) != "keep\n" {
					t.Errorf("the file was left as %q (%v)", kept, readErr)
				}
			}
		})
	}
}

// With --force a session onto a file runs and overwrites it; onto a
// directory it is still refused before it asks, since --force cannot
// overwrite a directory either (#424).
func TestForceRunsTheSessionUnlessADirectoryIsInTheWay(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file, folder := filepath.Join(dir, "kira.md"), filepath.Join(dir, "folder.md")

	err := os.WriteFile(file, []byte("keep\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Mkdir(folder, 0o700)
	if err != nil {
		t.Fatal(err)
	}

	err = run([]string{cmdNew, flagService, other, flagOutput, folder, flagForce}, unread{t}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("--force onto a directory said %v, want it refused as one before asking", err)
	}

	err = run([]string{cmdNew, flagService, other, flagOutput, file, flagForce},
		answers("1", 200), io.Discard, io.Discard, withDice(otherWalked.roller(t)))
	if err != nil {
		t.Fatalf("--force onto a file: %v", err)
	}

	replaced, err := os.ReadFile(file)
	if err != nil || !strings.HasPrefix(string(replaced), "---\nname:") {
		t.Errorf("--force left the file as %q (%v), want the new note", replaced, err)
	}
}

// A path whose folder is missing, or is a file, can't be written either, and
// is refused before the session asks, naming the folder - not after it, as an
// open that fails (#424).
func TestAGuidedSessionWithNowhereToWriteIsRefusedFirst(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "notes.txt")

	err := os.WriteFile(file, []byte("mine\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	for name, path := range map[string]string{
		"a missing folder":      filepath.Join(dir, "missing", "kira.md"),
		"a file for its folder": filepath.Join(file, "kira.md"),
	} {
		var asked strings.Builder

		err := run([]string{cmdNew, flagService, other, flagOutput, path}, unread{t}, io.Discard, &asked)
		if err == nil || !strings.Contains(err.Error(), filepath.Dir(path)) || asked.Len() != 0 {
			t.Errorf("%s: said %v after asking %d bytes, want it refused at once naming %s",
				name, err, asked.Len(), filepath.Dir(path))
		}
	}
}
