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
	OnEntering []string
	Terms      []termView
}

// termView is one term of service. The term a survival throw ended carries
// its number and nothing else.
type termView struct {
	Term      int
	Raised    []raiseView
	Gained    []string
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
		OnEntering: gainWords(history.OnEntering),
		Terms:      make([]termView, 0, len(history.Terms)),
	}

	for _, term := range history.Terms {
		served := termView{Term: int(term.Term), Gained: gainWords(term.Gained)}

		for _, raise := range term.Raised {
			served.Raised = append(served.Raised, raiseView{
				How: raise.How.String(), Rank: raise.Rank, Grants: gainWords(raise.Grants),
			})
		}

		for _, change := range term.Aged {
			served.Aged = append(served.Aged, changeWords(change))
		}

		for _, characteristic := range term.Recovered {
			served.Recovered = append(served.Recovered, characteristic.String())
		}

		out.Terms = append(out.Terms, served)
	}

	return out
}

// gainWords is each gain in the order received: a skill at the level it
// reached, a characteristic change as the delta the page's results are
// written in.
func gainWords(gains []traveller.Gain) []string {
	out := make([]string, 0, len(gains))

	for _, gain := range gains {
		var words gainWordsCodec

		// A fold over a codec cannot fail: every case only assigns.
		_ = gain.Fold(&words)

		out = append(out, words.out)
	}

	return out
}

// gainWordsCodec folds a gain into its words. Adding a kind of gain stops
// this compiling until it has some.
type gainWordsCodec struct{ out string }

func (g *gainWordsCodec) Skill(skill traveller.Skill) error {
	g.out = skillText(string(skill.Name), skill.Level)

	return nil
}

func (g *gainWordsCodec) Change(change traveller.Change) error {
	g.out = changeWords(change)

	return nil
}

// changeWords is a change as the table gave it, and, where a floor or a
// ceiling held the characteristic short of it, where it was held (#409).
//
// Implements: A clamped change says the table's change and where it was held
// [Sheet].
func changeWords(change traveller.Change) string {
	words := fmt.Sprintf("%v %+d", change.Characteristic, change.Delta)
	if change.To-change.From != change.Delta {
		words += fmt.Sprintf(" (held at %d)", change.To)
	}

	return words
}
