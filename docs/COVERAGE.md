# COVERAGE: the procedure, mapped to the code

2026-09-03. Every milestone complete. Companion to `PRD.md`.

Every rule of Book 1 pp. 4–25 that this tool implements, with the page that
governs it, the code that carries it, and the test that holds it. A rule with
no row here is a rule that is not implemented, and no row is without a test:
the generator refuses a rule an implementation names and no test holds.

The tables are generated from markers beside the code: an implementation
carries an `Implements:` line naming its rule, page and section, and the test
or golden that holds it carries a `Holds:` line naming the rule. Edit the
markers, not the tables, and regenerate with
`go test ./internal/coverage -regenerate`. The prose is written by hand.

Every table of pp. 4–25 is lifted and every one is consulted, and each is
transcribed twice — once in `rules/data`, once in the `rules` tests, from
the same visual reading of the page.

## What the engine walks

All six services. P. 10 makes ranks, commissions and promotions one fact —
"Ranks, commissions, and promotions are non-existent in the scout and other
services" — so the four that print a rank column have all three, and the two
that print none have none.

The book's own worked character (pp. 23–25) is replayed against the engine,
and Book 2 pp. 18–19 give the two ships the benefits name.

## Characteristics and the record

| Rule                                     | Page            | Implementation                | Test                                                         |
| ---------------------------------------- | --------------- | ----------------------------- | ------------------------------------------------------------ |
| Die roll conventions: `N+`, `N−`, exact  | 1:2–3           | `traveller.Target`            | `traveller.TestParseTarget`, `traveller.TestTargetSatisfied` |
| All characters begin at 18               | 1:4             | `chargen.startingAge`         | `chargen.TestGoldens`                                        |
| Floor of 0 for an aging reduction        | 1:4, 7–9 (E010) | `traveller.Profile.AgeReduce` | `traveller.TestAgeReduceFloorsAtZero`                        |
| Floor of 1 for every ordinary alteration | 1:4 (E010)      | `traveller.Profile.Alter`     | `traveller.TestTheTwoFloorsDiffer`                           |
| Six 2D rolls, in order                   | 1:4             | `chargen.run.rollProfile`     | `chargen.TestGoldens`                                        |
| Values 2–12 initially; never above 15    | 1:4             | `traveller.Profile.Alter`     | `traveller.TestAlterHoldsTheOrdinaryFloorAndTheCeiling`      |
| UPP as hexadecimal, in rolled order      | 1:8             | `traveller.Profile.UPP`       | `traveller.TestUPP`                                          |
| Two-die throws except the draft          | 1:10            | `chargen.roll`, `dice.Stream` | `dice.TestTwoDiceIsTwoDice`                                  |

## Enlistment and the draft

| Rule                                                            | Page       | Implementation               | Test                                    |
| --------------------------------------------------------------- | ---------- | ---------------------------- | --------------------------------------- |
| A civilian entered no service, and the record names none        | 1:5 (E001) | `chargen.Character.ServedIn` | `chargen.TestACivilianEnteredNoService` |
| Declining ends generation with a civilian                       | 1:5 (E001) | `chargen.run.draft`          | golden `scouts-civilian`                |
| Draftees are not eligible for commission in the first term only | 1:5        | `chargen.run.commission`     | golden `drafted-then-commissioned`      |
| One die, entering the service with that draft number            | 1:5        | `rules.Rules.Draft`          | `rules.TestDraft`                       |
| One enlistment attempt, against the Prior Service Table         | 1:5, 10    | `chargen.run.enlist`         | `chargen.TestGoldens`                   |
| The draft is offered, not compelled                             | 1:5 (E001) | `chargen.run.draft`          | golden `scouts-civilian`                |
| Cumulative characteristic DMs                                   | 1:10       | `rules.Throw.Modifier`       | `rules.TestModifierIsCumulative`        |

## Terms of service

