package render

import (
	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// historyRecord is the service history, a term at a time: what entering the
// service granted, and what each term conferred, gave and took (#401). It is
// absent for a civilian, who served none.
type historyRecord struct {
	OnEntering []gainRecord `json:"onEntering,omitempty"`
	Terms      []termRecord `json:"terms"`
}

// termRecord is one term of service. The term a survival throw ended carries
// its number and nothing else.
type termRecord struct {
	Term      int            `json:"term"`
	Raised    []raiseRecord  `json:"raised,omitempty"`
	Gained    []gainRecord   `json:"gained,omitempty"`
	Aged      []changeRecord `json:"aged,omitempty"`
	Recovered []string       `json:"recovered,omitempty"`
}

// raiseRecord is a commission or a promotion, and what the Rank and Service
// Skills box granted at the rank it conferred (p. 23).
type raiseRecord struct {
	How    string       `json:"how"`
	Rank   string       `json:"rank"`
	Grants []gainRecord `json:"grants,omitempty"`
}

// gainRecord is one gain, written as the one key it is: a skill at the level
// it reached, or a characteristic changed.
type gainRecord struct {
	Skill  *skillRecord  `json:"skill,omitempty"`
	Change *changeRecord `json:"change,omitempty"`
}

type changeRecord struct {
	Characteristic string `json:"characteristic"`
	From           int    `json:"from"`
	To             int    `json:"to"`
}

// projectHistory writes the service history. Its terms are an array even
// where the first one killed him, so a reader always finds the term he died
// in.
func projectHistory(history chargen.ServiceHistory) *historyRecord {
	out := &historyRecord{
		OnEntering: projectGains(history.OnEntering),
		Terms:      make([]termRecord, 0, len(history.Terms)),
	}

	for _, term := range history.Terms {
		served := termRecord{
			Term:   int(term.Term),
			Gained: projectGains(term.Gained),
			Aged:   projectChanges(term.Aged),
		}

		for _, raise := range term.Raised {
			served.Raised = append(served.Raised, raiseRecord{
				How: raise.How.String(), Rank: raise.Rank, Grants: projectGains(raise.Grants),
			})
		}

		for _, characteristic := range term.Recovered {
			served.Recovered = append(served.Recovered, characteristic.String())
		}

		out.Terms = append(out.Terms, served)
	}

	return out
}

func projectGains(gains []chargen.Gain) []gainRecord {
	out := make([]gainRecord, 0, len(gains))

	for _, gain := range gains {
		var codec gainCodec

		// A fold over a codec cannot fail: every case only assigns.
		_ = gain.Fold(&codec)

		out = append(out, codec.out)
	}

	return out
}

func projectChanges(changes []chargen.Change) []changeRecord {
	out := make([]changeRecord, 0, len(changes))

	for _, change := range changes {
		out = append(out, projectChange(change))
	}

	return out
}

func projectChange(change chargen.Change) changeRecord {
	return changeRecord{Characteristic: change.Characteristic.String(), From: change.From, To: change.To}
}

// gainCodec folds a gain into the one key it is written as. Adding a kind of
// gain stops this compiling until it is written.
type gainCodec struct{ out gainRecord }

func (g *gainCodec) Skill(skill traveller.Skill) error {
	g.out.Skill = &skillRecord{Name: string(skill.Name), Level: skill.Level}

	return nil
}

func (g *gainCodec) Change(change chargen.Change) error {
	projected := projectChange(change)

	g.out.Change = &projected

	return nil
}
