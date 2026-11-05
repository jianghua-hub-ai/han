# S413 independent CODE Review R01 - BLOCKED / STOP

Reviewer: Ptolemy/add-reviewer, 01a18492-af9e-7ea2-9cd5-e4dd368d7bee, non-author. Object: original five S4.13 files, Han/main/6da8abbceb0fa197364810a92a8fa2d97b38ddca. All23 frozen objects match entry and execution-release hashes. SOURCE rights remained zero. LOCAL/identityPG released after actual joins at local-pg-release.json; no further Go/SQL is planned.

## [HIGH] Approved anchored migration reaches bound but cannot restart normal

- Stable finding ID: RECOVERY-B13E-S413-CR-R01-F001. Status OPEN; blocks the bounded S413 code gate.
- Defect location: internal/recovery/installation.go:432-433 (literal Kind != genesis rejection in installationReader.VerifyGenesis).
- Actual consumer: internal/recovery/admission.go:468-474 and 539-547. Zero Store runs plus zero Recovery entries always selects the installed-origin reader; no alternate migration-origin branch exists there.
- Trigger: independently approved anchored_migration with Recovery initial head0/zero Begin/run and Article approved old prefix/head1; public Runtime.Install completes prepared -> both initial Anchors -> the SAME real RecoveryRunStore installation row -> bound. Stop that control Runtime, assemble a new Runtime and call public RunNormal.
- Expected: the valid bound origin supplies the initial no-Recovery-history proof, retaining independent source approval, exact installation/initial commitments, stable commitment reread and complete namespace continuity. Once any Begin exists, complete historical coverage remains mandatory.
- Actual: public Install succeeds and commitment is bound, but RunNormal returns errors.Is(ErrAdmissionBinding)=true, wrapped as ErrAdmissionIncomplete. Ordinary and race reproduce this. Genesis reaches normal in both modes. The full Store snapshot, metadata write count and media call count remain unchanged across the failing normal read; actual Stop returns.
- Impact: an otherwise accepted and successfully bound upgrade of an Article installation that never used recovery cannot enter normal operation. Creating a fake RecoveryRun/Begin or requiring a restore to get around the kind check would violate the original migration/no-Replay/no-Consume contract.

### Exact upstream basis and interpretation

The current frozen `.add/project/knowledge/design-workspace/identity-auth/design.md` SHA is 75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF.

- Lines3960-3962, section4.2 genesis/installation interruption/compatibility migration: anchored_migration is independently approved against the complete old-chain checkpoint/approved copy and expected head; metadata becomes prepared then bound; normal accepts bound and may verify continuity from the initial head; any Begin permanently switches to the complete-history path.
- Lines3964 and3968: migration does not Replay/Consume or invent/supersede missing runs; publication backfill applies only if actual old recovery histories need a verifiable source. No requirement says an approved Article-only migration must already have a RecoveryRun.
- Lines3598-3603 use the word genesis for the initial zero-history proof. That terminology is retained in this report rather than hidden. Reading it together with the explicit approved-migration/bound-continuity contract does not authorize fabricating a prior recovery history or excluding the accepted zero-history migration.
- Lines1662 and1682 define the independent read-only installation commitment validator and sameStore install ownership. This finding does not require a new provider/root/native prerequisite or a new grant interface.
- The current executor itself accepts this exact approved migration and reaches bound through the public producer. This is an actual producer-to-consumer mismatch, not an assumed future consumer requirement.

### Minimal repair boundary / independent recheck

Keep the original S413 five-file ownership. Correct the new installation reader's origin-kind eligibility so an independently approved, exact bound anchored_migration can use the legitimate zero-Recovery-history origin proof. Preserve strict manifest validation, exact binding/digest/initial heads, independent SourceVerifier, stable full commitment/revision reread, ctx/cause handling and prepared refusal. Do not open arbitrary kinds/purposes, weaken complete-history/Begin rejection, add fake runs, repair in normal, or change approval policy/SQL/wire/root requirements.

The likely minimal code change belongs in internal/recovery/installation.go, with regression coverage in the already-owned installation_test.go and recovery_installation_postgres_test.go. No production admission.go, Task, Plan or design change is requested by this report. The author must choose and implement the repair; the independent reviewer does not patch source.

Recheck on the new exact freeze using current full test prefixes plus the byte-retained reader and restart fragments. Both anchored negative reproductions must become PASS without changing expectations; retain genesis success, approval/prepared refusal, any-Begin/full-history failclosed, original37/2/16 risk groups and necessary related repair risks. Complete the remaining previously mapped validation before any final code PASS.

## Independent actual validation (current source23)

| Commands per mode | Leaves | Actual result |
| --- | ---: | --- |
| TestRecoveryInstallationBinding | 37 | normal0/race0, 37 PASS each |
| TestRecoveryInstallationStoreBinding | 2 | normal0/race0, 2 PASS each |
| TestReviewerS413BoundReaderKinds | 4 | normal1/race1, 3 PASS + 1 behavior FAIL each |
| TestReviewerS413AnchoredEmptyRecoveryRestart | 2 | normal1/race1, 1 PASS + 1 behavior FAIL each |