| Rule                                                                                            | Page             | Implementation              | Test                                            |
| ----------------------------------------------------------------------------------------------- | ---------------- | --------------------------- | ----------------------------------------------- |
| A term is four years                                                                            | 1:5              | `traveller.Years`           | `chargen.TestACareerPastTheTablesLastColumn`    |
| Per-term order is the exposition's                                                              | 1:5–6 (E002)     | `chargen.run.term`          | `chargen.TestGoldens`                           |
| Survival throw; failure is death                                                                | 1:5              | `chargen.run.survive`       | golden `other-death`                            |
| The dead character's age and term                                                               | 1:5, 9 (E004)    | `chargen.run.term`          | golden `other-death`                            |
| A 12 exactly forces another term                                                                | 1:6              | `chargen.run.reenlist`      | `chargen.TestACareerPastTheTablesLastColumn`    |
| A 12 recurs past the seventh term                                                               | 1:6–7, 21 (E003) | `chargen.run.reenlist`      | `chargen.TestACareerPastTheTablesLastColumn`    |
| A commission or promotion confers one skill eligibility                                         | 1:6              | `chargen.run.raise`         | `chargen.TestGoldens`, `rules.TestEligibility`  |
| A departure below term 5 is forced out if the service refused him, discharged if he chose to go | 1:6, 21          | `chargen.run.depart`        | `chargen.TestADepartureIsDecidedByTheTermCount` |
| Commission: once per term until achieved                                                        | 1:6, 10          | `chargen.run.commission`    | golden `navy-captain`                           |
| No promotion at the top of the ranks table                                                      | 1:6, 10 (E013)   | `chargen.run.promote`       | golden `merchants-captain`                      |
| Promotion: the commissioning term and every term after                                          | 1:6, 10          | `chargen.run.promote`       | golden `navy-captain`                           |
| Reenlistment throw every term, no DMs                                                           | 1:6              | `chargen.run.reenlist`      | `chargen.TestGoldens`                           |
| Seven terms of voluntary service                                                                | 1:7              | `chargen.lastVoluntaryTerm` | `chargen.TestVoluntaryServiceStopsAtSeven`      |
| Ranks and their names, per the Table of Ranks                                                   | 1:10             | `rules.Service.RankName`    | `rules.TestTableOfRanks`                        |
| Leaving at the end of term 5 or later is retirement                                             | 1:21             | `chargen.run.depart`        | `chargen.TestVoluntaryServiceStopsAtSeven`      |

## Skills and training

| Rule                                                           | Page           | Implementation                   | Test                             |
| -------------------------------------------------------------- | -------------- | -------------------------------- | -------------------------------- |
| Two eligibilities for the initial term, one thereafter         | 1:6            | `chargen.run.grantEligibility`   | `rules.TestEligibility`          |
| Blade and gun combat name a weapon at once                     | 1:11–13        | `chargen.applyResult.WeaponPick` | `chargen.TestGoldens`            |
| Names normalized to their description headings                 | 1:11–23 (E012) | `rules.Rules.Normalize`          | `rules.TestNormalization`        |
| The fourth table needs Education 8+                            | 1:11           | `chargen.run.offeredTables`      | `rules.TestEligibility`          |
| The skills table is designated before the die                  | 1:11           | `chargen.run.trainOnce`          | `chargen.TestGoldens`            |
| Skills accumulate as Skill-1, Skill-2, with no cap             | 1:12           | `chargen.Character.addSkill`     | `chargen.TestGoldens`            |
| Three kinds of result                                          | 1:12           | `chargen.applyResult`            | `traveller.TestTableResultFolds` |
| Rank-keyed grants, at the moment the rank is conferred         | 1:23           | `chargen.run.raise`              | golden `merchants-captain`       |
| Service-wide rank and service skills, granted once on entering | 1:23 (E005)    | `chargen.run.grantsOnEntering`   | golden `scouts-died`             |

## Aging

