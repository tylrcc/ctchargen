// Package rules holds every table of Book 1 pp. 4-25 as a transcription of
// the page, and lifts it into the domain values of the traveller package. The
// Acquired Skills tables are a Go literal in transcription.go; the rest are
// still embedded JSON in data/, until they move too (ADR 0005, #338).
//
// The lift validates the shape of every cell and the alphabet of every name.
// A cell that names no service, no characteristic, no benefit any row prints,
// or a target in a notation pp. 2-3 do not use, fails here — and because
// loading happens once on first use, a table that will not lift is a build
// defect that surfaces immediately rather than a runtime condition some path
// might reach.
//
// The readings are Go values here, one per erratum, so a reading missing or
// one with no erratum does not compile. They are not lifted: no markdown is
// embedded, and nothing parses prose.
//
// Every table is transcribed twice: once as the page prints it, and once into
// this package's tests, from the same visual reading of the page. The two must
// agree. That is what the reprint's font requires — its glyph substitutions
// look like data, so on Book 1 p. 9 the dash cells of Mustering Out Table 1
// extract as the digit 4 and "Travellers' Aid" extracts as Travellers9. A
// run that trusted the extraction would give a Scout a seventh benefit he
// does not have.
//
// So a value is the second transcription's to check, not the lift's. A key
// misspelled in a JSON file under data/ leaves its field at zero, and the lift accepts that
// zero wherever no check of its own refuses it; the tests' copy is what
// notices that the page prints something else.
package rules

import (
	"embed"
	"fmt"
	"slices"
	"sync"

	"github.com/philoserf/ctchargen/internal/traveller"
)

//go:embed data/*.json
var files embed.FS

// Sizes the printed tables fix.
const (
	// Faces is a die's faces, and so the rows of an Acquired Skills table
	// (p. 11).
	Faces = 6

	// MusterRows is the rows of a Mustering Out table (p. 9). Seven, not
	// six: a die of 1 through 6 plus the +1 a rank of 5 or 6 allows reaches
	// row 7.
	MusterRows = 7
)

// Rules is every table, lifted.
type Rules struct {
	services    [len(traveller.ServiceNames)]Service
	weapons     [len(traveller.WeaponCategories)][]traveller.WeaponName
	passages    [len(traveller.PassageClasses)]traveller.Credits
	hulls       [len(traveller.ShipKinds)]int
	grants      []Grant
	nobility    []Nobility
	normalize   map[string]string
	aging       Aging
	muster      Muster
	eligibility Eligibility
	education   Gate
	retirement  Retirement
}

// Load returns the lifted rules, loading and validating them once.
//
// Every caller in a process gets the same Rules, so nothing on it hands out a
// way to write to it: the tables are reached through methods, and a method
// that returns a slice returns a copy (#137).
func Load() (*Rules, error) { return loaded() }

var loaded = sync.OnceValues(load)

// Aging is the Aging Table (p. 9) and the medical crisis of pp. 7-8.
func (r *Rules) Aging() Aging { return r.aging }

// Muster is what p. 9's notes fix about mustering out.
func (r *Rules) Muster() Muster { return r.muster }

// Eligibility is the Basic Skill Eligibility box (p. 6).
func (r *Rules) Eligibility() Eligibility { return r.eligibility }

// Education is the gate on the fourth Acquired Skills table (p. 11).
func (r *Rules) Education() Gate { return r.education }

// Retirement is the Annual Retirement Pay table (p. 21).
func (r *Rules) Retirement() Retirement { return r.retirement }

// Service is one column of the Prior Service Table (p. 10), with everything
// the neighbouring tables key to a service.
type Service struct {
	Name        traveller.ServiceName
	Enlistment  Throw
	Draft       int
	Survival    Throw
	Reenlist    traveller.Target
	Ranks       []string
	Skills      [len(traveller.SkillTables)][Faces]traveller.TableResult
	Benefits    [MusterRows]traveller.BenefitRow
	Cash        [MusterRows]traveller.Credits
	PaysPension bool

	commission Throw
	promotion  Throw
}

// Throw is a target and the die modifiers that may apply to it (p. 10).
type Throw struct {
	Target traveller.Target
	DMs    []DM
}

// DM is one die modifier: an amount, and the characteristic threshold that
// earns it (p. 10).
type DM struct {
	Amount         int
	Characteristic traveller.Characteristic
	Threshold      traveller.Target
}

