# S4.14 / B13D independent CODE R01 - CHANGES REQUIRED

Original nonauthor Ptolemy/add-reviewer01a18492-af9e-7ea2-9cd5-e4dd368d7bee.
Actual author: Chandrasekhar/add-backend-dev01a187a1-c022-71f3-92f5-6a9dc032e32b.
Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca. Exact five frozen sources,
full entry/exit SHA map attached. Shared/source/SSOT/Git writes0; no production overlay.

**Two HIGH OPEN: RECOVERY-B13D-S414-CR-R01-F001 and F002.** They are independently
compiled behavior failures, not compiler failures, native-feature omissions or
Main's preliminary observations adopted without evidence. Author/source is not
modified by this Reviewer. Return to original Chandrasekhar for bounded repair.

## Two completed review rounds

Round1: read all five current source files in full (restore568/exclusive71,
test728/exclusiveTest186/PGtest283 lines), entire newfile patch independently
reconstructed. Read original S414 card in full, frozen identityr55 Restore contract
3843-3857 and TP current shifted RootP0 rows832-836. Own same-version full Role/
add-code-review/checklist/abstraction/productredline/coding/nohardcode/go-service/
authorization methods were hash-verified; Workflow and applicable hardcode rule
full texts refreshed. Examine strict12-field wire/UTF8/duplicate/null/budget/time/
digest, stable artifact ownership, permission/attempt/unknown, child pipe/Wait,
exclusive wrappers, targets and actual test assertions. No generic future-provider
requirement or extra native API is inferred.

Round2: follow existing actual Runtime.Enter (runtime.go1431) and committed Begin
producer1514-1544 -> real PhaseContext/CheckPhase/current binding1122-1180 ->
Replace1498 -> private driver317-337 -> tool443-459 -> actual fixed-verb Start525 /
Wait556 and pipe joins557-559 -> independent target358-361 -> real VerifyRestored
1547-1558 -> existing bind. Resume1561-1597 does not Begin/Replace. Read current
CheckOwnership1090-1106 and all new source/test actual callsites. Private constructors
are only test-consumed, no production registration or fallback; approved native
facilities are explicitly still gated. PG target is a new connection reading actual
identity Store + a pre-established software source witness, not real native backup
restore evidence. Driver idle handle/root lifetime remains the stated S4.16 boundary.

## HIGH F001 - actual child image is not bound to the approved image

Location: recovery_restore.go428-440 /447-459 /525 (primary525).
Preflight opens and verifies the executable digest and approveImage callback, then
closes the checked handle. After another artifact check and external source handoff,
exec.Command(t.executable,verb) reopens the path. The original approved image evidence
has no lifetime covering actual launch. This violates the existing sole-approved-
binary SHA contract (r55:3847-3849, TP:833), and G0 software constraint3 explicitly
calls out SHA-then-path replacement; it is not a new demand to select native OS APIs.

Independent image leaf uses a retained OWN COPY of the CURRENT testbinary. Its
approveImage callback independently hashes the supplied checked handle. After the
last real Preflight, the original same-handle source handoff runs, then the probe
appends only a harmless trailing noncode marker to that owned executable file.
Retained copies prove the ENTIRE original binary prefix is identical and only that
trailer differs; no program instructions or test child protocol are changed. Actual
public Enter has committed Begin1. The image digest differs, yet actual restore-v1
starts once, actually waits/joins all pipes and reaches independent targetRead1.
A later Verify preflight refuses; bind0. Expected zero launches for the invalidated
image. Normal and race compiled leaf FAIL; exact healthy copy control reaches two
children (restore+verify), targetRead2, bind1/boundary. No malicious program or real
DB restore was used. The later refusal cannot undo a real restore child's effects.

Minimal direction: approved-image/lifetime evidence must cover the concrete Start
image through the handoff interval; absent such a facility failclosed instead of
launching from stale replaceable-path evidence. Another rehash at an arbitrary
replaceable-path instant alone does not prove lifetime. Do not select a native
backend/SDK or invent a public permission contract in this review. Full trigger,
impact, references and re-review requirements are in findings.json.

## HIGH F002 - actual lease loss is detected after launching new restore work

Location: recovery_restore.go320-330 /453-459 (primary456), Start525.
Replace consumes real current CheckPhase/ReadPermitBinding before additional
prepared/execute/tool work. After the external handoff, Run checks only ctx.Err.
Actual loss of the SAME exclusive facility need not already have canceled this ctx;
existing Runtime only observes it when CheckOwnership is consumed again. The next
permission check is after the tool/target sequence has already run.

