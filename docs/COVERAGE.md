# COVERAGE: the procedure, mapped to the code

2026-09-03. Every milestone complete. Companion to `PRD.md`.

Every rule of Book 1 pp. 4–25 that this tool implements, with the page that
governs it, the code that carries it, and the test that holds it. A rule with
no row here is a rule that is not implemented; a row with no test is a defect.

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

| Rule                                     | Page            | Implementation                | Test                                                    |
| ---------------------------------------- | --------------- | ----------------------------- | ------------------------------------------------------- |
| Six 2D rolls, in order                   | 1:4             | `chargen.run.rollProfile`     | `chargen.TestGoldens`                                   |
| Values 2–12 initially; never above 15    | 1:4             | `traveller.Profile.Alter`     | `traveller.TestAlterHoldsTheOrdinaryFloorAndTheCeiling` |
| Floor of 1 for every ordinary alteration | 1:4 (E010)      | `traveller.Profile.Alter`     | `traveller.TestTheTwoFloorsDiffer`                      |
| Floor of 0 for an aging reduction        | 1:4, 7–9 (E010) | `traveller.Profile.AgeReduce` | `traveller.TestAgeReduceFloorsAtZero`                   |
| All characters begin at 18               | 1:4             | `chargen.startingAge`         | `chargen.TestGoldens`                                   |
| UPP as hexadecimal, in rolled order      | 1:8             | `traveller.Profile.UPP`       | `traveller.TestUPP`                                     |
| Die roll conventions: `N+`, `N−`, exact  | 1:2–3           | `traveller.Target`            | `traveller.TestParseTarget`, `TestTargetSatisfied`      |
| Two-die throws except the draft          | 1:10            | `chargen.roll`, `dice.Stream` | `dice.TestTwoDiceIsTwoDiceInOrder`                      |

## Enlistment and the draft

| Rule                                                            | Page       | Implementation               | Test                                    |
| --------------------------------------------------------------- | ---------- | ---------------------------- | --------------------------------------- |
| One enlistment attempt, against the Prior Service Table         | 1:5, 10    | `chargen.run.enlist`         | `chargen.TestGoldens`                   |
| Cumulative characteristic DMs                                   | 1:10       | `rules.Throw.Modifier`       | `rules.TestModifierIsCumulative`        |
| The draft is offered, not compelled                             | 1:5 (E001) | `chargen.run.draft`          | golden `scouts-civilian`                |
| One die, entering the service with that draft number            | 1:5        | `rules.Rules.Draft`          | `rules.TestDraft`                       |
| Declining ends generation with a civilian                       | 1:5 (E001) | `chargen.run.draft`          | golden `scouts-civilian`                |
| A civilian entered no service, and the record names none        | 1:5 (E001) | `chargen.Character.ServedIn` | `chargen.TestACivilianEnteredNoService` |
| Draftees are not eligible for commission in the first term only | 1:5        | `chargen.run.commission`     | golden `drafted-then-commissioned`      |

## Terms of service

| Rule                                                                                            | Page             | Implementation              | Test                                            |
| ----------------------------------------------------------------------------------------------- | ---------------- | --------------------------- | ----------------------------------------------- |
| A term is four years                                                                            | 1:5              | `traveller.Years`           | `chargen.TestACareerPastTheTablesLastColumn`    |
| Per-term order is the exposition's                                                              | 1:5–6 (E002)     | `chargen.run.term`          | golden transcripts                              |
| Survival throw; failure is death                                                                | 1:5              | `chargen.run.survive`       | golden `other-death`                            |
| The dead character's age and term                                                               | 1:5, 9 (E004)    | `chargen.run.term`          | golden `other-death`                            |
| Reenlistment throw every term, no DMs                                                           | 1:6              | `chargen.run.reenlist`      | golden transcripts                              |
| A 12 exactly forces another term                                                                | 1:6              | `chargen.run.reenlist`      | `chargen.TestACareerPastTheTablesLastColumn`    |
| A 12 recurs past the seventh term                                                               | 1:6–7, 21 (E003) | `chargen.run.reenlist`      | `chargen.TestACareerPastTheTablesLastColumn`    |
| Seven terms of voluntary service                                                                | 1:7              | `chargen.lastVoluntaryTerm` | `chargen.TestVoluntaryServiceStopsAtSeven`      |
| Leaving at the end of term 5 or later is retirement                                             | 1:21             | `chargen.run.depart`        | `chargen.TestVoluntaryServiceStopsAtSeven`      |
| A departure below term 5 is forced out if the service refused him, discharged if he chose to go | 1:6, 21          | `chargen.run.depart`        | `chargen.TestADepartureIsDecidedByTheTermCount` |
| Commission: once per term until achieved                                                        | 1:6, 10          | `chargen.run.commission`    | golden `navy-captain`                           |
| Promotion: the commissioning term and every term after                                          | 1:6, 10          | `chargen.run.promote`       | golden `navy-captain`                           |
| A commission or promotion confers one skill eligibility                                         | 1:6              | `chargen.run.raise`         | `rules.TestEligibility`; golden transcripts     |
| Ranks and their names, per the Table of Ranks                                                   | 1:10             | `rules.Service.RankName`    | `rules.TestTableOfRanks`                        |
| No promotion at the top of the ranks table                                                      | 1:6, 10 (E013)   | `chargen.run.promote`       | golden `merchants-captain`                      |

