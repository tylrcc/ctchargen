// Package render projects a generated character into the shapes a reader
// wants: JSON, a character sheet in the book's own style, and a transcript
// of the generation record.
//
// It also writes the readings: one in full (Reading), every one as an index
// (ReadingsIndex), those an id names (ReadingsNamed), and the section a
// transcript asked for with its readings closes on, before the line that says
// to keep the record. That is why it imports rules, where the readings are
// declared, and the only reason. A reading is written in this build's words,
// whatever build wrote the record naming it, since a record carries ids and
// never text. On a transcript, an id this build has no reading for is named as
// such rather than refused, since the rest of the record is still the
// record's; asked for by name (ReadingsNamed), it is refused (ErrNoReading).
//
// The domain types are the interface and JSON is a projection. Where the two
// disagree - a sum that must marshal flat, an array-backed profile that must
// marshal as six named keys - the codec here absorbs the difference and the
// domain type keeps its shape.
package render

import (
	"encoding/json"
	"fmt"

	"github.com/philoserf/ctchargen/internal/chargen"
	"github.com/philoserf/ctchargen/internal/traveller"
)

// JSONLine marshals a character onto one line, which is what batch writes:
// a stream of records, one per line, that a reader can take one at a time.
func JSONLine(character *chargen.Character) ([]byte, error) {
	projected, err := project(character)
	if err != nil {
		return nil, err
	}

	text, err := json.Marshal(projected)
	if err != nil {
		return nil, fmt.Errorf("marshalling the character: %w", err)
	}

	return append(text, '\n'), nil
}

// JSON marshals a character, indented, with a trailing newline.
func JSON(character *chargen.Character) ([]byte, error) {
	projected, err := project(character)
	if err != nil {
		return nil, err
	}

	text, err := json.MarshalIndent(projected, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling the character: %w", err)
	}

	return append(text, '\n'), nil
}

// recordShape is the number the record carries to say which shape it is.
//
// It counts the shape and nothing else: `build` names the writer and is
// absent from anything generated in-process, and `ruleset` names the source
// text. Neither says what the file looks like, and #74 settled that v1.0.0
// means a frozen and supported record - a promise needs something to promise
// about (#76, #62).
//
// It is 1 until v1.0.0, and counts from there. That is not the same as
// counting every change: the record is free to move before the freeze (#62),
// so a build that changes the shape before then leaves this alone, and two
// shapes that both say 1 are the price of that freedom rather than a defect
// in it (#96).
//
// From v1.0.0 the shape is a public contract and a build that changes it says
// so here. Nothing reads it yet, because until then there is only one value
// to read.
//
// The schema says `minimum` rather than a fixed number (#113). That is a
// statement about what a reader must accept, not about what this build writes:
// a reader holding the v1.0.0 schema has to validate a record from a later
// build rather than refuse it before looking at its contents. What holds this
// build to 1 is ctchargen.TestThisBuildWritesShapeOne.
//
// Implements: A record names the shape it is, so the v1 freeze has something to
// attach to [JSON conventions].
// Implements: This build writes record shape 1 [JSON conventions].
const recordShape = 1

type record struct {
	Record          int               `json:"record"`
	Name            string            `json:"name"`
	UPP             string            `json:"upp"`
	Characteristics map[string]int    `json:"characteristics"`
	Age             ageRecord         `json:"age"`
	Terms           int               `json:"terms"`
	Enlistment      enlistmentRecord  `json:"enlistment"`
	Service         string            `json:"service,omitempty"`
	Rank            int               `json:"rank,omitempty"`
	RankName        string            `json:"rankName,omitempty"`
	Skills          []skillRecord     `json:"skills"`
	Benefits        benefitsRecord    `json:"benefits"`
	Pension         int64             `json:"annualRetirementPay,omitempty"`
	Departure       *departureRecord  `json:"departure,omitempty"`
	Nobility        *nobilityRecord   `json:"nobility,omitempty"`
	Inputs          inputsRecord      `json:"inputs"`
	Errata          []string          `json:"errata"`
	Ruleset         string            `json:"ruleset"`
	Policy          *policyRecord     `json:"policy,omitempty"`
	Events          []json.RawMessage `json:"events"`
	Build           string            `json:"build,omitempty"`
}

type ageRecord struct {
	Years  int `json:"years"`
	Months int `json:"months,omitempty"`
}

type enlistmentRecord struct {
	How     string `json:"how"`
	Service string `json:"service,omitempty"`
}

type skillRecord struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type shipRecord struct {
	Kind         string `json:"kind"`
	Tons         int    `json:"tons"`
	Years        int    `json:"years"`
	PaymentYears int    `json:"paymentYears"`
}

