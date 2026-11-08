package rules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// The lift's parsers. Every one of them checks a printed string against the
// domain's own closed alphabet rather than against a second list kept here:
// a name that does not name a value of the type is a defect in the data
// file, and the lift is where it must be caught.

// named indexes an alphabet by the name its values give themselves, so that
// a data file's spelling is checked against the type and nothing else.
func named[T fmt.Stringer](values []T) map[string]T {
	index := make(map[string]T, len(values))
	for _, v := range values {
		index[v.String()] = v
	}

	return index
}

var (
	characteristics  = named(traveller.Characteristics[:])
	serviceNames     = named(traveller.ServiceNames[:])
	skillTables      = named(traveller.SkillTables[:])
	weaponCategories = named(traveller.WeaponCategories[:])
	passageClasses   = named(traveller.PassageClasses[:])
	shipKinds        = named(traveller.ShipKinds[:])
	titles           = named(traveller.Titles[:])
)

func lookup[T any](index map[string]T, name, kind string) (T, error) {
	value, ok := index[name]
	if !ok {
		var zero T

		return zero, fmt.Errorf("%w: %q is not a %s", errMalformed, name, kind)
	}

	return value, nil
}

func parseCharacteristic(name string) (traveller.Characteristic, error) {
	return lookup(characteristics, name, "characteristic")
}

func parseService(name string) (traveller.ServiceName, error) {
	return lookup(serviceNames, name, "service")
}

func parseSkillTable(name string) (traveller.SkillTable, error) {
	return lookup(skillTables, name, "skills table")
}

func parseTitle(name string) (traveller.Title, error) {
	return lookup(titles, name, "title")
}

// alteration matches a characteristic alteration as the tables print one:
// "+1 Strength", "-1 Social Standing", "+2 Education".
//
// The sign is whatever traveller.Sign reads as one, so a cell accepts the
// same minus glyphs a target does: someone transcribing p. 11 will type what
// he sees, and the two minuses are indistinguishable on screen. A cell whose
// first glyph is no sign is no alteration. A signed cell that does not parse
// here is refused outright rather than accumulating as a skill of that name.
var alteration = regexp.MustCompile(`^(.)(\d+) (.+)$`)

// parseAlteration reads a characteristic alteration, reporting whether the
// cell is one at all.
func parseAlteration(cell string) (traveller.Characteristic, int, bool, error) {
	match := alteration.FindStringSubmatch(cell)
	if match == nil {
		return 0, 0, false, nil
	}

	glyph, _ := utf8.DecodeRuneInString(match[1])

	sign, ok := traveller.Sign(glyph)
	if !ok {
		return 0, 0, false, nil
	}

	size, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, false, fmt.Errorf("%w: alteration %q: %w", errMalformed, cell, err)
	}

	size *= sign

	characteristic, err := parseCharacteristic(match[3])
	if err != nil {
		return 0, 0, false, fmt.Errorf("%w: alteration %q: %w", errMalformed, cell, err)
	}

	return characteristic, size, true, nil
}

// parseDM reads one die modifier as the Prior Service Table prints it: an
// amount, and the characteristic and threshold that earn it. The data file
// spells the condition the way p. 10 does, "Intelligence 8+".
func parseDM(amount int, condition string) (DM, error) {
	name, threshold, ok := strings.CutLast(condition, " ")
	if !ok {
		return DM{}, fmt.Errorf("%w: die modifier %q: want a characteristic and a threshold", errMalformed, condition)
	}

	characteristic, err := parseCharacteristic(name)
	if err != nil {
		return DM{}, fmt.Errorf("%w: die modifier %q: %w", errMalformed, condition, err)
	}

	target, err := traveller.ParseTarget(threshold)
	if err != nil {
		return DM{}, fmt.Errorf("%w: die modifier %q: %w", errMalformed, condition, err)
	}

	return DM{Amount: amount, Characteristic: characteristic, Threshold: target}, nil
}

// startsWithSign reports whether a cell opens with the sign of an
// alteration. P. 3 puts the sign of a modifier before its number, so a cell
// beginning with one is an alteration or it is a defect — never a skill.
func startsWithSign(cell string) bool {
	glyph, _ := utf8.DecodeRuneInString(cell)
	_, ok := traveller.Sign(glyph)

	return ok
}

// parseTableResult reads one cell of the Acquired Skills Table (p. 11).
// P. 12 names exactly three kinds a cell can be. Only the third is open —
// SkillName does not close at compile time — so the guard that matters here
// is the one on the other two: a cell that opens with a sign and did not
// parse as an alteration is refused rather than accumulating as a skill of
// that name. Without it "+x Social Standing" lifts as a skill whose name is
// the whole cell. A cell opening with an en dash is refused too, ahead of
// that guard and with an error naming the en dash, since "–1 Social
// Standing" is indistinguishable on screen from the real cell.
func parseTableResult(cell string, normalize map[string]string) (traveller.TableResult, error) {
	characteristic, size, isAlteration, err := parseAlteration(cell)
	if err != nil {
		return nil, err
	}

	if isAlteration {
		return traveller.AlterationResult{Characteristic: characteristic, Delta: size}, nil
	}

	if glyph, _ := utf8.DecodeRuneInString(cell); glyph == traveller.EnDash {
		return nil, fmt.Errorf("%w: %q: %w", errMalformed, cell, traveller.ErrEnDash)
	}

	if startsWithSign(cell) {
		return nil, fmt.Errorf("%w: %q begins with a sign but is not a characteristic alteration", errMalformed, cell)
	}

	spelled := expand(cell, normalize)
	if category, ok := weaponCategories[spelled]; ok {
		return traveller.WeaponPickResult{Category: category}, nil
	}

	return traveller.SkillResult{Name: traveller.SkillName(spelled)}, nil
}

// parseBenefitRow reads one cell of Mustering Out Table 1 (p. 9).
func parseBenefitRow(cell string, normalize map[string]string) (traveller.BenefitRow, error) {
	if cell == emDash {
		return traveller.NoBenefit{}, nil
	}

	characteristic, size, isAlteration, err := parseAlteration(cell)
	if err != nil {
		return nil, err
	}

	if isAlteration {
		return traveller.AlterationBenefit{Characteristic: characteristic, Delta: size}, nil
	}

	spelled := expand(cell, normalize)

	if class, ok := passageClasses[spelled]; ok {
		return traveller.PassageBenefit{Class: class}, nil
	}

	if kind, ok := shipKinds[spelled]; ok {
		return traveller.ShipBenefit{Kind: kind}, nil
	}

	if spelled == travellersAid {
		return traveller.TravellersAidBenefit{}, nil
	}

	// A weapon row prints the bare category: "Blade", "Gun". P. 9: "Weapon
	// benefits must be declared as to type immediately."
	if category, ok := weaponCategories[spelled+" Combat"]; ok {
		return traveller.WeaponCategoryBenefit{Category: category}, nil
	}

	return nil, fmt.Errorf("%w: %q is not a benefit any row of Table 1 prints", errMalformed, cell)
}

// emDash is what Table 1 prints where a service has no benefit at that row.
// It is a hyphen in the data file and an em dash on the page; text
// extraction turns it into the digit 4, which is why nothing here is ever
// read from extracted text.
const emDash = "-"

const travellersAid = "Travellers' Aid"

// expand applies E012: a printed abbreviation becomes the name its own
// description carries. A string with no entry is already spelled out.
func expand(printed string, normalize map[string]string) string {
	if spelled, ok := normalize[printed]; ok {
		return spelled
	}

	return printed
}