## Skills and training

| Rule                                                           | Page           | Implementation                   | Test                             |
| -------------------------------------------------------------- | -------------- | -------------------------------- | -------------------------------- |
| Two eligibilities for the initial term, one thereafter         | 1:6            | `chargen.run.grantEligibility`   | `rules.TestEligibility`          |
| The table is designated before the die                         | 1:11           | `chargen.run.trainOnce`          | golden transcripts               |
| The fourth table needs Education 8+                            | 1:11           | `chargen.run.offeredTables`      | `rules.TestEligibility`          |
| Three kinds of result                                          | 1:12           | `chargen.applyResult`            | `traveller.TestTableResultFolds` |
| Blade and gun combat name a weapon at once                     | 1:11–13        | `chargen.applyResult.WeaponPick` | golden transcripts               |
| Skills accumulate as Skill-1, Skill-2, with no cap             | 1:12           | `chargen.Character.addSkill`     | golden sheets                    |
| Service-wide rank and service skills, granted once on entering | 1:23 (E005)    | `chargen.run.grantsOnEntering`   | golden `scouts-died`             |
| Rank-keyed grants, at the moment the rank is conferred         | 1:23           | `chargen.run.raise`              | golden `merchants-captain`       |
| Names normalized to their description headings                 | 1:11–23 (E012) | `rules.Rules.Normalize`          | `rules.TestNormalization`        |

## Aging

| Rule                                                            | Page          | Implementation                | Test                                         |
| --------------------------------------------------------------- | ------------- | ----------------------------- | -------------------------------------------- |
| A round at the end of term 4 and every term after               | 1:7, 9 (E006) | `chargen.run.agingRound`      | golden `other-serve`                         |
| Read off the table's term row, not its age row                  | 1:9 (E006)    | `rules.Aging.At`              | `rules.TestAgingTable`                       |
| Saving throws in the table's row order                          | 1:9 (E007)    | `chargen.run.agingRound`      | golden transcripts                           |
| The last column is terminal                                     | 1:9 (E014)    | `rules.Aging.At`              | `chargen.TestACareerPastTheTablesLastColumn` |
| The table's last printed term, which decides where E014 governs | 1:9 (E014)    | `rules.Aging.LastPrintedTerm` | `rules.TestTheAgingTablesLastPrintedTerm`    |
| Education and Social Standing are unaffected by aging           | 1:9           | `rules` lift                  | `rules.TestAgingTable`                       |
| Intelligence is unaffected before age 66                        | 1:9           | `rules` lift                  | `rules.TestAgingTable`                       |
| A characteristic at zero is a medical crisis, resolved inline   | 1:7–8 (E007)  | `chargen.run.crisis`          | golden `other-crisis-survived`               |
| Saving throw 8+, with no modifier during generation             | 1:7 (E009)    | `chargen.run.crisis`          | `rules.TestMedicalCrisis`                    |
| Survival recovers to 1 and adds 1D months                       | 1:7–8         | `chargen.run.crisis`          | golden `other-crisis-survived`               |
| A failed crisis throw is death                                  | 1:7–8 (E008)  | `chargen.run.crisis`          | golden `other-crisis-died`                   |

## Mustering out

