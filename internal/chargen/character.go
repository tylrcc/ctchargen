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
// Its characteristics and skills are not kept on it but read from its log
// (#441): Profile and Skills fold the facts the outcomes carry.
type Character struct {
	Age   traveller.Age
	Terms int

	// Enlistment is how the character entered a service, or that he
	// declined the draft and never did (p. 5).
	Enlistment traveller.Enlistment
	Rank       traveller.Rank
	RankName   string

	Benefits Benefits

	Pension traveller.Credits

	// Departure is how the service ended. It is nil for a civilian, who
	// never joined one.
	Departure traveller.Departure

	// Nobility is what a final Social Standing of 11 or greater confers,
	// assessed once at the end of generation (E011).
	Nobility Nobility

	Inputs Inputs

	// Policy is what answered an auto run. It is nil on a guided run, where a
	// person answered and no policy decided anything. The version it
	// answered as is this build's PolicyVersion, which the record writes
	// beside it (#369).
	Policy *Policy

	Events []traveller.Event
	Errata []traveller.Erratum

	// Build is what wrote the record, from debug.ReadBuildInfo - the same
	// source the version subcommand reports. Naming the build is what lets a
	// referee say which code wrote the record he is reading or filing a
	// report against.
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

// Ship is a starship received as a mustering out benefit (pp. 22-23).
type Ship struct {
	Kind traveller.ShipKind

	// Tons is the hull the ship is built on (Book 2 pp. 18-19).
	Tons int

	// Years is the ship's age. A Free Trader arrives new and ages ten years
	// with each further receipt; a Scout ship's age the page does not give.
	Years int

	// PaymentYears is how many years of payments remain on a Free Trader.
	// A Scout ship is held without title and carries none.
	PaymentYears int
}

// Benefits is what mustering out delivered (p. 9, pp. 21-23).
type Benefits struct {
	Cash          traveller.Credits
	Passages      []traveller.PassageClass
	TravellersAid bool
	Weapons       []traveller.WeaponName
	Ships         []Ship
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

// serviceCodec folds an enlistment into the service it entered, and whether
// the draft is what entered it.
type serviceCodec struct {
	service traveller.ServiceName
	served  bool
	drafted bool
}

func (s *serviceCodec) Enlisted(service traveller.ServiceName) error {
	s.service, s.served = service, true

	return nil
}

func (s *serviceCodec) Drafted(service traveller.ServiceName) error {
	s.service, s.served, s.drafted = service, true, true

	return nil
}

// DeclinedTheDraft leaves both at their zero values, and the false is what
// the caller must read: the zero ServiceName is Navy.
func (s *serviceCodec) DeclinedTheDraft() error { return nil }

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
	var codec serviceCodec

	// A fold over a codec cannot fail: every case only assigns.
	_ = c.Enlistment.Fold(&codec)

	return codec.service, codec.served
}

// Profile is the six characteristics as generation left them, read from the
// log (#441): each starts at the value p. 4 rolled, and each change after it
// - a table's, an aging round's, a crisis's recovery - carries the value it
// left, so the last fact for a characteristic is its value.
func (c *Character) Profile() traveller.Profile {
	var fold sheetFold

	c.fold(&fold)

	return fold.profile
}

// Skills is every skill the character holds, at the level it reached, read
// from the log (#441), in name order, so a record reads the same way twice.
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

// fold walks the log's outcomes' facts through cases.
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

// sheetFold reads the characteristics and the skills out of the log's facts.
type sheetFold struct {
	profile traveller.Profile
	skills  []traveller.Skill
}

func (s *sheetFold) Rolled(f traveller.RolledFact) error {
	s.profile[f.Characteristic] = f.Value

	return nil
}

func (s *sheetFold) Applied(f traveller.AppliedFact) error {
	// A fold over this codec cannot fail: both cases only assign.
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
	s.profile[f.Characteristic] = f.To

	return nil
}

func (*sheetFold) Raised(traveller.RaisedFact) error     { return nil }
func (*sheetFold) Enlisted(traveller.EnlistedFact) error { return nil }

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