| Rule                                                            | Page          | Implementation                | Test                                         |
| --------------------------------------------------------------- | ------------- | ----------------------------- | -------------------------------------------- |
| A characteristic at zero is a medical crisis, resolved inline   | 1:7–8 (E007)  | `chargen.run.crisis`          | golden `other-crisis-survived`               |
| A failed crisis throw is death                                  | 1:7–8 (E008)  | `chargen.run.crisis`          | golden `other-crisis-died`                   |
| A round at the end of term 4 and every term after               | 1:7, 9 (E006) | `chargen.run.agingRound`      | golden `other-serve`                         |
| Saving throw 8+, with no modifier during generation             | 1:7 (E009)    | `chargen.run.crisis`          | `rules.TestMedicalCrisis`                    |
| Survival recovers to 1 and adds 1D months                       | 1:7–8         | `chargen.run.crisis`          | golden `other-crisis-survived`               |
| Education and Social Standing are unaffected by aging           | 1:9           | `rules.Rules.liftAging`       | `rules.TestAgingTable`                       |
| Intelligence is unaffected before age 66                        | 1:9           | `rules.Rules.liftAging`       | `rules.TestAgingTable`                       |
| Read off the table's term row, not its age row                  | 1:9 (E006)    | `rules.Aging.At`              | `rules.TestAgingTable`                       |
| Saving throws in the table's row order                          | 1:9 (E007)    | `chargen.run.agingRound`      | `chargen.TestGoldens`                        |
| The last column is terminal                                     | 1:9 (E014)    | `rules.Aging.At`              | `chargen.TestACareerPastTheTablesLastColumn` |
| The table's last printed term, which decides where E014 governs | 1:9 (E014)    | `rules.Aging.LastPrintedTerm` | `rules.TestTheAgingTablesLastPrintedTerm`    |

## Mustering out

| Rule                                                                | Page       | Implementation                       | Test                                                                         |
| ------------------------------------------------------------------- | ---------- | ------------------------------------ | ---------------------------------------------------------------------------- |
| One roll per term, plus rank extras                                 | 1:7, 9     | `rules.Muster.Rolls`                 | `rules.TestMusterRollsAndPassages`                                           |
| Retirement pay from term 5, not for Scouts or Other                 | 1:7, 21    | `chargen.run.pension`                | `rules.TestRetirementPay`, golden `navy-captain`                             |
| At most three rolls on Table 2                                      | 1:9        | `chargen.run.chooseMusterTable`      | `chargen.TestAtMostThreeRollsGoOnTableTwo`                                   |
| The +1 at rank 5 or 6 on Table 1                                    | 1:9        | `chargen.run.musterModifier`         | golden `merchants-table1-modifier`                                           |
| The +1 on Table 1 may be declined                                   | 1:9        | `chargen.run.musterModifier`         | golden `navy-spartan-declines`                                               |
| The +1 with gambling on Table 2                                     | 1:9        | `chargen.run.musterModifier`         | `chargen.TestGoldens`, `rules.TestMusterRollsAndPassages`                    |
| The dash rows deliver nothing                                       | 1:9        | `rules.Rules.liftBenefits`           | `rules.TestTheDashCellsAreNothing`                                           |
| The mustering-out table is designated before the die                | 1:9        | `chargen.run.chooseMusterTable`      | `chargen.TestGoldens`                                                        |
| The seven kinds of Table 1 row                                      | 1:9, 21–23 | `chargen.applyBenefit`               | `traveller.TestBenefitRowFolds`                                              |
| The pension table says whether it prices a term at all              | 1:21       | `rules.Retirement.Pay`               | `rules.TestRetirementPay`                                                    |
| A repeat weapon may be taken as expertise, or as a different weapon | 1:22       | `chargen.takeWeapon`                 | goldens `scouts-diversified`, `scouts-expertise`                             |
| Free Trader: 40 years of payments, 10 off per repeat                | 1:22–23    | `chargen.run.receiveShipAgain`       | golden `merchants-captain`                                                   |
| Travellers' Aid only once; duplicates wasted                        | 1:22       | `chargen.applyBenefit.TravellersAid` | `traveller.TestBenefitRowFolds`                                              |
| Scout ship: duplicates lost                                         | 1:23       | `chargen.run.receiveShipAgain`       | golden `scouts-second-ship`                                                  |
| The ships the two benefits name: Type S, Type A                     | 2:18–19    | `rules.Rules.Hull`                   | `rules.TestShipHulls`                                                        |
| Which departures are deaths is said in one place                    | (sums)     | `traveller.Fatal`                    | `cmd/ctchargen.TestOnlyTheTwoDeathsCountAsDying`, golden `other-crisis-died` |

## Titles

