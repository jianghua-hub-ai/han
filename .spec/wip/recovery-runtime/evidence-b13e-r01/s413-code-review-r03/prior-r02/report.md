# S413 independent CODE Review R02 - STOP / new MEDIUM OPEN

Ptolemy/add-reviewer, 01a18492-af9e-7ea2-9cd5-e4dd368d7bee, non-author. Fixed Han/main/6da8abbceb0fa197364810a92a8fa2d97b38ddca. Original S413 source5 and current frozen23 stayed exact through all execution and release. SOURCE rights0. LOCAL/identityPG is already RELEASED after actual all9 joins; only own-D packaging remains.

## Original HIGH: CLOSED by independent current execution

RECOVERY-B13E-S413-CR-R01-F001 is CLOSED. Full current source/repair diff was read, original test bodies were byte-exact after removing only the added dispatches/appended new regressions, and the ORIGINAL reviewer4 reader +2 public/realSQL restart fragments were appended byte-exact to the NEW full test prefixes. All six probes PASS in both ordinary/race. Current original37+2+16 and new reader13/PG7 also actually PASS both modes (scope75); no skip or unfinished leaf. See f001-independent-recheck.json, actual-leaf-evidence.json, probes/ and source5 snapshots.

The repair retains validated genesis/anchored_migration whitelist, exact bound/digest/binding/initial heads, independent approval and unchanged commitment/revision reread; it additionally requires exact zero initial Recovery head for the zero-history origin proof. Existing any-Begin/full-history failclosed and source/prefix/approval/ABA guards remain. No code change or softened test expectation was made by this reviewer. Peer Auth F001 is outside this reviewer's closure authority.

## [MEDIUM] Known Writer mismatch is checked after installation effects

- Stable ID: RECOVERY-B13E-S413-CR-R02-F001. Status OPEN. Overall code gate BLOCKED/pending repair and remaining validation; no reviewer risk acceptance.
- Primary location: internal/recovery/installation.go:161-163. Related path: Install41-62, guard90-94 and prepare148-163.
- Trigger: construct the public neutral InstallationExecutor with a typed Writer different from RuntimeDependencies.InstallationWriter, while the actual Store/Commitments/Sources/Anchor/Recovery/Article ports remain the same and the manifest/source are valid. Enter through actual public NewRuntime/Install, not a private permit or phase.
- Expected: reject the known same-owner dependency mismatch with ErrAdmissionBinding before publishing prepared, invoking Media or creating initial Anchors. Keep the valid bound branch independent of unused writer requirements.
- Actual, ordinary AND race: ErrAdmissionBinding is eventually returned, but commitment already is prepared; CAS invocation count=3 (prepared+two namespace initial Anchors), Media=1, Anchors=2. Both own and other DB-bind counts=0, actual Stop succeeds. Healthy same-writer fresh installation in the unchanged original scope passes both modes.
- Root cause: guard checks six configured port identities against the real Runtime but omits Writer. Fresh/prepared Install only checks nil/missing Writer and Media. The available exact Writer-to-Runtime identity comparison happens in prepare AFTER Media and both initial Anchor steps.
- Consequence: an ordinary, detectable assembly error partially installs approved metadata/media/Anchors before refusing. The caller receives a dependency binding error but must now retain/resume prepared state rather than having a side-effect-free rejected configuration. No wrong-DB write or authorization bypass was observed; this bounded configuration-failure consequence is rated MEDIUM, not inflated to HIGH/data loss.

### Original contract and real consumer applicability

Current frozen `.add/project/knowledge/design-workspace/identity-auth/design.md` SHA75AA4C5D449E2CABEF6C1F66D731CB98BF679693B6AE4CD0E6D8BA0494B200BF:

- Section3.3.1 line1662: InstallationWriter is implemented by the SAME identity Store, with zero second pool/Store.
- Lines1672-1678: action-specific real dependencies are checked before the partial action/effects; fresh/prepared Install requires the installation writer, while bound only coordinates.
- Section4.2 lines3960-3966: prepared/Anchor installation is durable, interruption can only resume the same approved manifest, no reset/overwrite/automatic cleanup. A known bad dependency must not need these effects to be discovered.

The current code already defines this exact configuration as ErrAdmissionBinding; moving its existing check earlier does not invent a new provider contract. The real identity constructor at internal/identity/recovery_installation.go:16-24 protects its own d.Writer/d.Store against the supplied Store but does not inspect the encapsulated RuntimeDependencies.InstallationWriter; the inner neutral executor still performs that Runtime comparison only at161. This is a concrete current assembly/consumer seam, not an unimplemented future root requirement or a lying external provider.

### Minimal repair / recheck boundary

Original Hooke should repair within the already-owned five sources. In the fresh/prepared branch, after the independently classified bound shortcut and before the first prepared CAS/Media/Anchor effect, validate the actual writer against the SAME current Runtime writer binding; preserve the common guard, ctx/current permit/ownership and independent source checks. Do not impose a writer/media dependency on a bound-only coordination action. No public phase/setter, second Store, new API, design/Task/Plan change, rollback or cleanup is requested.

