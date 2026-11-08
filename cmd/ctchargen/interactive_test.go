package main

import (
	"encoding/json"
	"errors"
	"io"
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

	out := mustRunTogether(t, []string{cmdNew, flagService, other, flagSheet}, answers("1", 200),
		withDice(otherWalked.roller(t)))

	shown := out

	for _, want := range []string{
		// The procedure, shown as it runs. The headings carry their
		// sequence number, which is what stops the numbers appearing to
		// skip past them.
		"## 1. characteristics (p. 4)",
		"Strength: rolled",
		"## 18. term 1 (pp. 5-7)",
		"survival: rolled",
		// A question, with the alternatives the engine offered.
		"Which skills table? It is designated before the die (p. 11)",
		"1) Personal Development Table",
		// And the sheet at the end.
		"UPP ",
	} {
		if !strings.Contains(shown, want) {
			t.Errorf("the run never showed %q", want)
		}
	}

	// A choice is echoed back, and in one line.
	//
	// It used to be shown as nothing at all, on the reasoning that the
	// player answered it a line earlier and repeating it pushes the useful
	// lines off the screen. That reasoning still holds against the
	// transcript's own form - "SubmitToDraft: player chose yes from yes, no"
	// - which names the point, the decider and every alternative he can
	// still see above him. What it does not survive is the cost: a choice
	// holds a sequence number, and showing nothing left the numbers with
	// holes in them that read as lost events.
	//
	// So the echo is the number and the answer, and nothing else.
	if strings.Contains(shown, "player chose") {
		t.Error("the live view used the transcript's form, which repeats what he can see")
	}

	// Named against the answer, not against the words around it. An echo
	// reading "you chose " with nothing after it keeps the numbering whole
	// and says nothing - which is what happens the moment liveCodec stops
	// carrying the field, a drift the fold cannot catch because the case
	// still compiles.
	if !strings.Contains(shown, "you chose Personal Development Table") {
		t.Error("the echo does not name what was chosen")
	}
}

