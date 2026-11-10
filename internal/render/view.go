package render

import (
	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// view is a character as the note reads it: the domain values turned into
// the words the sheet and the generation record print. It is built once, by
// viewOf, and nothing else builds one, so the sheet and the record cannot
// read the same character two ways.
type view struct {
	Name       string
	UPP        string
	Profile    traveller.Profile
	Age        ageView
	Terms      int
	Enlistment enlistmentView
	Service    string
	RankName   string
	Skills     []skillView
	Benefits   benefitsView
	Pension    int64
	History    historyView
	Departure  departureView
	Nobility   *nobilityView
	Attempt    attemptView
	ByPolicy   bool
	Events     []entry
	Build      string
}

type ageView struct {
	Years  int
	Months int
}

type enlistmentView struct {
	How     string
	Service string
}

type skillView struct {
	Name  string
	Level int
}

type shipView struct {
	// Item is the ship as one possession: "Free Trader", "Scout ship".
	Item         string
	Kind         string
	Tons         int
	Years        int
	PaymentYears int
}

type benefitsView struct {
	Cash          int64
	Passages      []string
	TravellersAid bool
	Weapons       []string
	Ships         []shipView
}

type departureView struct {
	How            string
	Fatal          bool
	Characteristic string
}

type nobilityView struct {
	Title   string
	Assumed bool
}

// attemptView is the service enlistment was attempted in, as chargen reads it
// from the record, and whether the policy named it (#397). Only a served
// character's is projected (#454); a civilian's is the zero value, never read.
type attemptView struct {
	Service  string
	ByPolicy bool
}

// answerer names who answered every choice: the policy on an auto run, the
// player on a guided one. The record says it on each choice's line.
//
// Implements: A note says who made each choice [Generation record].
func (r view) answerer() string {
	if r.ByPolicy {
		return "policy"
	}

	return "player"
}

// viewOf is the character as the note reads it.
func viewOf(character *chargen.Character) view {
	profile, skills, age := character.Profile(), character.Skills(), character.Age()

	out := view{
		Name:       character.Inputs.Name,
		UPP:        profile.UPP(),
		Profile:    profile,
		Age:        ageView{Years: age.Years(), Months: age.Months()},
		Terms:      character.Terms(),
		Enlistment: foldEnlistment(character.Enlistment()),
		RankName:   character.RankName(),
		Skills:     make([]skillView, 0, len(skills)),
		Benefits:   projectBenefits(character.Benefits()),
		Pension:    int64(character.Pension()),
		ByPolicy:   character.Policy != nil,
		Events:     entries(character.Events),
		Build:      character.Build,
	}

	if service, served := character.ServedIn(); served {
		out.Service = service.String()
		out.History = projectHistory(character.History())

		// Generation gives every served character a departure, and the view
		// reads one: a nil one is a generation defect, and panics here, as a
		// nil enlistment does (#393, #445). A civilian departed nothing: his
		// departure is the zero value, which writes no property and reads as
		// alive.
		out.Departure = foldDeparture(character.Departure())
		out.Attempt = projectAttempt(character)
	}

	for _, skill := range skills {
		out.Skills = append(out.Skills, skillView{Name: string(skill.Name), Level: skill.Level})
	}

	if nobility := character.Nobility(); nobility.Eligible {
		out.Nobility = &nobilityView{Title: nobility.Title.String(), Assumed: nobility.Assumed}
	}

	return out
}

// projectAttempt is the service a served character attempted. Every
// generation makes the enlistment throw, so one who served always has one; a
// served character without is a generation defect, and fails here rather than
// reading as one who asked for his service and got it (#454), as #393 and
// #445 decided for the enlistment and the departure.
func projectAttempt(character *chargen.Character) attemptView {
	attempt, attempted := character.Attempted()
	if !attempted {
		panic("render: a served character with no enlistment throw in its record")
	}

	return attemptView{
		Service: attempt.Service.String(), ByPolicy: attempt.ByPolicy,
	}
}

func projectBenefits(b chargen.Benefits) benefitsView {
	out := benefitsView{Cash: int64(b.Cash), TravellersAid: b.TravellersAid}

	for _, passage := range b.Passages {
		out.Passages = append(out.Passages, passage.String())
	}

	for _, weapon := range b.Weapons {
		out.Weapons = append(out.Weapons, string(weapon))
	}

	for _, ship := range b.Ships {
		out.Ships = append(out.Ships, shipView{
			Item: shipItem(ship.Kind), Kind: ship.Kind.String(), Tons: ship.Tons,
			Years: ship.Years, PaymentYears: ship.PaymentYears,
		})
	}

	return out
}

// shipItem is a ship as the possessions property names it, without the
// hull type the sheet gives (#475).
func shipItem(kind traveller.ShipKind) string {
	switch kind {
	case traveller.ScoutShip:
		return "Scout ship"
	case traveller.FreeTrader:
		return "Free Trader"
	}

	return kind.String()
}

// entries is every event, flat, in the order the run recorded them.
func entries(events []traveller.Event) []entry {
	out := make([]entry, 0, len(events))

	for _, event := range events {
		out = append(out, entryOf(event))
	}

	return out
}