Keep TestReviewerS413MiswiredWriterBeforeEffects assertions: same binding error and ZERO metadata CAS/Media/Anchor/DB effects, actual Stop joined. It must become PASS under ordinary/race on the new source freeze. Preserve original F0016 and the current75 including bound-only dispatch, actual SQL/legacy and source/prefix/any-Begin negative controls. Then finish sourceMissing and necessary C1/cancel pending probes; don't treat this partial failure handoff as final full code PASS.

### Actual independent evidence

`checks/writer-normal/{stdout,stderr}.log/result.json` and `checks/writer-race/...` each exit1, exactly one executed behavior FAIL, no compile/setup/no-match/race failure. Their test-only CURRENT-prefix fixture is `probes/candidates-wholeprefix_test.go.frozen` plus candidates-overlay.json; original candidate fragment retained. The CAS/media side effects are actual calls to explicit software fixture ports, committed in their fixture backend. They are NOT a claim of real external CAS/media/native provider I/O. The production ordering and rejection are the actual code under review.

## Current execution and join boundary

| Each mode | Actual leaves | Result |
| --- | ---: | --- |
| InstallationBinding + original reader4 | 54 | PASS54 |
| StoreBinding | 2 | PASS2 |
| InstallationPostgres + original restart2 | 25 | PASS25 |
| independent wrong-writer | 1 | behavior FAIL1 |

Ordinary/race exact current sets:82 leaves each,81PASS1FAIL,0skip/unfinished; total164 leaf executions. The81 successful leaves are scope75 +original6, not a rerun of unrelated457/611 matrices. All eight Go parents used GOMAXPROCS1/-p1/-parallel1, offline/local/modreadonly, serial, old SAME-reviewer private D caches with new own-D scratch/env. Current Auth dependencies were compiled from the frozen final12, not shadowed old packages; no Auth semantic review is claimed.

One external read-only psql job independently checked the exact NEW50 schema names only, host127.0.0.1/51935, DBhan_b13e_identity/userhan_b13e, PG180006/current MainD data directory. Full installation images/counts/runs/Audit and physical table existence are retained and mapped back to each ordinary/race test. No historical/prefix schema scan, old data reuse or DROP. Real PG assertions/row/tx/pool cleanups and Runtime Stop returned; tests recorded acquired0. See physical-exact-own-schema-receipt.json, schema50-case-map.json and checks/physical-exact-own50/.

All8 Go +1 psql actual parent-handle Wait, kernel Job active0, BOTH pipe EOF and pump-thread join completed before release (local-pg-release.json/all-command-lifecycle.json). No own session/callback/SQL work remains. Job membership/observed PID evidence is not host-wide process or native-provider process-tree proof. Main owns server stop; peerConfucius lane is not released by this report.

## Two rounds, provenance and remaining limitations

Round1: complete three-file repair diff, current production installation.go full449lines, added reader13/PG7 bodies, unchanged original source5/other two files reused by exact SHA; full original1957line review remains applicable where unchanged. Round2: actual admission zero-history/complete-prefix selection, public Runtime manifest/classification/grant/postproof, sameStore identity wrapper, existing Article control core/namespace lifetime and callback ownership seams were traced within bounded dependencies. Exact selected methods/rules/profile/workflow were verified against our OWN prior complete same-version reads, Role/mainSkill refreshed. No unrelated Auth12 implementation body was read.

Static preparation root `D:/Temp/han-b13e-s413-review-r02-static-6qztdvu2` is retained. It independently parsed AUTHOR457 normal/race raw and full sealed author report/diff; those remain author execution evidence, not our test results or closure. Main's64author schemas/PID/packet reception is Main-only. The initially named currentfrozen-objects.json was absent; actual frozen-objects.json had the current23 map exactly matching the canonical grant/live. Source map proof is retained rather than silently substituting an old freeze.

Under the first newly confirmed bug stop rule, sourceMissing candidate and extra cancel/true-join/actual Article C1-core probes remain NOTRUN in R02. They have prepared fragments but no PASS, failure or exoneration. C1 prepared actual-core fixtures explicitly distinguish portable controlled I/O/software Store/CAS from native wrapper, real SQL, production provider/root. The new writer finding does not reopen prior B13A/B13B/Article/S414 closures.

Runner Python indentation and a nested-quote physical-wrapper preparation error occurred BEFORE process launch; diagnostics are preserved, corrected, and not called behavior Red. No production source overlay, source/repo/SSOT/Git write, cleanup/delete/move/archive, secret/otherDSN use, network installation or subdelegation occurred. Old23/root formatter evidence is not rerun, not reclassified PASS and no inherited/global waiver is adopted here. Native/external approval/media/ACL/provider/root/S416/long-lived-owner/E2E/business/human acceptance remain unverified. No Task/G4/global gate is signed.

STOP. Original HIGH CLOSED; new MEDIUM OPEN; remaining actual probes pending original-author repair and same-reviewer recheck. LOCAL/identityPG released, only this lane.
