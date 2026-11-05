# Ptolemy independent S4.8 F001 code re-review R02

Result: CHANGES REQUIRED. Original RECOVERY-B13B-S48-CR-R01-F001 is CLOSED for its independently demonstrated held-lock/next-write defect. New HIGH RECOVERY-B13B-S48-CR-R02-F001 is OPEN: a real Anchor/retirement port panic bypasses operation expiry and held-lock release. The two causes/dispositions are separate. No implementation is patched and no Task/batch is closed. STOP/frozen for Main handback.

Reviewer: original non-author Ptolemy/add-reviewer 01a18492-af9e-7ea2-9cd5-e4dd368d7bee. Author: original Zeno/add-backend-dev 01a18486-3e11-7852-9723-ad380e0bb392. Workspace D:/AID/VSCode/Codex/Han, main, HEAD 6da8abbceb0fa197364810a92a8fa2d97b38ddca. Scope is the four-source R01-to-R02 repair and related guard-lifecycle risks; original S4.8 R01 full review and all unchanged scope are reused, not automatically replaced by self-tests. B13A/compat historical conclusions remain unchanged.

## Frozen object

Author Temp C:/Users/J00010385/AppData/Local/Temp/han-b13b-s48-r02-author-g9b4p1bp. Primary receipt.md SHA256 8A5BB9FC448D28ADC6217B6849C3D86353B53002D4AE361B70E1E9912BAA6832. Author index SHA256 9B9766DF5C6FA699D812ABBEFAEE01E8A684B7F83F2B4FD4006756A6C0AD5CE1; all72 indexed artifacts independently byte/hash checked. Source4.diff 1078 lines, +599/-121 including fixtures. Four before images match R01, four current live files match source-final frozen, remaining seven S4.8 sources and six direct protected recovery consumers match R01/compat hashes. Complete current original-eleven assembly below was checked at entry and completion.

| Current assembled original S4.8 source | SHA256, unchanged during Review |
|---|---|
| internal/recovery/journal_snapshot.go | 5F603DD09BAEC5DF69D6C2BD2B2D44F9BB2D4D1425FD2068EEFB94D4D56E38AC |
| internal/recovery/journal_snapshot_test.go | 99E6770617FA5217907F059E5A2A6896075226F385B7E33F69521F18F24C78A2 |
| internal/recovery/coordinator_context_test.go | 33CC74827F8A204E630681CEE25814996C978EAFA8869B5E93B859C53DF551BC |
| internal/recovery/journal.go | B346D5E56ED0BCA552F95CF35A5428EA5AB3DE13A23675CB82B3E97F14C12B95 |
| internal/recovery/coordinator.go | 14E56074C88B197E64500BD4E9121C5683AD5CF3ED2CE602F36689C9EBBD0BFA |
| internal/recovery/journal_test.go | F872120F08A2D36F1168C09E9ED0EF42204C5CB01604F776A68B97400CC8F256 |
| internal/recovery/journal_windows_test.go | 03FC45B08A248F428D63219479A0D720136CEB9D027F4F9156F0DF342BB27E7F |
| internal/recovery/filelock_flock.go | D7A5BA43B1E9E47BC2F4C7B8AD72AED6F206ED5E56688AD665463C96036457E3 |
| internal/recovery/filelock_unsupported.go | 4AD82AEE6250919AB4236FCF367FAC006D26205378409591E88611E160359765 |
| internal/recovery/filelock_flock_test.go | 2FDDAA752658E0CB22E5C9E25F6E927F0E2A1FE496BEDB5AC50EB00762BC0198 |
| internal/identity/recovery_run_store_integration_test.go | CADC4A4529ECAB2135FA2BAF3CCF632A5FC0E0B922272F2B528C50C78B7DB1F2 |

Only journal.go production changed. journal_snapshot.go, Coordinator, both native/unsupported lock implementations and the integration consumer remain byte-identical to R01. Pure validateRecords and HMAC/payload/entry helper bodies independently compare equal after newline normalization; no wire/crypto or actual SHA validation change.

## Two actual bounded review rounds

Round 1: complete source4.diff (all1078 lines), affected journal.go writer/operation/CAS/staging/dirsync/install/pending/opaque-retirement code and every changed callback/test caller. Reviewed instance-bound owned ctx, original source hash/Anchor observation, exact self-CAS succession, ctx values/deadline/cancellation, post-step loss checks, cause/unknown and release/expiry order. Read all15 new portable test leaves plus changed legacy retirement/error/join assertions and native pending-artifact test. Legacy fixture migration retains assertions and gives synthetic locks explicit Check behavior; no production fallback accepts a missing Check.