All90 leaf executions have terminal events:86 PASS,4 FAIL,0skip/unfinished. Per mode45 leaves43PASS2FAIL; these are NOT the author's original55-scope inventory. Eight Go commands used CPU1/-p1, offline/local/modreadonly, serial private D caches, empty ordinary DSN. Command argv, real returns, raw GoJSON and actual leaf names are retained under checks/ and actual-leaf-evidence.json. No test selector or compile failure is presented as behavior Red.

Independent reader normal/race raw: checks/reader-normal and checks/reader-race. The failing leaf is TestReviewerS413BoundReaderKinds/anchored-empty-recovery-old-article. Genesis-control, approval-refusal and prepared-refusal PASS in both modes.

Independent public/SQL normal/race raw: checks/restart-normal and checks/restart-race. The failing leaf is TestReviewerS413AnchoredEmptyRecoveryRestart/anchored_migration. Actual public Install -> production recovery executor -> production identity executor -> same real SQL Store -> production installation reader -> public RunNormal is exercised. The genesis leaf PASSes in both modes. Approval/CAS/media and Article/Recovery snapshots are explicit software fixtures, NOT native media/HMAC/source-provider proof. No private phase/grant is minted by the probes.

Four NEW own schemas were retained. Exact physical SQL rechecked host127.0.0.1/port51935/DBhan_b13e_identity/userhan_b13e/PG180006/current MainD data directory, then those four names only:24 tables each, one installation row/revision1 each, zero recovery_runs and Audit rows, anchored Article initial sequence1 versus genesis0. See physical-exact-own-schema-receipt.json and checks/physical-exact-own-schemas-explicit-target/.

The first external psql physical-check attempt used URI-in-PGDATABASE without explicit endpoint CLI; it attempted localhost5432 and was refused before any SQL. This reviewer setup error, exit2 and actual join are preserved under checks/physical-exact-own-schemas/; it is not product Red. The corrected new immutable attempt supplies the exact authorized target explicitly and exits0. No fallback data or old schemas were used. The earlier historical32-zip preparation assumption was corrected to the actual9 local zip inputs, retained as a setup diagnostic; all actual compilation was offline.

## Two rounds, provenance and pending scope

Round1 reused the already completed independent full1957-line source5/all-added diff audit: neutral installation/reader/codec, identity authority/SQL transaction/NULL-once migration and all three complete tests. Round2 reused bounded actual Runtime Install/admission/Journal/Store/0010-0011/Article controlled-core/historical-publication consumers and original Task/design/TP risks. Current actual execution focused the producer-to-normal mismatch and verified the dependency freeze including Lorentz's now-final12, without shadowing the identity package. Exact prior extents and method versions are retained under static-provenance/ and readset.json; this is not whole-repository or Auth12 semantic review.

Author55/611 and original PGfivegroups3/5/1/6/1 were independently parsed in the static phase, but remain AUTHOR executions. Author recoveryrace used exact recovery2 with earlier unrelated identity3; that boundary is retained rather than falsely claiming every command used final5. Four author formatter mutations and missing precise timestamps remain disclosed. Main's455payload/111schema/PID reception is Main evidence, not this reviewer's execution.

Under the explicit first-confirmed-bug stop rule, original PGscope16 normal/race, writer mismatch and delayed SourceVerifier ErrAnchorMissing candidates, extra real Article C1 executor/core seam and cancellation/resource-join probes are NOTRUN in this actual phase. Those two other static candidates are not findings or exonerations. This handoff returns the confirmed blocker to original Hooke; it does not claim the full final risk matrix completed. Stable prior reviews remain closed; source-dependent remaining validation resumes on the author's repair freeze.

All8 Go +2 external SQL attempts returned; each has suspended-before-assignment ownership, actual parent handle Wait, kernel Job active0, both stdout/stderr EOF and pump-thread join. All test cleanups/Runtime Stop and PG pools returned; PG physical helper recorded acquired0. These are own job and synchronous fixture lifecycle proofs, not host-wide or native provider process-tree claims. Main owns PG server stop; peer Confucius's lane is outside this release.

No source/repo/SSOT/design/Task/Plan/Git writes, production overlays, cleanup/deletion/move/archive, network installation or subdelegation occurred. Existing static discovery incident involving package headers remains disclosed in static-provenance/report-static.md. All selected current methods/rules were hash-verified against our own previous full reads; Role/mainSkill were refreshed in this continuation.

No root/global7/new format waiver is asserted. Author explicit old23 format subset FAIL2 is not a current root run and old dispositions do not expand. Real external approval/CAS/media/ACL/native wrapper/provider, full S413 production root/S416/long-lived resource ownership/E2E/business/human acceptance are unverified. Do not infer Task/G4-batch or human acceptance from this bounded review.

STOP: one HIGH OPEN; bounded code gate BLOCKED. LOCAL/identityPG already released; no own execution pending. Original author repair and original independent reviewer recheck are required.