// Modifier is the total modifier a profile earns on this throw.
//
// P. 10: "DMs are cumulative (in the case of Enlistment) if the characters
// have the necessary prerequisites." Only enlistment prints two, so summing
// every modifier that applies is the same rule everywhere else.
//
// Implements: Cumulative characteristic DMs [1:10].
func (t Throw) Modifier(p traveller.Profile) int {
	total := 0

	for _, dm := range t.DMs {
		if dm.Threshold.Satisfied(p[dm.Characteristic]) {
			total += dm.Amount
		}
	}

	return total
}

// Commissions reports whether the service commissions at all. P. 10:
// "Ranks, commissions, and promotions are non-existent in the scout and
// other services." The same sentence governs all three, so one condition
// answers for all three: whether the Table of Ranks prints a column.
func (s Service) Commissions() bool { return len(s.Ranks) > 0 }

// Commission is the commission throw, and whether the service has one.
func (s Service) Commission() (Throw, bool) { return s.commission, s.Commissions() }

// Promotion is the promotion throw, and whether the service has one.
func (s Service) Promotion() (Throw, bool) { return s.promotion, s.Commissions() }

// MaxRank is the highest rank the service's column of the Table of Ranks
// prints. It is the ceiling E013 reads: where the column ends there is no
// next higher rank, so no promotion throw is made.
func (s Service) MaxRank() traveller.Rank { return traveller.Rank(len(s.Ranks)) }

// RankName is the name the Table of Ranks gives a rank, and whether the
// service prints one for it. Rank 0 is "not commissioned" and has no name.
//
// Not "title": the book keeps that word for nobility (p. 5, Book 3 p. 22),
// and calls Captain a rank (p. 10, and p. 25's "Merchant Captain (of rank 5
// on the scale of ranks)").
//
// Implements: Ranks and their names, per the Table of Ranks [1:10].
func (s Service) RankName(r traveller.Rank) (string, bool) {
	if r < 1 || int(r) > len(s.Ranks) {
		return "", false
	}

	return s.Ranks[r-1], true
}

// Row is the Table 1 benefit and the Table 2 cash at a roll of n (p. 9).
func (s Service) Row(n int) (traveller.BenefitRow, traveller.Credits, error) {
	if n < 1 || n > MusterRows {
		return nil, 0, fmt.Errorf("%w: mustering out roll %d: the tables print rows 1 through %d",
			errNoSuchRow, n, MusterRows)
	}

	return s.Benefits[n-1], s.Cash[n-1], nil
}

// Result is the Acquired Skills Table cell for a table and a die (p. 11).
func (s Service) Result(table traveller.SkillTable, die int) (traveller.TableResult, error) {
	if die < 1 || die > Faces {
		return nil, fmt.Errorf("%w: skills table roll %d: the table prints rows 1 through %d", errNoSuchRow, die, Faces)
	}

	return s.Skills[table][die-1], nil
}

// Service is the column of the Prior Service Table for a service. Indexing
// by the closed type is what removes the lookup that can miss.
func (r *Rules) Service(name traveller.ServiceName) Service { return r.services[name].clone() }

// clone is a Service sharing no backing array with the one it was taken from:
// the ranks and every throw's DMs are slices, and a Service is otherwise a
// value.
func (s Service) clone() Service {
	s.Ranks = slices.Clone(s.Ranks)
	s.Enlistment = s.Enlistment.clone()
	s.Survival = s.Survival.clone()
	s.commission = s.commission.clone()
	s.promotion = s.promotion.clone()

	return s
}

func (t Throw) clone() Throw {
	t.DMs = slices.Clone(t.DMs)

	return t
}

// Draft is the service a draft roll of n enters (p. 5).
//
// Implements: One die, entering the service with that draft number [1:5].
func (r *Rules) Draft(n int) (traveller.ServiceName, error) {
	for _, s := range r.services {
		if s.Draft == n {
			return s.Name, nil
		}
	}

	return 0, fmt.Errorf("%w: draft roll %d: no service prints that draft number", errNoSuchRow, n)
}

// Weapons is the printed list for a category (pp. 12-13), column-major: the
// page settles the order, as liftWeapons records.
//
// A copy: Load hands every caller in a process the same Rules, so returning
// the ruleset's own slice would let any caller's write change what every
// later character is offered (#137). A decider is handed a copy of its own
// besides, by the gate that asks it (#278); this one keeps the ruleset
// whole for every other caller.
func (r *Rules) Weapons(c traveller.WeaponCategory) []traveller.WeaponName {
	list := make([]traveller.WeaponName, len(r.weapons[c]))
	copy(list, r.weapons[c])

	return list
}