Round 2: followed guard across actual persistRecords -> stage -> pending CAS -> install/dirsync -> commit and pending/retirement callbacks, then checked the separate unchanged read-only Journal path and existing Runtime/Coordinator source/permit consumption. Original R01 full consumer review is reused after exact hash reconciliation. Main raised a precise panic-lifecycle question while this independent review was active; it was treated as a hypothesis, then independently reproduced via actual port callbacks below. No new architecture/SSOT requirement or future implementation was assumed.

## Original finding RECOVERY-B13B-S48-CR-R01-F001: CLOSED

Current fix: journal.go:704-779 validates an invocation-local owner/held lock/source and retains the exact observed Anchor. withLock:781-811 passes that owned ctx; each private mutating helper checks it at persistent boundaries. Only confirmWriterCAS:423-445 updates the expected observation after independent exact intended-successor confirmation. Instance/source/Anchor drift, loss or ctx cancellation prevents the next persistent action/readback. No receiver-global lock reference or reacquisition is used.

Original independent probes were rebuilt against CURRENT journal_snapshot_test.go. Only the two private callback signatures and helper ctx plumbing were adapted: helpers now receive callback-owned ctx. Original byte/CAS/loss/release/cause expectations remain; CAS assertion was strengthened from no-more-than-one to EXACTLY ONE first CAS and ErrRecoveryWriteUnknown. Thus an unguarded Background rejection before the first CAS cannot fake Green.

Actual normal/race both pass:

- TestS48ReviewerWriterLockLossFence/LossAfterCreateStopsBeforeWrite: staged_bytes=0, closed=1, original loss retained plus unknown. Its PositiveStaging control actually writes the original396-byte payload and closes once.
- TestS48ReviewerLostLockRejectsNextCAS: CAS_calls=1, lock_closed=1, loss+unknown retained; zero second CAS. This is the same original behavioral defect repaired, not a compile/no-match result.
- Independent actual nested persist checks Create/Write/PendingCAS/Rename/DirectorySync/CommitCAS/ABA/Cancel/Source. Zero next bytes/CAS/sync/readback and exactly one release are asserted at each appropriate boundary, with original ACK/loss/cancel/source cause and unknown. Both first and commit CAS injections are actually reached, not pre-entry rejection.
- Independent valid owned helper chain actually stages/installs396 bytes, executes two confirmed CAS calls and one directory sync, preserves values/deadline, releases once, then rejects the expired ctx. Before/after opaque retirement loss controls also block zero/next calls and retain post-call cause/unknown. The existing new15 portable author tests, pending helper positives/negatives, unknown/readback and true-join cases execute independently inside reader54.

The exact original R01 report/raw remain immutable in the original Temp/canonical chain. Closing its specific held-lock defect does not imply the entire new guard lifecycle is correct; the new finding below prevents S4.8 acceptance.

## [HIGH] RECOVERY-B13B-S48-CR-R02-F001: port panic leaves the operation live and the held lock unreleased

- Status: OPEN on the exact four/new-eleven hashes above.
- Location: internal/recovery/journal.go:799 (`operation(owned)`) and :810-811 (expiry/Close only on normal control flow); withLock starts at :781. Actual panic paths include compareAndSwapAnchor:420 and verifyAndConsumeKeyRetirementCapability:651.
- Basis: owned-operation contract expires before release when the invocation ends; core coding-standards resource ownership requires closure ownership, and adopted go-service/rules/coding-standards.md:12 explicitly requires defer for resource release. This is a concrete resource/authority failure rather than a style request to use defer. A synchronous external port can panic; upstream panic recovery must not be the owner of a hidden Journal lease.
- Trigger: Anchor.CompareAndSwap or the opaque retirement VerifyAndConsume actually panics after acquiring the lock/creating the owned context. Unwinding skips both op.closed.Store(true) and lock.Close because no deferred finalization is registered.
- Independently observed consequence: both real call paths are reached exactly once. The original panic object propagates unchanged, but release count=0, op_closed=false, and writerOperation(escapedCtx) returns nil after unwind. The acquired lock remains owned and its escaped context still passes the new guard. On the native mapping this can leave the exclusive file lock/handle retained while a caller recovers, blocking later operations; it also defeats the promised operation lifetime. Actual native flock execution is NOT claimed by these portable tests.
- Evidence: TestS48R02ReviewerPanicUnwindReleasesOwnedOperation/Anchor and /Retirement, checks/panic-normal and panic-race stdout.log/result.json. Each run has exactly two concrete behavior FAIL leaves, no skip, timeout, build failure or race warning. Both log original_panic_preserved=true, calls=1, op_closed=false, releases=0, escaped_guard_error=<nil>. Probe recovery exists only outside withLock to inspect and assert cleanup plus exact panic identity; production did not swallow/rewrite the panic.
- Minimum repair: register invocation-owned finalization immediately after acquisition/operation creation, guarantee expiry before exactly one held-lock Close on normal return, error and panic unwind, and preserve the original panic propagation. Preserve current normal error/cause aggregation and true join; do not add a global guard, silently recover into success, return before synchronous port unwind, or require the caller to know the hidden lease. Original-owner repair fits the existing writer/test boundary; no new design decision is presumed.
- Re-review conditions: both independent port-panic leaves must pass with the same original panic identity, exactly one release, closed operation and denied escaped ctx; original loss/CAS/byte assertions and normal/unknown/committed/join/readonly controls must remain green. Original F001 repair must not be weakened to solve panic cleanup. This is a distinct new root cause and stable finding, not a reinterpretation of Main's static observation as execution.

