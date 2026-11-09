package rules //nolint:testpackage // the JSON lift it keeps is unexported

// The JSON lift of skills.json, kept unchanged from before the move to
// transcription.go so that the values the transcription lifts can be proven
// equal to what the JSON lifted. It goes, with skills.json, when the last
// table has moved (#338).

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/philoserf/ctchargen/internal/traveller"
)

type wireSkills struct {
	Services    []string              `json:"services"`
	Tables      map[string][][]string `json:"tables"`
	Eligibility struct {
		InitialTerm       int `json:"initialTerm"`
		PerSubsequentTerm int `json:"perSubsequentTerm"`
		OnCommission      int `json:"onCommission"`
		OnPromotion       int `json:"onPromotion"`
	} `json:"eligibility"`
	EducationGate struct {
		Table          string `json:"table"`
		Characteristic string `json:"characteristic"`
		Threshold      string `json:"threshold"`
	} `json:"educationGate"`
	Normalizations map[string]string `json:"normalizations"`
}

// skillTablesByNameJSON reads the four tables' names, in sorted order so a file
// with two unknown names refuses the same one every run.
func skillTablesByNameJSON(wire map[string][][]string) (map[traveller.SkillTable][][]string, error) {
	tables := withoutNotes(wire)
	if len(tables) != len(traveller.SkillTables) {
		return nil, fmt.Errorf("%w: acquired skills: %d tables, want %d", errMalformed,
			len(tables), len(traveller.SkillTables))
	}

	byTable := make(map[traveller.SkillTable][][]string, len(tables))

	for _, name := range slices.Sorted(maps.Keys(tables)) {
		table, err := parseSkillTable(name)
		if err != nil {
			return nil, fmt.Errorf("%w: acquired skills: %w", errMalformed, err)
		}

		byTable[table] = tables[name]
	}

	return byTable, nil
}

// liftSkillTablesJSON lifts every cell of the four tables, in the order p. 11
// prints them (traveller.SkillTables) and never a map's, so a file broken two
// ways names the first fault every run.
func (r *Rules) liftSkillTablesJSON(services []string, byTable map[traveller.SkillTable][][]string) error {
	for _, table := range traveller.SkillTables {
		rows := byTable[table]

		if len(rows) != Faces {
			return fmt.Errorf("%w: acquired skills, %v: %d rows, want %d", errMalformed, table, len(rows), Faces)
		}

		for die, row := range rows {
			what := fmt.Sprintf("acquired skills, %v, row %d", table, die+1)

			err := eachService(services, row, what, func(s traveller.ServiceName, cell string) error {
				result, err := parseTableResult(cell, r.normalize)
				if err != nil {
					return err
				}

				r.services[s].Skills[table][die] = result

				return nil
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// liftSkillsJSON lifts skills.json: the four acquired skills tables, the
// eligibility and the education gate. Names are read in sorted order and the
// tables in the order p. 11 prints them, so a file broken two ways reports the
// same fault every run.
func (r *Rules) liftSkillsJSON(wire wireSkills) error {
	byTable, err := skillTablesByNameJSON(wire.Tables)
	if err != nil {
		return err
	}

	err = r.liftSkillTablesJSON(wire.Services, byTable)
	if err != nil {
		return err
	}

	r.eligibility = Eligibility{
		InitialTerm:       wire.Eligibility.InitialTerm,
		PerSubsequentTerm: wire.Eligibility.PerSubsequentTerm,
		OnCommission:      wire.Eligibility.OnCommission,
		OnPromotion:       wire.Eligibility.OnPromotion,
	}

	table, err := parseSkillTable(wire.EducationGate.Table)
	if err != nil {
		return fmt.Errorf("%w: education gate: %w", errMalformed, err)
	}

	characteristic, err := parseCharacteristic(wire.EducationGate.Characteristic)
	if err != nil {
		return fmt.Errorf("%w: education gate: %w", errMalformed, err)
	}

	threshold, err := traveller.ParseTarget(wire.EducationGate.Threshold)
	if err != nil {
		return fmt.Errorf("%w: education gate: %w", errMalformed, err)
	}

	r.education = Gate{Table: table, Characteristic: characteristic, Threshold: threshold}

	return nil
}

// The skills transcription lifts to exactly what skills.json lifted: every
// cell of the four tables, the eligibility, the education gate and the
// normalizations. Both lifts run on a fresh Rules and are compared whole.
func TestTheSkillsTranscriptionLiftsAsTheJSONDid(t *testing.T) {
	t.Parallel()

	fromJSON := &Rules{normalize: withoutNotes(mustRead[wireSkills](t, "skills.json").Normalizations)}

	err := fromJSON.liftSkillsJSON(mustRead[wireSkills](t, "skills.json"))
	if err != nil {
		t.Fatalf("lifting skills.json: %v", err)
	}

	fromGo := &Rules{normalize: maps.Clone(transcribedSkills.normalizations)}

	err = fromGo.liftSkills(transcribedSkills)
	if err != nil {
		t.Fatalf("lifting the transcription: %v", err)
	}

	if !maps.Equal(fromJSON.normalize, fromGo.normalize) {
		t.Errorf("normalizations: JSON %v, transcription %v", fromJSON.normalize, fromGo.normalize)
	}

	for _, s := range traveller.ServiceNames {
		if fromJSON.services[s].Skills != fromGo.services[s].Skills {
			t.Errorf("%v: JSON lifted %v, the transcription %v", s, fromJSON.services[s].Skills, fromGo.services[s].Skills)
		}
	}

	if fromJSON.eligibility != fromGo.eligibility {
		t.Errorf("eligibility: JSON %+v, transcription %+v", fromJSON.eligibility, fromGo.eligibility)
	}

	if fromJSON.education != fromGo.education {
		t.Errorf("education gate: JSON %+v, transcription %+v", fromJSON.education, fromGo.education)
	}
}