// Hull is the ship's hull size in tons (Book 2 pp. 18-19): the Type S is
// built on a type 100 hull, the Type A on a type 200.
//
// Implements: The ships the two benefits name: Type S, Type A [2:18–19].
func (r *Rules) Hull(kind traveller.ShipKind) int { return r.hulls[kind] }

// Passage is the purchase price of a passage class (pp. 21-22).
func (r *Rules) Passage(c traveller.PassageClass) traveller.Credits { return r.passages[c] }

// Normalize applies E012: a name as a table prints it becomes the name its
// own description carries.
//
// Implements: Names normalized to their description headings [1:11–23 (E012)].
func (r *Rules) Normalize(printed string) traveller.SkillName {
	return traveller.SkillName(expand(printed, r.normalize))
}

// Grant is one row of the Rank and Service Skills box (p. 23): a result that
// accrues automatically, without a throw and without using up eligibility.
//
// Rank 0 means the grant is by virtue of the service itself, which E005
// reads as granted once, on entering it.
type Grant struct {
	Service traveller.ServiceName
	Rank    traveller.Rank
	Result  traveller.TableResult
}

// GrantsOnEntering is what a service confers the moment a character joins it
// (p. 23, E005).
func (r *Rules) GrantsOnEntering(s traveller.ServiceName) []traveller.TableResult {
	return r.grantsAt(s, 0)
}

// GrantsAtRank is what a rank confers the moment it is conferred (p. 23).
func (r *Rules) GrantsAtRank(s traveller.ServiceName, rank traveller.Rank) []traveller.TableResult {
	return r.grantsAt(s, rank)
}

// Nobility is one row of Book 3 p. 22's table.
type Nobility struct {
	SocialStanding int
	Title          traveller.Title
}

// TitleFor is the title a Social Standing confers, and whether it confers
// one (Book 3 p. 22).
//
// Implements: The five ranks of nobility [3:22].
func (r *Rules) TitleFor(social int) (traveller.Title, bool) {
	for _, n := range r.nobility {
		if n.SocialStanding == social {
			return n.Title, true
		}
	}

	return 0, false
}

func (r *Rules) grantsAt(s traveller.ServiceName, rank traveller.Rank) []traveller.TableResult {
	var results []traveller.TableResult

	for _, g := range r.grants {
		if g.Service == s && g.Rank == rank {
			results = append(results, g.Result)
		}
	}

	return results
}

// Eligibility is the Basic Skill Eligibility box (p. 6).
type Eligibility struct {
	InitialTerm       int
	PerSubsequentTerm int
	OnCommission      int
	OnPromotion       int
}

// Gate is a characteristic threshold that opens something — here, the fourth
// Acquired Skills table (p. 11).
type Gate struct {
	Table          traveller.SkillTable
	Characteristic traveller.Characteristic
	Threshold      traveller.Target
}

// Open reports whether a profile passes the gate.
func (g Gate) Open(p traveller.Profile) bool { return g.Threshold.Satisfied(p[g.Characteristic]) }

// Retirement is the Annual Retirement Pay table (p. 21).
type Retirement struct {
	byTerms           map[int]traveller.Credits
	PerAdditionalTerm traveller.Credits

	// lastTabled is the largest term the table prints a row for, taken from
	// the data rather than written here. P. 21's table ends at 8 terms, and
	// pay beyond it is counted from that last row, so the 8 indexes a printed
	// table (CLAUDE.md's Authority point 6).
	lastTabled int
}

// Pay is the annual pension for a number of terms served, in a service that
// pays one, and whether the table prints one at all. P. 21: "Service beyond 8
// terms adds CR 2000 per additional term."
//
// The second return is the point. The table begins at five terms, so Pay(4)
// read a missing key and returned zero - a real-looking answer for a question
// the page does not answer - and the only thing standing between that and a
// record was a guard in the caller (#52). A precondition belongs to the
// function that has it.
//
// Implements: The pension table says whether it prices a term at all [1:21].
func (r Retirement) Pay(terms int) (traveller.Credits, bool) {
	if terms > r.lastTabled {
		beyond := traveller.Credits(terms - r.lastTabled)

		return r.byTerms[r.lastTabled] + beyond*r.PerAdditionalTerm, true
	}

	pay, tabled := r.byTerms[terms]

	return pay, tabled
}