## Actual execution and scope reuse

Only a CURRENT journal_snapshot_test.go test overlay is used; no production overlay. Original probes.frozen copied as body evidence into new r01-probes-adapted.frozen; old package stays untouched. New artifacts use .frozen. All tests run through unchanged copied Main OS/path allowlist runner, GOENV off/GOTOOLCHAIN local/GOPROXY+GOSUMDB off/GOFLAGS+testDSN empty.

Commands and complete raw are in probe-summary.json:

1. independent-normal and independent-race: `go test [-race] -overlay <Temp>/probe-overlay.json ./internal/recovery -run '^(TestS48Reviewer|TestS48R02Reviewer|TestRecoveryJournalReadOnlyContext)' -count=1 -timeout=30s -json`. Each actually75 concrete leaves, all75 PASS, zero skip/fail/race warning. These include current reader54, original8 and13 added independent boundary/positive/retirement leaves. Reader retained K1/K2 success executes in the group body as well as negative children.
2. panic-normal and panic-race: same command using panic-overlay.json and -run '^TestS48R02ReviewerPanicUnwindReleasesOwnedOperation$'. Each2 leaves, both behavior FAIL as above. Primary passing run source remains retained separately from the panic extension; no rerun or failure was hidden/reclassified.

Initial overlay-building assertion mistakenly counted Close callback literals along with operation callbacks; it failed before Go. The generator was corrected to adapt only the exact two withLock call sites. No test compilation/runtime failure was credited as Green/Red and no assertion was weakened.

Shared readonly path checked directly: FileRecoveryJournal.Read/VerifyAgainst/ReadVerifiedSnapshot/VerifyBeginSecret still route to readerForJournal's separate verifiedRecoveryJournal.withLock/snapshot and old pure validateRecords. That path does not call checkWriter or demand writerOperation. Unchanged source bytes, actual reader54 and the retained-key independent positive confirm the writer fence did not become a pure-reader prerequisite. The synthetic native pending-artifact loss test remains Linux static-only on this Windows host; portable pending helpers executed, not real native artifact locking/parser path.

Author final evidence was parsed independently, not presented as Reviewer execution: normal/race405 nodes/367 concrete PASS leaves =54 reader+42 Coordinator+271 Runtime; all previous352 exact names retained. Full package race446 leaves=433pass+13 old POSIX skips; no fail/race warning. Source4fmt and Win/Linux/integration static vets exit0. Old R01 two fails in red-reused retain their original Reviewer provenance; they were not mechanically compiled as current tests. Initial-green and package-check failures remain failed intermediate author diagnostics. No full repository seven or stable271 rerun by Reviewer.

## Checks, reception limits and STOP

Main's current-object joint checks/format applicability are independently owned by Main. R01 six checks, root format FAIL2 and R01 exact15 disposition are historical, not extended to this new four-source object by this reviewer. No full-seven PASS or new format waiver is claimed here, and format decisions cannot waive the new HIGH. Component technical review proceeds without waiting for global-format disposition.

R01 full S4.8 review + R02 targeted repair evidence supports closure of original held-lock F001 only. The new panic resource/authority blocker prevents complete S4.8 code technical reception; Curie's separate B13B mechanism/batch assessment, Main Task/batch closure and human acceptance remain separate. B13A/compat and metadata findings/status are not reopened.

NOT RUN: real native POSIX/Anchor/provider/key service, real PG/DDL/integration/StoreSQL/publication IO, services/workers/restore/root/E2E. No real secret ENV/DSN/key access, network/download/install, shared source/document write, Git write, cleanup or subdelegation. All actual probes finish synchronously; panic probes hold only synthetic lock counters and no live OS lease/service remains awaiting join. All eleven/source4 and protected direct-consumer hashes rechecked unchanged. STOP/frozen; return to Main for original-author repair and same-reviewer follow-up.
