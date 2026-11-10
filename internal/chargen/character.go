package chargen

import (
	"cmp"
	"slices"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// Character is the record a generation produces, complete whether it ended
// in mustering out or in death.
//
// A deliberate delta from sibling tools: Books 1-3 have no calendar and no
// birthdate rules, so the record carries an age and a count of terms, not a
// date. That is the books, not an omission.
//
// Nothing it says of the character is kept on it beside its record (#441):
// Profile, Skills, Benefits, Pension, Nobility, Enlistment, Rank, RankName,
// Age and Departure fold the facts the outcomes carry, and Terms reads the
// number of the last term the record opens. What it keeps besides the record is
// what the generation was given and what stamped it: its Inputs, its
// Policy, its Errata and its Build.
type Character struct {
	Inputs Inputs

	// Policy is what answered an auto run. It is nil on a guided run, where a
	// person answered and no policy decided anything.
	Policy *Policy

	Events []traveller.Event
	Errata []traveller.Erratum

	// Build is the version of what wrote the record, from
	// debug.ReadBuildInfo - the same source the version subcommand reports.
	// Naming the build is what lets a referee say which code wrote the record
	// being read, or a report filed against.
	//
	// The command fills it, through WithBuild. A record generated
	// in-process by a test does not carry one, because the build info of a
	// test binary is not the build of a released tool and would make a
	// golden differ by machine.
	Build string
}

// Nobility is what the record carries about the hereditary title of p. 5:
// whether the character was eligible, which title the value confers (Book 3
// p. 22), and whether he assumed it. A character who died is assessed but not
// asked (E011).
type Nobility struct {
	Eligible bool
	Title    traveller.Title
	Assumed  bool
}

// Benefits is what mustering out delivered (p. 9, pp. 21-23).
type Benefits struct {
	Cash          traveller.Credits
	Passages      []traveller.PassageClass
	TravellersAid bool
	Weapons       []traveller.WeaponName
	Ships         []traveller.Ship
}

// Inputs is what a generation was given besides who answers it, so that a
// record says what made it. Who answers is the way the run came in - Auto
// under a policy, or Guided - and the policy's strategies travel with it.
type Inputs struct {
	Name string

	// Forced is the service enlistment was attempted in, when one was
	// forced. It forces the attempt only: the throw is still made, and a
	// failed throw still goes to the draft (p. 5).
	Forced ForcedService
}

// ForcedService is the service a generation was told to attempt, or none.
//
// It is one value, not a service beside a flag (#253). There is nothing to
// set without the other: the service is the only field, and its absence is
// the zero value, which forces nothing. ForceService is the one way to name
// one, and it copies the name, so no caller can change it afterwards.
type ForcedService struct {
	service *traveller.ServiceName
}

// ForceService forces enlistment to be attempted in service.
func ForceService(service traveller.ServiceName) ForcedService {
	return ForcedService{service: &service}
}

// Service is the forced service, and whether one was forced.
func (f ForcedService) Service() (traveller.ServiceName, bool) {
	if f.service == nil {
		return 0, false
	}

	return *f.service, true
}

// serviceFold folds an enlistment into the service it entered, whether it
// entered one, and whether the draft is what entered it.
type serviceFold struct {
	service traveller.ServiceName
	served  bool
	drafted bool
}

func (s *serviceFold) Enlisted(service traveller.ServiceName) error {
	s.service, s.served = service, true

	return nil
}

func (s *serviceFold) Drafted(service traveller.ServiceName) error {
	s.service, s.served, s.drafted = service, true, true

	return nil
}

// DeclinedTheDraft leaves all three at their zero values, and served's false
// is what the caller must read: the zero ServiceName is Navy.
func (s *serviceFold) DeclinedTheDraft() error { return nil }

// ServedIn is the service the character entered, and whether he entered one.
//
// It is read off the Enlistment rather than held beside it (#42). The sum
// already says all of it, and a Service field kept separately would sit at
// its zero value, which is Navy, for a civilian, with only one more field to
// keep him from reading as a sailor.
//
// Implements: A civilian entered no service, and the record names none [1:5
// (E001)].
func (c *Character) ServedIn() (traveller.ServiceName, bool) {
	return servedIn(c.Enlistment())
}

// servedIn is the service an enlistment entered, and whether it entered one.
// Only generation sets an Enlistment, and it always does: a nil one panics
// here rather than reading as no service (#377).
func servedIn(enlistment traveller.Enlistment) (traveller.ServiceName, bool) {
	var fold serviceFold

	// This fold cannot fail: every case only assigns.
	_ = enlistment.Fold(&fold)

	return fold.service, fold.served
}

// Enlistment is how the character entered a service, or that he declined the
// draft and never did (p. 5), read from the record (#441): an enlistment throw
// he made, the service the draft put him in, or the draft declined. Nil for
// a record that holds none of the three.
func (c *Character) Enlistment() traveller.Enlistment {
	var fold sheetFold

	c.fold(&fold)

	return fold.enlistment
}

// Terms is how many terms the character began, read from the record (#441): the
// number of the last term a step opened, the fatal one included (E004).
func (c *Character) Terms() int {
	terms := 0

	for _, event := range c.Events {
		if step, isStep := event.(traveller.StepEvent); isStep && step.Opens > 0 {
			terms = int(step.Opens)
		}
	}

	return terms
}

// Rank is the character's place in his service's Table of Ranks (p. 10),
// read from the record (#441): the rank the last commission or promotion
// conferred, which the fact carries, or zero if none did.
func (c *Character) Rank() traveller.Rank {
	var fold sheetFold

	c.fold(&fold)

	return fold.rank
}

// RankName is what the Table of Ranks calls that rank (p. 10), read from the
// last commission or promotion's fact (#441), or empty if none was conferred.
func (c *Character) RankName() string {
	var fold sheetFold

	c.fold(&fold)

	return fold.rankName
}

// Age is the age generation left the character at, read from the record
// (#441). The UPP rolled at the start carries the age he began at, and every
// fact that moves it carries the age it left him at - the reenlistment
// throw's outcome at the end of each term, the fatal term's death, a crisis's
// months of recovery - so the last is his age, and the fold adds nothing to
// it.
func (c *Character) Age() traveller.Age {
	var fold sheetFold

	c.fold(&fold)

	return fold.age
}

// Departure is how the service ended, read from the record (#441): the last
// fact that ends it, since a crisis can kill a character the term's
// reenlistment throw has already retired. It is nil for a civilian, who
// never joined a service.
func (c *Character) Departure() traveller.Departure {
	var fold sheetFold

	c.fold(&fold)

	return fold.departure
}

// Profile is the six characteristics as generation left them, read from the
// record (#441): each starts at the value p. 4 rolled, and each change after it
// - a table's, an aging round's, a crisis's recovery - carries the value it
// left, so the last fact for a characteristic is its value.
func (c *Character) Profile() traveller.Profile {
	var fold sheetFold

	c.fold(&fold)

	return fold.profile
}

// Skills is every skill the character holds, at the level it reached, read
// from the record (#441), in name order, so a record reads the same way twice.
// Each fact that gives a skill carries the level the engine reached, so the
// last for a name is its level; the fold applies no rule of p. 12's.
//
// Implements: Its skills, as its inset lists them [1:25].
func (c *Character) Skills() []traveller.Skill {
	var fold sheetFold

	c.fold(&fold)

	slices.SortFunc(fold.skills, func(a, b traveller.Skill) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return fold.skills
}

// Benefits is what mustering out delivered, read from the record (#441). A
// passage, a weapon or a first ship is appended in the order received; the
// cash fact carries the running total and a repeat Free Trader the ship as it
// stands, so for those the last is the value. Which result is a repeat is
// the fact's to say, so the fold applies no rule of pp. 21-23.
func (c *Character) Benefits() Benefits {
	var fold sheetFold

	c.fold(&fold)

	return fold.benefits
}

// Pension is the annual retirement pay, read from the record (#441); zero for a
// character who draws none.
func (c *Character) Pension() traveller.Credits {
	var fold sheetFold

	c.fold(&fold)

	return fold.pension
}

// Nobility is what a final Social Standing of 11 or greater confers, assessed
// once at the end of generation (E011), read from the record (#441).
func (c *Character) Nobility() Nobility {
	var fold sheetFold

	c.fold(&fold)

	return fold.nobility
}

// fold walks the record's outcomes' facts through cases.
func (c *Character) fold(cases traveller.FactCases) {
	for _, event := range c.Events {
		outcome, isOutcome := event.(traveller.OutcomeEvent)
		if !isOutcome || outcome.Fact == nil {
			continue
		}

		// An error from cases is dropped, so every FactCases handed here must
		// be one whose cases only assign and cannot fail.
		_ = outcome.Fact.Fold(cases)
	}
}

// sheetFold reads the sheet's facts out of the record: the characteristics, the
// skills, the benefits, the pension, the title, and the career - how he
// entered, his rank, his age and how he left.
type sheetFold struct {
	profile  traveller.Profile
	skills   []traveller.Skill
	benefits Benefits
	pension  traveller.Credits
	nobility Nobility

	enlistment traveller.Enlistment
	rank       traveller.Rank
	rankName   string
	age        traveller.Age
	departure  traveller.Departure
}

func (s *sheetFold) Rolled(f traveller.RolledFact) error {
	s.profile[f.Characteristic] = f.Value

	return nil
}

func (s *sheetFold) Applied(f traveller.AppliedFact) error {
	// This fold cannot fail: both cases only assign.
	_ = f.Gain.Fold(s)

	return nil
}

func (s *sheetFold) Skill(skill traveller.Skill) error {
	s.skill(skill)

	return nil
}

func (s *sheetFold) Change(change traveller.Change) error {
	s.profile[change.Characteristic] = change.To

	return nil
}

func (s *sheetFold) WeaponPicked(f traveller.WeaponPickedFact) error {
	s.skill(f.Skill)

	return nil
}

func (s *sheetFold) Expertise(f traveller.ExpertiseFact) error {
	s.skill(f.Skill)

	return nil
}

func (s *sheetFold) Aged(f traveller.AgedFact) error {
	s.profile[f.Change.Characteristic] = f.Change.To

	return nil
}

func (s *sheetFold) Recovered(f traveller.RecoveredFact) error {
	s.profile[f.Characteristic], s.age = f.To, f.Age

	return nil
}

func (s *sheetFold) Cash(f traveller.CashFact) error {
	s.benefits.Cash = f.Total

	return nil
}

func (s *sheetFold) Passage(f traveller.PassageFact) error {
	s.benefits.Passages = append(s.benefits.Passages, f.Class)

	return nil
}

func (s *sheetFold) TravellersAid(traveller.TravellersAidFact) error {
	s.benefits.TravellersAid = true

	return nil
}

func (s *sheetFold) WeaponReceived(f traveller.WeaponReceivedFact) error {
	s.benefits.Weapons = append(s.benefits.Weapons, f.Weapon)

	return nil
}

// ShipReceived appends a ship received for the first time; a repeat Free
// Trader is FreeTraderAgain's, and a second scout ship carries no fact.
func (s *sheetFold) ShipReceived(f traveller.ShipReceivedFact) error {
	s.benefits.Ships = append(s.benefits.Ships, f.Ship)

	return nil
}

// FreeTraderAgain replaces the Free Trader already received with the fact's,
// which stands as the engine left it; the fact, not the fold, says it is a
// repeat.
func (s *sheetFold) FreeTraderAgain(f traveller.FreeTraderAgainFact) error {
	for i := range s.benefits.Ships {
		if s.benefits.Ships[i].Kind == traveller.FreeTrader {
			s.benefits.Ships[i] = f.Ship
		}
	}

	return nil
}

func (s *sheetFold) Pension(f traveller.PensionFact) error {
	s.pension = f.Pay

	return nil
}

func (s *sheetFold) Titled(f traveller.TitledFact) error {
	s.nobility = Nobility{Eligible: true, Title: f.Title, Assumed: f.Assumed}

	return nil
}

func (s *sheetFold) Raised(f traveller.RaisedFact) error {
	s.rank, s.rankName = f.Rank, f.Name

	return nil
}

// Enlisted enters the service the throw was made against when it accepted
// him. A rejection enters nothing: the draft's fact, or its refusal's, says
// what followed.
func (s *sheetFold) Enlisted(f traveller.EnlistedFact) error {
	if f.Accepted {
		s.enlistment = traveller.Enlisted{Service: f.Service}
	}

	return nil
}

func (s *sheetFold) Drafted(f traveller.DraftedFact) error {
	// A conversion, so a field added to either type stops this compiling.
	s.enlistment = traveller.Drafted(f)

	return nil
}

func (s *sheetFold) Declined(traveller.DeclinedFact) error {
	s.enlistment = traveller.DeclinedTheDraft{}

	return nil
}

func (s *sheetFold) Started(f traveller.StartedFact) error {
	s.age = f.Age

	return nil
}

func (s *sheetFold) Died(f traveller.DiedFact) error {
	s.age, s.departure = f.Age, traveller.KilledBySurvivalThrow{}

	return nil
}

func (s *sheetFold) DiedOfCrisis(f traveller.DiedOfCrisisFact) error {
	s.departure = traveller.KilledByMedicalCrisis{Characteristic: f.Characteristic}

	return nil
}

func (s *sheetFold) Retained(f traveller.RetainedFact) error {
	s.age = f.Age

	return nil
}

func (s *sheetFold) Reenlisted(f traveller.ReenlistedFact) error {
	s.age = f.Age

	return nil
}

func (s *sheetFold) Retired(f traveller.RetiredFact) error {
	s.age, s.departure = f.Age, traveller.Retired{}

	return nil
}

func (s *sheetFold) DeniedReenlistment(f traveller.DeniedReenlistmentFact) error {
	s.age, s.departure = f.Age, traveller.DeniedReenlistment{}

	return nil
}

func (s *sheetFold) Discharged(f traveller.DischargedFact) error {
	s.age, s.departure = f.Age, traveller.Discharged{}

	return nil
}

// skill holds a skill at the level a fact says it reached.
func (s *sheetFold) skill(skill traveller.Skill) {
	for i := range s.skills {
		if s.skills[i].Name == skill.Name {
			s.skills[i].Level = skill.Level

			return
		}
	}

	s.skills = append(s.skills, skill)
}
