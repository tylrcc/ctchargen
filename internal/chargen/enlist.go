package chargen

import (
	"fmt"

	"github.com/philoserf/ctchargen/internal/traveller"
)

// enlist runs the enlistment attempt and, on rejection, the draft (p. 5).
//
// One attempt is permitted per character. The throw is made whether the
// service was chosen or forced by a flag, and a failed throw still goes to
// the draft, which can land anywhere.
//
// Implements: One enlistment attempt, against the Prior Service Table [1:5,
// 10].
func (r *run) enlist() error {
	r.log.step("enlistment", "pp. 5, 10")

	name, err := r.chooseService()
	if err != nil {
		return err
	}

	service := r.tables.Service(name)

	throw := roll(r.roll, service.Enlistment.Target, service.Enlistment.Modifier(r.char.Profile))
	seq := r.log.throw("enlistment, "+name.String(), throw)

	if throw.succeeded {
		r.log.outcomef(seq, nil, "enlisted in the %v", name)

		return r.join(traveller.Enlisted{Service: name})
	}

	r.log.outcomef(seq, nil, "the %v rejected him", name)

	return r.draft()
}

// draft offers the draft and takes the answer. P. 5 prints the step both
// ways - "he may submit to the draft" and "the character must submit to the
// draft" - and E001 reads the permissive one, so declining ends generation
// with an eighteen-year-old civilian.
//
// Implements: The draft is offered, not compelled [1:5 (E001)].
// Implements: Declining ends generation with a civilian [1:5 (E001)].
func (r *run) draft() error {
	submit, err := r.decide.SubmitToDraft()
	if err != nil {
		return fmt.Errorf("submitting to the draft: %w", err)
	}

	if !submit {
		r.log.outcomef(0, []traveller.Erratum{traveller.E001},
			"declined the draft, and remains a civilian")

		r.char.Enlistment = traveller.DeclinedTheDraft{}

		return nil
	}

	face := r.roll.Die()

	name, err := r.tables.Draft(face)
	if err != nil {
		return fmt.Errorf("the draft: %w", err)
	}

	seq := r.log.dice("draft", face)
	r.log.outcomef(seq, []traveller.Erratum{traveller.E001}, "drafted into the %v", name)

	return r.join(traveller.Drafted{Service: name})
}

// chooseService decides which service to attempt, unless a flag forced one.
func (r *run) chooseService() (traveller.ServiceName, error) {
	if service, forced := r.char.Inputs.Forced.Service(); forced {
		return service, nil
	}

	offers := make([]traveller.EnlistmentOffer, 0, len(traveller.ServiceNames))

	for _, name := range traveller.ServiceNames {
		service := r.tables.Service(name)

		offers = append(offers, traveller.EnlistmentOffer{
			Service: name,
			Target:  service.Enlistment.Target,
			DM:      service.Enlistment.Modifier(r.char.Profile),
		})
	}

	name, err := r.decide.Service(offers)
	if err != nil {
		return name, fmt.Errorf("choosing a service: %w", err)
	}

	return name, nil
}

// join enters the service and takes what it grants on entering.
//
// The service is read back off the enlistment rather than passed beside it:
// a name parameter would be a second copy of the fact the sum already
// carries, and nothing would hold the two equal. Everything after it reads
// the name as r.service.Name.
func (r *run) join(how traveller.Enlistment) error {
	r.char.Enlistment = how

	// join is handed only an enlistment that entered a service, so ServedIn's
	// second result is true here; a civilian never joins.
	name, _ := r.char.ServedIn()

	r.service = r.tables.Service(name)

	return r.grantsOnEntering()
}

// grantsOnEntering takes what the Rank and Service Skills box confers by
// virtue of the service itself (p. 23). E005 reads "as soon as he becomes
// eligible" as once, on entering, rather than once per term.
//
// Implements: Service-wide rank and service skills, granted once on entering
// [1:23 (E005)].
func (r *run) grantsOnEntering() error {
	granted := r.tables.GrantsOnEntering(r.service.Name)
	if len(granted) == 0 {
		return nil
	}

	r.log.step("rank and service skills", "p. 23")

	for _, result := range granted {
		err := r.apply(result, 0, []traveller.Erratum{traveller.E005})
		if err != nil {
			return err
		}
	}

	return nil
}

// apply applies one Acquired Skills table result, or one grant, which take
// the same three shapes (p. 12).
func (r *run) apply(result traveller.TableResult, because int, errata []traveller.Erratum) error {
	applied := &applyResult{run: r, because: because, errata: errata}

	err := result.Fold(applied)
	if err != nil {
		return fmt.Errorf("applying a result: %w", err)
	}

	return nil
}

// applyResult is the fold that applies a table result. Adding a fourth kind
// to the sum stops this compiling until it is handled here.
//
// Implements: Three kinds of result [1:12].
type applyResult struct {
	run     *run
	because int
	errata  []traveller.Erratum
}

func (a *applyResult) Alteration(characteristic traveller.Characteristic, delta int) error {
	before := a.run.char.Profile[characteristic]

	a.run.char.Profile = a.run.char.Profile.Alter(characteristic, delta)

	after := a.run.char.Profile[characteristic]

	a.run.log.outcomef(a.because, a.errata, "%v %+d, %d to %d", characteristic, delta, before, after)

	return nil
}

func (a *applyResult) Skill(name traveller.SkillName) error {
	level := a.run.char.addSkill(name)
	a.run.log.outcomef(a.because, a.errata, "%s-%d", name, level)

	return nil
}

// Implements: Blade and gun combat name a weapon at once [1:11–13].
func (a *applyResult) WeaponPick(category traveller.WeaponCategory) error {
	list := a.run.tables.Weapons(category)

	// The run's own dice are the only variation a decider gets, and only
	// here: pp. 12-13 print a list and no basis for picking from it.
	chosen, err := a.run.decide.Weapon(category, list, varyOnly{a.run.roll})
	if err != nil {
		return fmt.Errorf("choosing a weapon: %w", err)
	}

	level := a.run.char.addSkill(traveller.SkillName(chosen))
	a.run.log.outcomef(a.because, a.errata, "%v: %s-%d", category, chosen, level)

	return nil
}