| Rule                                                                | Page       | Implementation                       | Test                                                                     |
| ------------------------------------------------------------------- | ---------- | ------------------------------------ | ------------------------------------------------------------------------ |
| One roll per term, plus rank extras                                 | 1:7, 9     | `rules.Muster.Rolls`                 | `rules.TestMusterRollsAndPassages`                                       |
| The table is designated before the die                              | 1:9        | `chargen.run.chooseMusterTable`      | golden transcripts                                                       |
| At most three rolls on Table 2                                      | 1:9        | `chargen.run.chooseMusterTable`      | `chargen.TestAtMostThreeRollsGoOnTableTwo`                               |
| The +1 at rank 5 or 6 on Table 1                                    | 1:9        | `chargen.run.musterModifier`         | golden `merchants-table1-modifier`                                       |
| The +1 on Table 1 may be declined                                   | 1:9        | `chargen.run.musterModifier`         | golden `navy-spartan-declines`                                           |
| The +1 with gambling on Table 2                                     | 1:9        | `chargen.run.musterModifier`         | `rules.TestMusterRollsAndPassages`; golden transcripts                   |
| The seven kinds of Table 1 row                                      | 1:9, 21–23 | `chargen.applyBenefit`               | `traveller.TestBenefitRowFolds`                                          |
| The dash rows deliver nothing                                       | 1:9        | `rules` lift                         | `rules.TestTheDashCellsAreNothing`                                       |
| Travellers' Aid only once; duplicates wasted                        | 1:22       | `chargen.applyBenefit.TravellersAid` | `traveller.TestBenefitRowFolds`                                          |
| A repeat weapon may be taken as expertise, or as a different weapon | 1:22       | `chargen.takeWeapon`                 | goldens `scouts-expertise`, `scouts-diversified`                         |
| Free Trader: 40 years of payments, 10 off per repeat                | 1:22–23    | `chargen.run.receiveShipAgain`       | golden `merchants-captain`                                               |
| Scout ship: duplicates lost                                         | 1:23       | `chargen.run.receiveShipAgain`       | golden `scouts-second-ship`                                              |
| The ships the two benefits name: Type S, Type A                     | 2:18–19    | `rules.Rules.Hull`                   | `rules.TestShipHulls`                                                    |
| Retirement pay from term 5, not for Scouts or Other                 | 1:7, 21    | `chargen.run.pension`                | `rules.TestRetirementPay`; golden `navy-captain`                         |
| The pension table says whether it prices a term at all              | 1:21       | `rules.Retirement.Pay`               | `rules.TestRetirementPay`                                                |
| Which departures are deaths is said in one place                    | (sums)     | `traveller.Fatal`                    | `ctchargen.TestOnlyTheTwoDeathsCountAsDying`, golden `other-crisis-died` |

## Titles

| Rule                                                      | Page             | Implementation            | Test                                                                                                           |
| --------------------------------------------------------- | ---------------- | ------------------------- | -------------------------------------------------------------------------------------------------------------- |
| Social Standing 11+ may assume the hereditary title       | 1:5; 3:22        | `chargen.run.assessTitle` | golden `other-title`                                                                                           |
| Assessed once, at the end, against the final value        | 1:5; 3:22 (E011) | `chargen.run.assessTitle` | golden `other-title`                                                                                           |
| The dead are assessed but not asked                       | (E011)           | `chargen.run.assessTitle` | golden `died-a-noble`                                                                                          |
| E011 is stamped only where the reading changed the answer | (E011)           | `chargen.run.assessTitle` | `docsgate.TestEveryReadingIsStampedWhereItsConditionHolds`; goldens `marines-ennobled`, `other-lost-the-title` |
| The five ranks of nobility                                | 3:22             | `rules.Rules.TitleFor`    | `rules.TestNobility`                                                                                           |

## The record