| Rule                                                      | Page             | Implementation            | Test                                                                                                          |
| --------------------------------------------------------- | ---------------- | ------------------------- | ------------------------------------------------------------------------------------------------------------- |
| Assessed once, at the end, against the final value        | 1:5; 3:22 (E011) | `chargen.run.assessTitle` | golden `other-title`                                                                                          |
| Social Standing 11+ may assume the hereditary title       | 1:5; 3:22        | `chargen.run.assessTitle` | golden `other-title`                                                                                          |
| The five ranks of nobility                                | 3:22             | `rules.Rules.TitleFor`    | `rules.TestNobility`                                                                                          |
| E011 is stamped only where the reading changed the answer | (E011)           | `chargen.run.assessTitle` | `chargen.TestEveryReadingIsStampedWhereItsConditionHolds`, goldens `marines-ennobled`, `other-lost-the-title` |
| The dead are assessed but not asked                       | (E011)           | `chargen.run.assessTitle` | golden `died-a-noble`                                                                                         |

## The record

| Rule                                                                          | Page             | Implementation                             | Test                                                                                                              |
| ----------------------------------------------------------------------------- | ---------------- | ------------------------------------------ | ----------------------------------------------------------------------------------------------------------------- |
| Only the two deaths count as dying                                            | 1:5, 1:7–8       | `traveller.Fatal`                          | `cmd/ctchargen.TestOnlyTheTwoDeathsCountAsDying`                                                                  |
| The headline names the service attempted, not the one the draft gave          | 1:5              | `render.whoseService`                      | `render.TestTheHeadlineSaysWhoseServiceItIs`                                                                      |
| The gambling modifier is offered only to one who has the expertise            | 1:9              | `cmd/ctchargen.player.MusterTable2DM`      | `cmd/ctchargen.TestInteractiveOffersTheGamblingModifier`                                                          |
| The skills table designated is the one he has trained on least                | 1:11             | `chargen.Policy.SkillTable`                | `chargen.TestOneCharacterTrainsOnMoreThanOneTable`                                                                |
| The weapon for a Blade or Gun result is drawn, not always the first printed   | 1:11-13          | `chargen.Policy.Weapon`                    | `chargen.TestTheAutoPolicyNamesMoreThanOneWeapon`, golden `marines-armed`                                         |
| A ship's terms follow its kind, not its numbers                               | 1:22–23          | `render.shipLine`                          | `render.TestAShipsTermsFollowItsKind`                                                                             |
| A batch checks for collisions without generating anybody                      | CLI sketch       | `cmd/ctchargen.plannedPaths`               | `cmd/ctchargen.TestTheCollisionCheckNeedsNoCharacters`                                                            |
| A batch says how many it wrote and how many died                              | CLI sketch       | `cmd/ctchargen.tally`                      | `cmd/ctchargen.TestABatchReportsASummaryThatCouldNotBeWritten`, `cmd/ctchargen.TestABatchSaysHowManyDied`         |
| A batch to a file is written all at once, never a record at a time            | CLI sketch       | `cmd/ctchargen.buffered`                   | `cmd/ctchargen.TestABatchToAFileIsStillWrittenAtOnce`                                                             |
| A batch to standard output arrives a record at a time                         | CLI sketch       | `cmd/ctchargen.intoStream`                 | `cmd/ctchargen.TestABatchToStandardOutputArrivesARecordAtATime`                                                   |
| A character is written one way, not two                                       | CLI sketch       | `cmd/ctchargen.flagsThatContradict`        | `cmd/ctchargen.TestTheTwoRenderingsCannotBothBeAskedFor`                                                          |
| A data file broken two ways reports the same one every run                    | JSON conventions | `rules.Rules.liftPriorService`             | `rules.TestABrokenServiceNamesTheSameColumnEveryRun`                                                              |
| A directory batch that cannot write a member leaves what came before          | CLI sketch       | `cmd/ctchargen.intoDirectory`              | `cmd/ctchargen.TestADirectoryBatchLeavesWhatItWroteBeforeAnObstruction`                                           |
| A fixture generates one golden, and the same one twice                        | Determinism      | `chargen.Generate`                         | `chargen.TestGoldensRegenerate`                                                                                   |
| A guided run's choices are recorded as the player's, not the policy's         | FR9              | `chargen.WithAnswerer`                     | `cmd/ctchargen.TestTheQuestionsStayOutOfTheRecord`                                                                |
| A key a later build adds inside an existing object is still refused           | JSON conventions | `docs/character.schema.json`               | `render.TestAnUnknownKeyInsideAnEventDoesNotValidate`                                                             |
| A one-term character reads as one term                                        | Presentation     | `render.terms`                             | `render.TestOneTermIsSingular`                                                                                    |
| A player nobody can be shown anything is not asked                            | FR9              | `cmd/ctchargen.player.sayf`                | `cmd/ctchargen.TestInteractiveStopsWhenItCannotBeRead`                                                            |
| A read that fails is not the input ending                                     | FR9              | `cmd/ctchargen.player.stopped`             | `cmd/ctchargen.TestAFailedReadIsNotTheInputEnding`                                                                |
| A record from a later build validates and renders, which is what v1 promises  | JSON conventions | `docs/character.schema.json`               | `render.TestALaterShapeWithAnUnknownFieldStillValidatesAndRenders`                                                |
| A record names the policy that answered it                                    | JSON conventions | `chargen.PolicyVersion`                    | `cmd/ctchargen.TestOnlyARecordThePolicyAnsweredNamesOne`                                                          |
| A record names the shape it is, so the v1 freeze has something to attach to   | JSON conventions | `render.recordShape`                       | `chargen.TestGoldens`, `render.TestARecordWithoutItsShapeDoesNotValidate`                                         |
| A record renders to what generating it rendered                               | FR8              | `render.SheetFrom`                         | `render.TestARecordRendersToWhatGeneratedItRendered`                                                              |
| A refused batch writes none of itself, on the members it will actually write  | CLI sketch       | `cmd/ctchargen.noMemberExists`             | `cmd/ctchargen.TestARefusedBatchWritesNothing`, `cmd/ctchargen.TestARefusedSurvivorBatchKeysOnTheIndicesItWrites` |
| A roll with nothing to meet is not recorded as a throw that succeeded         | FR11             | `traveller.RollEvent`                      | `render.TestEveryGoldenMatchesTheSchema`, `render.TestTheFlatShapeIsWhatWasWritten`                               |
| A stopped session says what was answered and that nothing was written         | ADR 0001         | `cmd/ctchargen.player.stopped`             | `cmd/ctchargen.TestAStoppedSessionSaysWhatWasLost`                                                                |
| A word after a help flag is refused, not discarded                            | CLI sketch       | `cmd/ctchargen.writeHelp`                  | `cmd/ctchargen.TestAWordAfterAHelpFlagIsRefused`                                                                  |
| A word that names no strategy never becomes one                               | FR9              | `chargen.parsed`                           | `chargen.TestAWordThatNamesNoStrategyIsNotOne`                                                                    |
| A write that fails partway stops the batch, naming the member                 | CLI sketch       | `cmd/ctchargen.writingTo`                  | `cmd/ctchargen.TestABatchReportsAWriteThatFailedPartway`                                                          |
| An answer outside the offered set is refused, not applied                     | FR9              | `chargen.logging.record`                   | `chargen.TestAnAnswerOutsideTheOfferIsRefused`                                                                    |
| An event kind this build does not know is said to be unknown                  | FR11             | `render.writeEvent`                        | `render.TestAnUnknownEventKindIsSaidToBeUnknown`                                                                  |
| An unreadable answer is asked again, never guessed at                         | FR9              | `cmd/ctchargen.player.choose`              | `cmd/ctchargen.TestInteractiveReAsksWhatItCannotRead`                                                             |
| Each choice names who decided and what was offered                            | FR11             | `chargen.logging`                          | `chargen.TestGoldens`                                                                                             |
| Every batch member matches the schema                                         | JSON conventions | `cmd/ctchargen.batch`                      | `cmd/ctchargen.TestEveryBatchMemberMatchesTheSchema`                                                              |
| Every choice point the engine offers is reached by a record                   | Documents        | `chargen.fixtures`                         | `chargen.TestEveryChoicePointIsReachedByARecord`                                                                  |
| Every question the player is asked is one the engine asked                    | FR9              | `cmd/ctchargen.player`                     | `cmd/ctchargen.TestInteractiveWalksACharacter`                                                                    |
| Every reading that governed a record is named on it                           | Authority        | `chargen.log.stamped`                      | `chargen.TestEveryReadingIsReachable`                                                                             |
| Every rule here names an implementation and a test that exist                 | Documents        | `internal/coverage`                        | `internal/coverage.TestAMarkerRemovedFailsNamingTheRule`                                                          |
| Every step the errata conditions read is found, as the kind they read it as   | Documents        | `chargen.record.dice`                      | `chargen.TestEveryStepTheConditionsReadIsFound`                                                                   |
| Every step, throw, choice and consequence, in order                           | FR11             | `chargen.log`                              | `chargen.TestGoldens`                                                                                             |
| The alternatives offered are exactly the engine's                             | FR9              | `cmd/ctchargen.player.choose`              | `cmd/ctchargen.TestInteractiveAsksTheYesOrNoPoints`                                                               |
| The errata a record stamps are exactly those whose condition ERRATA.md states | Documents        | `chargen.conditions`                       | `chargen.TestEveryReadingIsStampedWhereItsConditionHolds`                                                         |
| The errata are on the transcript and not the sheet                            | FR11             | `render.sheetOf`                           | `render.TestTheErrataAreOnTheTranscriptAndNotTheSheet`                                                            |
| The input ending is an error, not a default                                   | FR9              | `cmd/ctchargen.errNoAnswer`                | `cmd/ctchargen.TestInteractiveRefusesToInventAnAnswer`                                                            |
| The procedure is shown between the questions                                  | FR11             | `chargen.WithObserver`, `render.EventLine` | `cmd/ctchargen.TestInteractiveShowsWhatAutoDoesNot`                                                               |
| The questions go to stderr, not into the record                               | CLI sketch       | `cmd/ctchargen.run`                        | `cmd/ctchargen.TestTheQuestionsStayOutOfTheRecord`                                                                |
| The record matches the schema that describes it                               | JSON conventions | `docs/character.schema.json`               | `render.TestEveryGoldenMatchesTheSchema`                                                                          |
| The record names its ruleset and the build that wrote it                      | Determinism      | `chargen.Ruleset`, `cmd/ctchargen.stamp`   | `cmd/ctchargen.TestRunWritesEachRendering`                                                                        |
| The shape a reader gets back is the shape that was written                    | FR11             | `render.eventCodec`                        | `render.TestTheFlatShapeIsWhatWasWritten`                                                                         |
| The spellings that gate rests on tell every value of an alphabet apart        | FR9              | `traveller`                                | `traveller.TestNoTwoValuesOfAnAlphabetShareASpelling`                                                             |
| The two documented examples are generated, not written                        | Documents        | `chargen.documentedExample`                | `chargen.TestGoldens`                                                                                             |
| This build writes record shape 1                                              | JSON conventions | `render.recordShape`                       | `cmd/ctchargen.TestThisBuildWritesShapeOne`                                                                       |
| What the command writes matches the schema, build stamp and all               | JSON conventions | `cmd/ctchargen.run`                        | `cmd/ctchargen.TestWhatTheCommandWritesMatchesTheSchema`                                                          |
| `--survivors` gives up rather than drawing forever                            | CLI sketch       | `cmd/ctchargen.survivorAttempts`           | `cmd/ctchargen.TestSurvivorsGivesUpRatherThanDrawingForever`                                                      |
| `--survivors` writes only the living, and generates another                   | CLI sketch       | `cmd/ctchargen.eachMember`                 | `cmd/ctchargen.TestSurvivorsWritesOnlyTheLiving`                                                                  |

## The book's own character

| Rule                                                     | Page           | Implementation                                 | Test                                       |
| -------------------------------------------------------- | -------------- | ---------------------------------------------- | ------------------------------------------ |
| A printed table governs over the example's stated result | 1:9, 25 (E015) | `rules.Rules.liftBenefits`                     | `chargen.TestTheWorkedExamplesDepartures`  |
| The worked example reproduces                            | 1:23–25        | `chargen.Generate`                             | `chargen.TestTheWorkedExampleReproduces`   |
| The example's per-term order is not followed             | 1:24 (E002)    | `chargen.run.term`                             | `chargen.TestTheWorkedExamplesDepartures`  |
| Its ship, cash and pension                               | 1:25           | `chargen.run.musterOut`, `chargen.run.pension` | `chargen.TestTheWorkedExamplesPossessions` |
| Its skills, as its inset lists them                      | 1:25           | `chargen.Character.addSkill`                   | `chargen.TestTheWorkedExamplesSkills`      |
| The example's batched aging is not followed              | 1:25 (E006)    | `chargen.run.agingRound`                       | `chargen.TestTheWorkedExamplesDepartures`  |