type benefitsRecord struct {
	Cash          int64        `json:"cash"`
	Passages      []string     `json:"passages,omitempty"`
	TravellersAid bool         `json:"travellersAid,omitempty"`
	LostShips     int          `json:"scoutShipsLost,omitempty"`
	Weapons       []string     `json:"weapons,omitempty"`
	Ships         []shipRecord `json:"ships,omitempty"`
}

type departureRecord struct {
	How            string `json:"how"`
	Fatal          bool   `json:"fatal,omitempty"`
	Characteristic string `json:"characteristic,omitempty"`
}

type nobilityRecord struct {
	Eligible bool   `json:"eligible"`
	Title    string `json:"title"`
	Assumed  bool   `json:"assumed"`
}

type inputsRecord struct {
	Name    string `json:"name,omitempty"`
	Service string `json:"service,omitempty"`
}

// policyRecord is what an auto run's record says of the policy that answered
// it, and the whole of what tells an auto record from a guided one: present
// on every record the policy answered, whether or not it was asked anything,
// and absent from every record a person answered. Its strategies are here and
// not in `inputs` because they are the policy's, and a guided run has none.
type policyRecord struct {
	Version int    `json:"version"`
	Career  string `json:"career"`
	Skills  string `json:"skills"`
	Muster  string `json:"muster"`
}

func projectPolicy(answered *chargen.AutoPolicy) *policyRecord {
	if answered == nil {
		return nil
	}

	return &policyRecord{
		Version: answered.Version,
		Career:  answered.Policy.Career.String(),
		Skills:  answered.Policy.Skills.String(),
		Muster:  answered.Policy.Muster.String(),
	}
}

// answerer names who answered every choice in the record: the policy on an
// auto run, the player on a guided one.
func (r record) answerer() string {
	if r.Policy != nil {
		return "policy"
	}

	return "player"
}

func project(character *chargen.Character) (record, error) {
	out := record{
		Record:          recordShape,
		Name:            character.Name,
		UPP:             character.Profile.UPP(),
		Characteristics: map[string]int{},
		Age:             ageRecord{Years: character.Age.Years(), Months: character.Age.Months()},
		Terms:           character.Terms,
		Enlistment:      foldEnlistment(character.Enlistment),
		Rank:            int(character.Rank),
		RankName:        character.RankName,
		Skills:          make([]skillRecord, 0, len(character.Skills)),
		Benefits:        projectBenefits(character.Benefits, character.DuplicateShips),
		Pension:         int64(character.Pension),
		Inputs:          projectInputs(character.Inputs),
		Errata:          make([]string, 0, len(character.Errata)),
		Ruleset:         character.Ruleset,
		Policy:          projectPolicy(character.Policy),
		Events:          nil,
		Build:           character.Build,
	}

	events, err := projectEvents(character.Events)
	if err != nil {
		return record{}, fmt.Errorf("projecting the generation record: %w", err)
	}

	out.Events = events

	for _, characteristic := range traveller.Characteristics {
		out.Characteristics[characteristic.String()] = character.Profile[characteristic]
	}

	if service, served := character.ServedIn(); served {
		out.Service = service.String()
	}

	for _, skill := range character.Skills {
		out.Skills = append(out.Skills, skillRecord{Name: string(skill.Name), Level: skill.Level})
	}

	for _, erratum := range character.Errata {
		out.Errata = append(out.Errata, erratum.String())
	}

	if character.Departure != nil {
		departure := foldDeparture(character.Departure)

		out.Departure = &departure
	}

	if character.Nobility.Eligible {
		out.Nobility = &nobilityRecord{
			Eligible: true, Title: character.Nobility.Title.String(), Assumed: character.Nobility.Assumed,
		}
	}

	return out, nil
}

func projectInputs(in chargen.Inputs) inputsRecord {
	out := inputsRecord{Name: in.Name}
	if service, forced := in.Forced.Service(); forced {
		out.Service = service.String()
	}

	return out
}

func projectBenefits(b chargen.Benefits, lost int) benefitsRecord {
	out := benefitsRecord{Cash: int64(b.Cash), TravellersAid: b.TravellersAid, LostShips: lost}

	for _, passage := range b.Passages {
		out.Passages = append(out.Passages, passage.String())
	}

	for _, weapon := range b.Weapons {
		out.Weapons = append(out.Weapons, string(weapon))
	}

	for _, ship := range b.Ships {
		out.Ships = append(out.Ships, shipRecord{
			Kind: ship.Kind.String(), Tons: ship.Tons,
			Years: ship.Years, PaymentYears: ship.PaymentYears,
		})
	}

	return out
}
