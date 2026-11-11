package render

import (
	"fmt"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// historyView is the service history as the sheet tells it, a term at a
// time: what entering the service granted, and what each term conferred, gave
// and took (#401). Each gain is already the words the sheet prints.
type historyView struct {
	OnEntering    []string
	Terms         []termView
	RetiredDenied bool
	MusteredOut   []string
}

// termView is one term of service. The term a survival throw ended carries
// its number and nothing else.
type termView struct {
	Term      int
	Raised    []raiseView
	Gained    []string
	Retained  bool
	Aged      []string
	Recovered []string
}

// raiseView is a commission or a promotion, and what the Rank and Service
// Skills box granted at the rank it conferred (p. 23).
type raiseView struct {
	How    string
	Rank   string
	Grants []string
}

// projectHistory is the service history as the sheet tells it.
func projectHistory(history chargen.ServiceHistory) historyView {
	out := historyView{
		OnEntering:    movedWords(history.OnEntering),
		Terms:         make([]termView, 0, len(history.Terms)),
		RetiredDenied: history.RetiredDenied,
		MusteredOut:   movedWords(history.MusteredOut),
	}

	for _, term := range history.Terms {
		served := termView{Term: int(term.Term), Gained: movedWords(term.Gained), Retained: term.Retained}

		for _, raise := range term.Raised {
			served.Raised = append(served.Raised, raiseView{
				How: raise.How.String(), Rank: raise.Rank, Grants: movedWords(raise.Grants),
			})
		}

		aged := make([]move, 0, len(term.Aged))
		for _, change := range term.Aged {
			aged = append(aged, changeMove(change))
		}

		served.Aged = told(aged)

		for _, recovery := range term.Recovered {
			served.Recovered = append(served.Recovered, recoveryWords(recovery))
		}

		out.Terms = append(out.Terms, served)
	}

	return out
}

// move is one gain or aging reduction as the sheet tells it: what moved, the
// value it moved from and to, and, where a floor or a ceiling held the
// characteristic short of the table's change, the value it was held at.
type move struct {
	name     string
	from, to int
	held     bool
	heldAt   int
}

// movedWords is a list of gains in the words the sheet prints.
func movedWords(gains []chargen.Moved) []string {
	moves := make([]move, 0, len(gains))

	for _, gain := range gains {
		words := moveFold{from: gain.From}

		// This fold cannot fail: every case only assigns.
		_ = gain.Gain.Fold(&words)

		moves = append(moves, words.out)
	}

	return told(moves)
}

// moveFold folds a gain into its move. Adding a kind of gain stops this
// compiling until it has one.
type moveFold struct {
	from int
	out  move
}

func (m *moveFold) Skill(skill traveller.Skill) error {
	m.out = move{name: string(skill.Name), from: m.from, to: skill.Level}

	return nil
}

func (m *moveFold) Change(change traveller.Change) error {
	m.out = changeMove(change)

	return nil
}

// changeMove is a characteristic change as a move, held where the table's
// change was not the distance it moved (#409).
func changeMove(change traveller.Change) move {
	return move{
		name: change.Characteristic.String(), from: change.From, to: change.To,
		held: change.To-change.From != change.Delta, heldAt: change.To,
	}
}

// told is a list of moves as the sheet tells them: one entry for each skill
// or characteristic, in the order it first moved, from the value before the
// first move to the value after the last (#562). That replaces #409's change
// as the table gave it; the generation record still says each one. It says
// where a floor or a ceiling held the characteristic only where one did, at
// the value the last held move stopped at. A skill and a characteristic
// never share a name.
//
// Implements: The service record tells each skill or characteristic once in
// a list of gains [Sheet].
//
// Implements: The service record says a gain's value before and after, and
// where a floor or a ceiling held it [Sheet].
func told(moves []move) []string {
	var order []string

	merged := map[string]move{}

	for _, next := range moves {
		was, seen := merged[next.name]
		if !seen {
			order = append(order, next.name)
			merged[next.name] = next

			continue
		}

		was.to = next.to

		if next.held {
			was.held, was.heldAt = true, next.heldAt
		}

		merged[next.name] = was
	}

	out := make([]string, 0, len(order))

	for _, name := range order {
		m := merged[name]

		words := fmt.Sprintf("%s %d→%d", m.name, m.from, m.to)
		if m.held {
			words += fmt.Sprintf(" (held at %d)", m.heldAt)
		}

		out = append(out, words)
	}

	return out
}

// recoveryWords is a crisis survived: the characteristic, the value it came
// back to, and the months the recovery added to his age (#561).
//
// Implements: A medical crisis recovery on the service record says its value
// and months [Sheet].
func recoveryWords(recovery chargen.Recovery) string {
	months := "months"
	if recovery.Months == 1 {
		months = "month"
	}

	return fmt.Sprintf("%v recovered from a medical crisis to %d (%d %s older)",
		recovery.Characteristic, recovery.To, recovery.Months, months)
}
