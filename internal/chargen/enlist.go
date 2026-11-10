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
	r.record.step("enlistment", "pp. 5, 10")

	name, err := r.chooseService()
	if err != nil {
		return err
	}

	service := r.tables.Service(name)

	throw := roll(r.roll, service.Enlistment.Target, service.Enlistment.Modifier(r.profile))
	seq := r.record.throw("enlistment, "+name.String(), throw)

	if throw.succeeded {
		r.record.fact(seq, nil, traveller.EnlistedFact{Service: name, Accepted: true})

		return r.join(traveller.Enlisted{Service: name})
	}

	r.record.fact(seq, nil, traveller.EnlistedFact{Service: name})

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
		r.enlistment = traveller.DeclinedTheDraft{}
		r.record.fact(0, []traveller.Erratum{traveller.E001}, traveller.DeclinedFact{})

		return nil
	}

	face := r.roll.Die()

	name, err := r.tables.Draft(face)
	if err != nil {
		return fmt.Errorf("the draft: %w", err)
	}

	seq := r.record.dice("draft", face)
	r.record.fact(seq, []traveller.Erratum{traveller.E001}, traveller.DraftedFact{Service: name})

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

		offers = append(offers, traveller.EnlistmentOffer{Service: name, Odds: r.odds(service.Enlistment)})
	}

	name, err := r.decide.Service(offers, varyOnly{r.roll})
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
	r.enlistment = how

	// join is handed only an enlistment that entered a service, so servedIn's
	// second result is true here; a civilian never joins.
	name, _ := servedIn(r.enlistment)

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

	r.record.step("rank and service skills", "p. 23")

	for _, result := range granted {
		err := r.apply(result, 0, []traveller.Erratum{traveller.E005}, traveller.OnEntering)
		if err != nil {
			return err
		}
	}

	return nil
}

// apply applies one Acquired Skills table result, or one grant, which take
// the same three shapes (p. 12), and records what it gives as a fact naming its
// source, which is where the service history files it. A mustering out
// benefit names AtMusteringOut, which the history leaves out: mustering out
// is not a term, and the sheet already lists what it delivered.
func (r *run) apply(
	result traveller.TableResult, because int, errata []traveller.Erratum, source traveller.Source,
) error {
	applied := &applyResult{run: r, because: because, errata: errata, source: source}

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
	source  traveller.Source
}

// Alteration applies a characteristic result, and keeps the table's change
// beside the two ends, so a change the floor or the ceiling held is not read
// as one of nothing (#409).
//
// Implements: A held change keeps the table's change [Generation record].
func (a *applyResult) Alteration(characteristic traveller.Characteristic, delta int) error {
	before := a.run.profile[characteristic]

	a.run.profile = a.run.profile.Alter(characteristic, delta)

	after := a.run.profile[characteristic]

	a.run.record.fact(a.because, a.errata, traveller.AppliedFact{
		Gain:   traveller.Change{Characteristic: characteristic, From: before, To: after, Delta: delta},
		Source: a.source,
	})

	return nil
}

func (a *applyResult) Skill(name traveller.SkillName) error {
	level := a.run.addSkill(name)
	a.run.record.fact(a.because, a.errata, traveller.AppliedFact{
		Gain: traveller.SkillGain{Skill: traveller.Skill{Name: name, Level: level}}, Source: a.source,
	})

	return nil
}

// Implements: Blade and gun combat name a weapon at once [1:11–13].
func (a *applyResult) WeaponPick(category traveller.WeaponCategory) error {
	list := a.run.tables.Weapons(category)

	// The run's own dice are a decider's only variation, here and for the
	// service (#420). Here pp. 12-13 print a list and no basis for picking
	// from it.
	chosen, err := a.run.decide.Weapon(category, list, varyOnly{a.run.roll})
	if err != nil {
		return fmt.Errorf("choosing a weapon: %w", err)
	}

	level := a.run.addSkill(traveller.SkillName(chosen))
	a.run.record.fact(a.because, a.errata, traveller.WeaponPickedFact{
		Category: category, Skill: traveller.Skill{Name: traveller.SkillName(chosen), Level: level}, Source: a.source,
	})

	return nil
}
