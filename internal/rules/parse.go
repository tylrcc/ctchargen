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
// domain's own closed alphabet rather than against a second list kept here,
// save one: the abbreviations the tables print for a characteristic, which
// the type does not spell. A name that names no value of the type is a defect
// in the transcription, and the lift is where it must be caught.

// named indexes an alphabet by the name its values give themselves, so that
// the transcription's spelling is checked against the type and nothing else.
func named[T fmt.Stringer](values []T) map[string]T {
	index := make(map[string]T, len(values))
	for _, v := range values {
		index[v.String()] = v
	}

	return index
}

var (
	characteristics  = named(traveller.Characteristics[:])
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

// parseCharacteristic reads a characteristic as the page names it: in full,
// as p. 4 prints the six and the Aging Table repeats them, or abbreviated, as
// the narrower tables print them.
func parseCharacteristic(name string) (traveller.Characteristic, error) {
	if c, ok := abbreviatedCharacteristics[name]; ok {
		return c, nil
	}

	return lookup(characteristics, name, "characteristic")
}

// abbreviatedCharacteristics is each abbreviation the tables print for a
// characteristic: p. 9's Table 1 ("+1 Intel", "+2 Educ", "+2 Social"),
// p. 10's DM rows ("Intel 8+", "Stren 8+", "Dext 6+", "Endur 5+"), p. 11's
// Acquired Skills Table ("+1 Stren", "−1 Social") and p. 23's box
// ("+1 Social").
//
// Settled by the page and not a reading: each is the opening of exactly one of
// the six names p. 4 prints, so nothing is decided in reading it. It is not
// E012's either, which says so of Table 1's rows itself: "Table 1's
// characteristic rows (+1 Intel, +2 Educ, +2 Social and the rest) are not
// normalized here because a characteristic is never recorded as a printed
// string".
var abbreviatedCharacteristics = map[string]traveller.Characteristic{
	"Stren":  traveller.Strength,
	"Dext":   traveller.Dexterity,
	"Endur":  traveller.Endurance,
	"Intel":  traveller.Intelligence,
	"Educ":   traveller.Education,
	"Social": traveller.SocialStanding,
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
// amount, and the characteristic and threshold that earn it. The
// transcription spells the condition the way p. 10 does, "Intelligence 8+".
func parseDM(amount int, condition string) (DM, error) {
	name, threshold, ok := strings.CutLast(condition, " ")
	if !ok {
		return DM{}, fmt.Errorf("%w: die modifier %q: want a characteristic and a threshold", errMalformed, condition)
	}

	characteristic, err := parseCharacteristic(name)
	if err != nil {
		return DM{}, fmt.Errorf("die modifier %q: %w", condition, err)
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

// refuseEmpty refuses a cell with nothing in it. A Go array short of a cell is
// padded with "" in silence, and the page prints a dash where a cell means
// nothing, so an empty cell is a transcription defect and never a value.
func refuseEmpty(cell string) error {
	if strings.TrimSpace(cell) == "" {
		return fmt.Errorf("%w: an empty cell; the page prints a dash where it means nothing", errMalformed)
	}

	return nil
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
	err := refuseEmpty(cell)
	if err != nil {
		return nil, err
	}

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
	err := refuseEmpty(cell)
	if err != nil {
		return nil, err
	}

	if cell == emDash {
		return traveller.NoBenefit{}, nil
	}

	err = refuseDashMistyped(cell)
	if err != nil {
		return nil, err
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

// emDash is what the page prints where a cell means nothing: a benefit
// Table 1 does not give, and a commission, a promotion, a DM or a rank a
// service does not have. Text extraction turns it into the digit
// 4, which is why nothing here is ever read from extracted text.
//
// Three dashes look alike on screen, and each is decided here. The em dash is
// what the page prints, and means nothing. An en dash standing alone in a
// cell is refused by name, and one opening a number is refused as no sign
// (#288). A hyphen standing alone is refused by name too: it is what the
// transcription held before it was restored to the page's text (#339), and a
// cell typed with one is a cell not read off the page. A hyphen that opens a
// number is a minus, and is read as one (traveller.Sign).
const emDash = "—"

// refuseDashMistyped refuses a cell that is a lone hyphen or en dash where the
// page prints an em dash.
func refuseDashMistyped(cell string) error {
	switch cell {
	case "-":
		return fmt.Errorf("%w: %q is a hyphen; the page prints an em dash where a cell means nothing", errMalformed, cell)
	case string(traveller.EnDash):
		return fmt.Errorf("%w: %q: %w", errMalformed, cell, traveller.ErrEnDash)
	}

	return nil
}

const travellersAid = "Travellers' Aid"

// expand applies E012: a printed abbreviation becomes the name its own
// description carries. A string with no entry is already spelled out.
func expand(printed string, normalize map[string]string) string {
	if spelled, ok := normalize[printed]; ok {
		return spelled
	}

	return printed
}