// An answer it cannot read is asked again rather than guessed at.
//
// Holds: An unreadable answer is asked again, never guessed at.
func TestInteractiveReAsksWhatItCannotRead(t *testing.T) {
	t.Parallel()

	out := mustRunTogether(t, []string{cmdNew, flagService, other, flagSheet}, unusableFirst(),
		withDice(otherWalked.roller(t)))

	shown := out

	if strings.Count(shown, "answer with a number from 1 to") != 3 {
		t.Errorf("three unusable answers drew %d re-prompts, want 3",
			strings.Count(shown, "answer with a number from 1 to"))
	}

	// The complaint names what was typed, so a reader can see which of his
	// answers was refused rather than inferring it.
	for _, given := range []string{`"banana"`, `"0"`, `"99"`} {
		if !strings.Contains(shown, given+" is not one of them") {
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

	out := mustRunTogether(t, []string{cmdNew, flagSheet}, unusableFirst())

	shown := out

	const question = "Which service will you try to enlist in?"

	if asked := strings.Count(shown, question); asked != 1 {
		t.Errorf("three bad answers printed the menu %d times, want 1", asked)
	}

	if refused := strings.Count(shown, "is not one of them"); refused != 3 {
		t.Errorf("three bad answers drew %d complaints, want 3", refused)
	}
}

// The numbers a player watches do not skip.
//
// They appeared to - 17, 18, then 20 - because the headings and the questions
// hold sequence numbers and rendered unnumbered. Nothing was ever missing, but
// a gap in a transcript whose whole purpose is auditability costs a reader
// trust he should not have to spend.
func TestTheNumbersAPlayerWatchesDoNotSkip(t *testing.T) {
	t.Parallel()

	out := mustRunTogether(t, []string{cmdNew, flagService, other, flagSheet}, answers("1", 200),
		withDice(otherWalked.roller(t)))

	numbered := regexp.MustCompile(`(?m)^\s*(?:## )?(\d+)\. `)

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

// The two Advanced Education tables are told apart at a glance.
//
// They are identical up to a parenthesis, which is honest and easy to pick
// wrong at speed. What separates them belongs at the front, because the eye
// scans down the left of a numbered list.
func TestTheTwoAdvancedEducationTablesReadDifferently(t *testing.T) {
	t.Parallel()

	// The Navy walk reaches a term offering all four tables, which needs
	// the education 8+ gate open.
	out := mustRunTogether(t, []string{cmdNew, flagService, navy, flagSheet}, answers("1", 200),
		withDice(navyWalked.roller(t)))

	shown := out
	if !strings.Contains(shown, "Advanced Education Table\n") {
		t.Fatalf("the run never offered the skills tables:\n%s", shown)
	}

	if !strings.Contains(shown, "Education 8+ table (the second Advanced Education table)") {
		t.Errorf("the two Advanced Education entries are not told apart:\n%s", shown)
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

	asked := mustRunTogether(t, []string{cmdNew, flagService, other, flagSheet}, answers("1", 200),
		withDice(otherWalked.roller(t)))

	automatic := mustRunTogether(t, []string{cmdNew, flagAuto, flagService, other, flagSheet}, nil,
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

	out := mustRunTogether(t, []string{cmdNew, flagService, navy, flagSheet}, answers("1", 300),
		withDice(draftedWalked.roller(t)))

	shown := out

	for _, want := range []string{
		"Rejected. Submit to the draft? (p. 5)",
		"Attempt a commission this term? (p. 6)",
		"Attempt a promotion this term? (p. 6)",
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

// The gambling modifier is offered only to someone who has the expertise, so
// reaching it takes a walk that draws the skill: the gambler, taking the
// Service Skills Table every time.
//
// Holds: The gambling modifier is offered only to one who has the expertise.
func TestInteractiveOffersTheGamblingModifier(t *testing.T) {
	t.Parallel()

	out := mustRunTogether(t, []string{cmdNew, flagService, other, flagSheet}, answers("2", 300),
		withDice(gamblerWalked.roller(t)))

	if !strings.Contains(out, "Your gambling expertise allows +1 on this table") {
		t.Error("a character with Gambling was never offered the modifier p. 9 gives him")
	}
}

// brokenPipe fails every write, as a terminal does when the reader on the
// other end goes away.
type brokenPipe struct{}

func (brokenPipe) Write([]byte) (int, error) { return 0, errClosedPipe }

var errClosedPipe = errors.New("the other end closed")

// A player nobody can be shown anything is not asked anything either. The
// failure is remembered and reported at the next question, rather than being
// swallowed once per line for the length of a career.
//
// Holds: A player nobody can be shown anything is not asked.
func TestInteractiveStopsWhenItCannotBeRead(t *testing.T) {
	t.Parallel()

	// Partly: the run stops at the first question nobody can be shown.
	err := run([]string{cmdNew, flagService, other, flagSheet},
		answers("1", 300), io.Discard, brokenPipe{}, withDice(otherWalked.partly(t)))

	switch {
	case err == nil:
		t.Fatal("the run finished while nobody could see the questions")
	case !errors.Is(err, errClosedPipe):
		t.Errorf("error %q does not carry the write failure that caused it", err)
	}
}

// The questions do not go where the record goes.
//
// A guided run still writes JSON to stdout, so `ctchargen new | jq`
// has to receive a record and not a conversation. The two channels are
// separate arguments for that reason, and this is what holds them apart.
//
// Holds: The questions go to stderr, not into the record.
// Holds: A guided run's choices are recorded as the player's, not the policy's.
func TestTheQuestionsStayOutOfTheRecord(t *testing.T) {
	t.Parallel()

	record, asked := mustRun(t, []string{cmdNew, flagService, other}, answers("1", 300),
		withDice(otherWalked.roller(t)))

	var written map[string]any

	invalid := json.Unmarshal([]byte(record), &written)
	if invalid != nil {
		t.Fatalf("the data channel does not hold a record on its own: %v", invalid)
	}

	if !strings.Contains(asked, "Which skills table?") {
		t.Error("the questions did not go to the channel meant for them")
	}

	// And the record names who answered. A guided run's choices were the
	// player's, and a record that calls them the policy's misdescribes how
	// the character was made - the one thing the log exists to say.
	events, _ := written["events"].([]any)

	choices := 0

	for _, event := range events {
		fields, _ := event.(map[string]any)
		if fields["kind"] != "choice" {
			continue
		}

		choices++

		if fields["by"] != "player" {
			t.Errorf("choice %v says %v answered it, want the player",
				fields["point"], fields["by"])
		}
	}

	if choices == 0 {
		t.Fatal("the record holds no choice events, so it asserts nothing about who answered")
	}
}

// A session cut short says what was answered and that nothing was written.
//
// The half-built character is lost: it is not a record and does not match the
// schema, and there is no way back to the question it stopped on (ADR 0001).
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

// armed is what a record says of the weapon benefits it took: the weapons,
// whether Dagger-1 is among the skills, and the last weapon-benefit choice
// in the record's own wording.
type armed struct {
	weapons     []string
	daggerSkill bool
	chosen      string
}

func armedBy(t *testing.T, record string) armed {
	t.Helper()

	var got struct {
		Skills []struct {
			Name  string `json:"name"`
			Level int    `json:"level"`
		} `json:"skills"`
		Benefits struct {
			Weapons []string `json:"weapons"`
		} `json:"benefits"`
		Events []struct {
			Point  string `json:"point"`
			Chosen string `json:"chosen"`
		} `json:"events"`
	}

	err := json.Unmarshal([]byte(record), &got)
	if err != nil {
		t.Fatalf("reading the record: %v", err)
	}

	found := armed{weapons: got.Benefits.Weapons}

	for _, skill := range got.Skills {
		if skill.Name == dagger && skill.Level == 1 {
			found.daggerSkill = true
		}
	}

	for _, event := range got.Events {
		if event.Point == "MusterWeapon" {
			found.chosen = event.Chosen
		}
	}

	return found
}