| Rule                                                                          | Page             | Implementation                             | Test                                                                                                      |
| ----------------------------------------------------------------------------- | ---------------- | ------------------------------------------ | --------------------------------------------------------------------------------------------------------- |
| Every step, throw, choice and consequence, in order                           | FR11             | `chargen.log`                              | golden transcripts                                                                                        |
| Each choice names who decided and what was offered                            | FR11             | `chargen.logging`                          | golden transcripts                                                                                        |
| Every reading that governed a record is named on it                           | Authority        | `chargen.log.stamped`                      | `chargen.TestEveryReadingIsReachable`                                                                     |
| One seed reproduces one character                                             | Determinism      | `dice.Stream`                              | `chargen.TestGoldensRegenerate`                                                                           |
| The record matches the schema that describes it                               | JSON conventions | `docs/character.schema.json`               | `render.TestEveryGoldenMatchesTheSchema`                                                                  |
| A record names the shape it is, so the v1 freeze has something to attach to   | JSON conventions | `render.recordShape`                       | `render.TestARecordWithoutItsShapeDoesNotValidate`, `chargen.TestGoldens`                                 |
| This build writes record shape 1                                              | JSON conventions | `render.recordShape`                       | `ctchargen.TestThisBuildWritesShapeOne`                                                                   |
| A record from a later build validates and renders, which is what v1 promises  | JSON conventions | `docs/character.schema.json`               | `render.TestALaterShapeWithAnUnknownFieldStillValidatesAndRenders`                                        |
| A key a later build adds inside an existing object is still refused           | JSON conventions | `docs/character.schema.json`               | `render.TestAnUnknownKeyInsideAnEventDoesNotValidate`                                                     |
| A record names the policy that answered it, and the document says so too      | JSON conventions | `chargen.PolicyVersion`                    | `docsgate.TestThePolicyVersionMatchesTheDocument`, `ctchargen.TestOnlyARecordThePolicyAnsweredNamesOne`   |
| The weapon for a Blade or Gun result is drawn, not always the first printed   | 1:11-13          | `chargen.Policy.Weapon`                    | `chargen.TestTheAutoPolicyNamesMoreThanOneWeapon`, golden `marines-armed`                                 |
| The skills table designated is the one he has trained on least                | 1:11             | `chargen.Policy.SkillTable`                | `chargen.TestOneCharacterTrainsOnMoreThanOneTable`                                                        |
| Every choice point the engine offers is reached by a record                   | Documents        | `internal/docsgate`                        | `docsgate.TestEveryChoicePointIsReachedByARecord`                                                         |
| What the command writes matches the schema, build stamp and all               | JSON conventions | `cmd/ctchargen.run`                        | `ctchargen.TestWhatTheCommandWritesMatchesTheSchema`                                                      |
| Every batch member matches the schema                                         | JSON conventions | `cmd/ctchargen.batch`                      | `ctchargen.TestEveryBatchMemberMatchesTheSchema`                                                          |
| The two documented examples are generated, not written                        | Documents        | `chargen.documentedExample`                | `chargen.TestGoldens`                                                                                     |
| The record names its ruleset and the build that wrote it                      | Determinism      | `chargen.Ruleset`, `cmd/ctchargen`         | `ctchargen.TestRunWritesEachRendering`                                                                    |
| Every row here cites a test and a golden that exist                           | Documents        | `internal/docsgate`                        | `docsgate.TestCoverageCitesTestsThatExist`                                                                |
| The errata a record stamps are exactly those whose condition ERRATA.md states | Documents        | `internal/docsgate`                        | `docsgate.TestEveryReadingIsStampedWhereItsConditionHolds`                                                |
| A batch member is seeded base + i, numbered from zero                         | Determinism      | `cmd/ctchargen.memberSeed`                 | `ctchargen.TestOneMemberIsTheSameCharacterAsNew`                                                          |
| Each member carries its own derived seed                                      | Determinism      | `cmd/ctchargen.member`                     | `ctchargen.TestEachMemberCarriesItsOwnSeed`                                                               |
| An explicit seed is never re-bounded                                          | Determinism      | `cmd/ctchargen.inputsFrom`                 | `ctchargen.TestAnExplicitSeedIsNeverReBounded`                                                            |
| A record renders to what generating it rendered                               | FR8              | `render.SheetFrom`                         | `render.TestARecordRendersToWhatGeneratedItRendered`                                                      |
| A ship's terms follow its kind, not its numbers                               | 1:22–23          | `render.shipLine`                          | `render.TestAShipsTermsFollowItsKind`                                                                     |
| The headline names the service attempted, not the one the draft gave          | 1:5              | `render.whoseService`                      | `render.TestTheHeadlineSaysWhoseServiceItIs`                                                              |
| A one-term character reads as one term                                        | Presentation     | `render.terms`                             | `render.TestOneTermIsSingular`                                                                            |
| The errata are on the transcript and not the sheet                            | FR11             | `render.sheetOf`                           | `render.TestTheErrataAreOnTheTranscriptAndNotTheSheet`                                                    |
| An event kind this build does not know is said to be unknown                  | FR11             | `render.writeEvent`                        | `render.TestAnUnknownEventKindIsSaidToBeUnknown`                                                          |
| A roll with nothing to meet is not recorded as a throw that succeeded         | FR11             | `traveller.RollEvent`                      | `render.TestTheFlatShapeIsWhatWasWritten`, `render.TestEveryGoldenMatchesTheSchema`                       |
| The shape a reader gets back is the shape that was written                    | FR11             | `render.eventCodec`                        | `render.TestTheFlatShapeIsWhatWasWritten`                                                                 |
| Every step the errata conditions read is found, as the kind they read it as   | Documents        | `internal/docsgate`                        | `docsgate.TestEveryStepTheConditionsReadIsFound`                                                          |
| A refused batch writes none of itself, on the members it will actually write  | CLI sketch       | `cmd/ctchargen.noMemberExists`             | `ctchargen.TestARefusedBatchWritesNothing`, `ctchargen.TestARefusedSurvivorBatchKeysOnTheIndicesItWrites` |
| A drawn seed can be the top of its own documented range                       | CLI sketch       | `cmd/ctchargen.drawSeed`                   | `ctchargen.TestADrawnSeedReachesTheTopOfItsRange`                                                         |
| A data file broken two ways reports the same one every run                    | JSON conventions | `rules.liftPriorService`                   | `rules.TestABrokenServiceNamesTheSameColumnEveryRun`                                                      |
| A batch to standard output arrives a record at a time                         | CLI sketch       | `cmd/ctchargen.intoStream`                 | `ctchargen.TestABatchToStandardOutputArrivesARecordAtATime`                                               |
| A batch to a file is written all at once, never a record at a time            | CLI sketch       | `cmd/ctchargen.buffered`                   | `ctchargen.TestABatchToAFileIsStillWrittenAtOnce`                                                         |
| A batch checks for collisions without generating anybody                      | CLI sketch       | `cmd/ctchargen.plannedPaths`               | `ctchargen.TestTheCollisionCheckNeedsNoCharacters`                                                        |
| A directory batch that cannot write a member leaves what came before          | CLI sketch       | `cmd/ctchargen.intoDirectory`              | `ctchargen.TestADirectoryBatchLeavesWhatItWroteBeforeAnObstruction`                                       |
| A batch says how many it wrote and how many died                              | CLI sketch       | `cmd/ctchargen.tally`                      | `ctchargen.TestABatchSaysHowManyDied`, `ctchargen.TestABatchReportsASummaryThatCouldNotBeWritten`         |
| `--survivors` writes only the living, and goes on to the next seed            | CLI sketch       | `cmd/ctchargen.eachMember`                 | `ctchargen.TestSurvivorsWritesOnlyTheLiving`                                                              |
| A member written under `--survivors` is still what its own seed makes         | CLI sketch       | `cmd/ctchargen.eachMember`                 | `ctchargen.TestASurvivorRegeneratesFromItsOwnSeed`                                                        |
| `--survivors` gives up rather than drawing forever                            | CLI sketch       | `cmd/ctchargen.survivorAttempts`           | `ctchargen.TestSurvivorsGivesUpRatherThanDrawingForever`                                                  |
| Only the two deaths count as dying                                            | 1:5, 1:7–8       | `cmd/ctchargen.fatalDeparture`             | `ctchargen.TestOnlyTheTwoDeathsCountAsDying`                                                              |
| A write that fails partway stops the batch, naming the member                 | CLI sketch       | `cmd/ctchargen.writingTo`                  | `ctchargen.TestABatchReportsAWriteThatFailedPartway`                                                      |
| Every question the player is asked is one the engine asked                    | FR9              | `cmd/ctchargen.player`                     | `ctchargen.TestInteractiveWalksACharacter`                                                                |
| The alternatives offered are exactly the engine's                             | FR9              | `cmd/ctchargen.choose`                     | `ctchargen.TestInteractiveAsksTheYesOrNoPoints`                                                           |
| A stopped session says what was answered and that nothing was written         | ADR 0001         | `cmd/ctchargen.stopped`                    | `ctchargen.TestAStoppedSessionSaysWhatWasLost`                                                            |
| A replay reproduces the character that was typed                              | Determinism      | `cmd/ctchargen.next`                       | `ctchargen.TestAReplayReproducesTheTypedCharacter`                                                        |
| A read that fails is not the input ending                                     | FR9              | `cmd/ctchargen.stopped`                    | `ctchargen.TestAFailedReadIsNotTheInputEnding`                                                            |
| Answers left over belong to another run, and are refused                      | Determinism      | `cmd/ctchargen.leftover`                   | `ctchargen.TestAnswersLeftOverBelongToAnotherRun`                                                         |
| A replay and --auto cannot both answer the questions                          | CLI sketch       | `cmd/ctchargen.newCharacter`               | `ctchargen.TestAnswersAndAutoCannotBothBeGiven`                                                           |
| An answer outside the offered set is refused, not applied                     | FR9              | `chargen.logging.record`                   | `chargen.TestAnAnswerOutsideTheOfferIsRefused`                                                            |
| The spellings that gate rests on tell every value of an alphabet apart        | FR9              | `traveller` alphabets                      | `traveller.TestNoTwoValuesOfAnAlphabetShareASpelling`                                                     |
| A strategy no row of POLICY.md carries never becomes one                      | FR9              | `chargen.parsed`                           | `chargen.TestAWordNoRowCarriesIsNotAStrategy`                                                             |
| An unreadable answer is asked again, never guessed at                         | FR9              | `cmd/ctchargen.choose`                     | `ctchargen.TestInteractiveReAsksWhatItCannotRead`                                                         |
| The input ending is an error, not a default                                   | FR9              | `cmd/ctchargen.errNoAnswer`                | `ctchargen.TestInteractiveRefusesToInventAnAnswer`                                                        |
| A player nobody can be shown anything is not asked                            | FR9              | `cmd/ctchargen.player.sayf`                | `ctchargen.TestInteractiveStopsWhenItCannotBeRead`                                                        |
| The questions go to stderr, not into the record                               | CLI sketch       | `cmd/ctchargen.run`                        | `ctchargen.TestTheQuestionsStayOutOfTheRecord`                                                            |
| A word after a help flag is refused, not discarded                            | CLI sketch       | `cmd/ctchargen.writeHelp`                  | `ctchargen.TestAWordAfterAHelpFlagIsRefused`                                                              |
| A character is written one way, not two                                       | CLI sketch       | `cmd/ctchargen.flagsThatContradict`        | `ctchargen.TestTheTwoRenderingsCannotBothBeAskedFor`                                                      |
| The procedure is shown between the questions                                  | FR11             | `render.EventLine`, `chargen.WithObserver` | `ctchargen.TestInteractiveShowsWhatAutoDoesNot`                                                           |
| The gambling modifier is offered only to one who has the expertise            | 1:9              | `cmd/ctchargen.player.MusterTable2DM`      | `ctchargen.TestInteractiveOffersTheGamblingModifier`                                                      |
| A guided run's choices are recorded as the player's, not the policy's         | FR9              | `chargen.WithAnswerer`                     | `ctchargen.TestTheQuestionsStayOutOfTheRecord`                                                            |

## The book's own character

| Rule                                                     | Page           | Implementation                     | Test                                       |
| -------------------------------------------------------- | -------------- | ---------------------------------- | ------------------------------------------ |
| The worked example reproduces                            | 1:23–25        | the whole engine                   | `chargen.TestTheWorkedExampleReproduces`   |
| Its skills, as its inset lists them                      | 1:25           | `chargen.Character.addSkill`       | `chargen.TestTheWorkedExamplesSkills`      |
| Its ship, cash and pension                               | 1:25           | `chargen.run.musterOut`, `pension` | `chargen.TestTheWorkedExamplesPossessions` |
| A printed table governs over the example's stated result | 1:9, 25 (E015) | `rules` lift                       | `chargen.TestTheWorkedExamplesDepartures`  |
| The example's per-term order is not followed             | 1:24 (E002)    | `chargen.run.term`                 | `chargen.TestTheWorkedExamplesDepartures`  |
| The example's batched aging is not followed              | 1:25 (E006)    | `chargen.run.agingRound`           | `chargen.TestTheWorkedExamplesDepartures`  |