// Muster is what p. 9's notes fix about mustering out.
type Muster struct {
	Table1DMFromRank5or6 int
	Table2DMFromGambling int

	// Table2ModifierFrom is the skill that earns the Table 2 DM (p. 9).
	//
	// The name lives beside the modifier it earns rather than as a literal
	// in the engine (#47). It has to match what the skills transcription lifts
	// to after E012's normalization, and the engine comparing against a Go string
	// meant that adding a normalization for it would silently stop the
	// modifier applying, with only an accident of roster composition to
	// notice. The lift applies that normalization to this name too, so the
	// two sides cannot spell the skill differently.
	Table2ModifierFrom traveller.SkillName

	ResalePercent int
}

// The benefit roll counts and the rank thresholds that earn them (p. 9).
//
// Each is a bare procedural bound - it indexes no printed table - so it is a
// constant beside the code that applies it rather than a row of
// mustering.json (#43, and CLAUDE.md's Authority point 6). Being printed is
// not the test; indexing a printed table is.
const (
	rollsPerTerm            = 1
	extraRollsForRank1or2   = 1
	extraRollsForRank3Plus  = 2
	minRankForOneExtraRoll  = 1
	minRankForTwoExtraRolls = 3
)

// Rolls is how many benefit rolls a character has earned.
//
// P. 9 states it in one sentence and reaches every rank: "Characters are
// allowed one roll per term of service; rank 1 or 2 is allowed one extra
// roll, rank 3 or higher is allowed two extra rolls." P. 7 says the same at
// more length, and adds that a character of rank 5 or 6 "receives 2 extra
// rolls, and may add 1 to his die roll when consulting Table 1" - which is
// why the test expects seven rolls for Jamison at five terms and rank 5.
//
// Implements: One roll per term, plus rank extras [1:7, 9].
func (m Muster) Rolls(terms int, rank traveller.Rank) int {
	rolls := terms * rollsPerTerm

	switch {
	case int(rank) >= minRankForTwoExtraRolls:
		rolls += extraRollsForRank3Plus
	case int(rank) >= minRankForOneExtraRoll:
		rolls += extraRollsForRank1or2
	}

	return rolls
}

// AgingEffect is one characteristic's line in the Aging Table at one band of
// terms (p. 9): the reduction, and the saving throw that avoids it.
type AgingEffect struct {
	Characteristic traveller.Characteristic
	Reduction      int
	Saving         traveller.Target
}

// Aging is the Aging Table (p. 9) and the medical crisis of pp. 7-8.
type Aging struct {
	bands           []agingBand
	lastPrintedTerm traveller.Term
	Crisis          Crisis
}

type agingBand struct {
	fromTerm traveller.Term
	effects  []AgingEffect
}

// Crisis is what happens when a characteristic reaches zero (pp. 7-8).
// E008 reads a failed saving throw as death; E009 reads the printed DM for
// attending medical personnel as having no referent during generation.
type Crisis struct {
	Saving     traveller.Target
	RecoversTo int
	MonthsDice int
}

// At is the aging effects that apply at the end of a term, in the table's
// own row order (E007). A term before the first band gets none.
//
// The last band is terminal: the Age row's 74+ is the only cell in either
// header row carrying a plus, so the column it labels is open-ended, and the
// Term row's 14 is simply the term that first arrives there (E014).
//
// Implements: Read off the table's term row, not its age row [1:9 (E006)].
// Implements: The last column is terminal [1:9 (E014)].
func (a Aging) At(term traveller.Term) []AgingEffect {
	var effects []AgingEffect

	for _, band := range a.bands {
		if term >= band.fromTerm {
			effects = band.effects
		}
	}

	return slices.Clone(effects)
}

// LastPrintedTerm is the last term the table's own header row names (p. 9).
//
// Past it a term is read off the last band's column anyway (E014).
//
// The number lives in the data because it indexes the printed table's header
// row, and because it is E014's stamping condition (#43). Held as a Go
// constant beside the code, changing the bands would move the table without
// moving the term at which a record declares the reading governed it, and
// nothing would say so.
//
// Implements: The table's last printed term, which decides where E014 governs
// [1:9 (E014)].
func (a Aging) LastPrintedTerm() traveller.Term { return a.lastPrintedTerm }