Independent lease leaf uses that exact actual Runtime/publicEnter/driver/current
permit path and unchanged image. Immediately after successful original source
handoff it marks the same external fixture lease lost while ctx is live. The child
still starts once, targetRead1; only then real CheckPhase rejects and preserves
ErrRecoveryWriteUnknown. begin1/bind0. Expected zero NEW child after known lease
loss. Normal/race both behavior FAIL, healthy PASS; every child/pipe/Stop/Close joined
and deployment Close1 before reporting the failure. This is a current consumer
boundary defect, not a future root/native adapter implementation requirement.

Minimal direction: after completed handoff and before actual next Start, consume
existing current phase/binding/real lease authority in the execution path; context
error alone cannot stand in for lease validation. Keep attempted/unknown/readback
and true joins; do not add arbitrary grant minting or a new public Runtime API.

## Actual commands and evidence boundary

Only two own Go commands were needed before stopping on these blockers:

| Own execution | Actual leaves/result |
| --- | --- |
| original80-and-launch-normal | original80 PASS + healthy1 PASS + image1 FAIL + lease1 FAIL; exit1 |
| launch-race | healthy1 PASS + image1 FAIL + lease1 FAIL; exit1 |

Every selected leaf had a run/terminal event, zero skip/unfinished, no race warning
or compiler diagnostic. Full argv/PID/actual wait return/stdout/stderr are under
checks/. Own normal80 matches the author current80 inventory; author's complete
normal/race80/7/319 streams were also independently parsed and retained as AUTHOR
execution (author-evidence-audit), not counted as reviewer reruns. Their Green does
not close either independent finding. Original test bytes are an unchanged prefix
of the test-only overlay; no old assertion or production source is replaced.

Eight focal probe children (four per mode) have actual PID/Waited/PipesJoined
observations. Seventeen distinct child PIDs are logged across the full normal
invocation and focal race, including original tests; this is logged-PID evidence,
not a claim of a new whole-Host process enumeration. All direct Go waits returned;
actual focal Runtime.Stop, driver.Close, original callback and pipe join paths
returned before any test failure oracle. Private child stdin/stderr are not saved;
Go raw scans contain0 synthetic-canary leak matches. Owned binary copies/artifacts
remain in logged OS Temp paths outside the compact packet, with path/hash/unchanged-
prefix proof in actual-validation. No resource/evidence cleanup or schema DROP.

After confirming the two real failures, no new PG/SQL/old319/root/global7/fmt/vet/
Linuxcompile verification was started. Reviewer private restore_dsn was never read,
no reviewer schema created, original21 author schemas untouched. Main actualSQL21
and author20Go/22child/42PID evidence is explicitly separate from this independent
execution. Remaining required normal/race/P/oldregression checks are for the exact
repaired freeze; this R01 returns CHANGES REQUIRED, not incomplete scope disguised
as PASS. The fail->author->sameReviewer cycle is not bypassed by author Green.

## PG fixed resource observation and remaining limits

PGtest38 pins127.0.0.1:51934 and han_b13d_restore after reading the explicit DSN.
That is the exact authorized retained fixture resource in THIS grant, so current
PG7 author evidence is meaningful. The shared core rule permits temporary fixture
values; no alternative host/port/database was authorized or tested. Therefore this
is recorded as a portability limitation/nonblocking configuration suggestion, not
a third independently reproduced Restore runtime finding or a waiver. Future
reuse on a fresh dynamic resource should supply the expected approved target via
controlled test configuration and preserve isolation checks, not silently accept
arbitrary DBs. No new config names/provider choice or source changes are made here.

Full native Gate, production OS-lock backend, secure no-follow crossprocess handle
handoff, real approved native tools/durable receipt and process-tree isolation,
rootS4.16 idle-owner lifetime, full restore/E2E and human acceptance remain unknown/
closed. Software callbacks consume the same artifact handle in the parent; this is
not actual child-native-handle backup restore. No old closed B13A/B13B/B13C finding
was reopened, no active Article r27 design was read/hashed, and no rule/contract/
Task/Plan/waiver/G4/global status is changed. Declared native limits do not exempt
the current software image/lease launch boundary proved above.

**STOP / CHANGES REQUIRED / sole Go and Reviewer RestorePG window RELEASED.**
All own commands really joined, no pending Go/PG/callback; source5 entry/exit and
canonical frozen SHAs match. Author repair requires a new exact freeze and original
Reviewer re-review. Do not rerun from this receipt as an execution authorization.
