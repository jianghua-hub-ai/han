# S414 / B13D independent CODE R02 - CHANGES REQUIRED

Original Ptolemy/add-reviewer01a18492-af9e-7ea2-9cd5-e4dd368d7bee; original author
Chandrasekhar/add-backend-dev01a187a1-c022-71f3-92f5-6a9dc032e32b. Han/main/HEAD
6da8abbceb0fa197364810a92a8fa2d97b38ddca. Exact source5 entry/exit/frozen map attached;
only restore.go/test changed. Source/shared/SSOT/Git writes0; TEST-only overlays.

Original HIGH RECOVERY-B13D-S414-CR-R01-F001 and F002 are independently CLOSED on
this freeze. New HIGH RECOVERY-B13D-S414-CR-R02-F001 remains OPEN; overall review
CHANGES REQUIRED. Original failure chain and literal DBDEDF fragment are preserved.

## Original findings: actual independent closure

The exact original fragment DBDEDF67A4E252BB9E22E6316FF4A31CED4C9689569F109A7F811D48517073BA
was appended to the latest complete89B0 test prefix; author9/original80 stay intact.
Ordinary AND race92 concrete leaves all actually run/pass, no skip/unfinished.
Original image mutation happens after approval2/digest_changedtrue, but now child0,
targetRead0,bind0. Original real-lease loss also child0,targetRead0,bind0. Healthy
still reaches Begin1/bind1/restore+verify2 children/targetRead2/approval3; no missing-
facility early failure masks a positive. All child/pipe/Stop/Close assertions run.

Round1 read/reconstructed the ENTIRE727-line repair patch against frozen R01 before,
exactly obtaining both current whole files; all changed bodies/new tests read.
Round2 followed actual public Runtime.CheckPhase/ReadPermitBinding/current lease
through Replace331 -> Run525 -> owner concrete Start check, retaining scoped grant,
no public API/mint or direct runner path fallback. Image owner interface420-428
makes approved launch/lifetime explicit; facility acquisition and returned-owned-
error cleanup hold through child/pipe/callback terminal. Actual author9 also pass:
missingfacility/nil/acquireerror, concrete late lease, startedcmd+error realWait and
blocked terminalClose. Borrowed verify ownership checker is not closed by driver;
future root must inject the same actual deployment lease. Same-version own full
Role/rules/profile and original R01 full5/realconsumer/upstream contract remain
valid; stable3 exact hashes unchanged. No future native/root owner assumed complete.

Original RestorePG7 ordinary/race were also independently executed on exact granted
51934/han_b13d_restore with fixed0001-0010, fresh quoted schemas. All7 actualpass in
both modes;14 new schema names retained. Actual new connection/identityStore and
pre-established SOFTWARE source witness are not native backup restore/receipt
proof. Private restore_dsn stayed only runner memory/private stdin, no raw child
stdin/stderr saved, no oldDB/glob/autodrop/service manipulation.

## New HIGH: checked image owner is lost when acquisition callbacks panic

ID RECOVERY-B13D-S414-CR-R02-F001. Primary location recovery_restore.go474;
related485/491 and Preflight499-500, Run508-515. openImage has already opened and
owns the current executable FD at470-474, then invokes external approveImage485
or acquireImage491. Its callers register cleanup only AFTER openImage returns.
A callback panic bypasses return/ownership transfer; neither caller gets the local
owner, nor does driver.Close retain/reach it. There is no acquisition-local
unwind cleanup. The original callback panic propagates, with real FD still live.

Independent TestReviewerS414ImagePanicCleanup has a same-path returned-error
positive and two panic negatives. It calls real public Runtime.Enter -> driver
Prepare/Preflight -> actual opened current image handle, uses sentinel panic
identity, then calls real Runtime.Stop and driver.Close. Ordinary AND race compiled
behavior: returned-error-control PASS, approve-panic FAIL, acquire-panic FAIL.
Both negatives: original_panictrue, Begin0/children0, Stop/Close returned,
deploymentClose1, but File.Read yields n1 (descriptor_closed_by_componentfalse).
The positive returns os.ErrClosed and needs no manual closure. This uses actual
File.Read, not a Windows-specific Stat error classifier or FD-count guess.

After recording each leaked FD, the probe MANUALLY closes that resource so this
review leaves no live own resource. That manual finalization is explicitly logged
and never credited as product cleanup/PASS. No child is created by these panic
cases. Existing native/process-tree gate is unchanged. This is a current local
resource acquisition/unwind defect: even later driver/root Close cannot find the
unpublished owner. It is not the explicitly deferred S416 already-idle handle
lifetime, a future provider omission, or a style preference about defer placement.

Minimal fix: register cleanup where openImage first owns the file, before callbacks;
transfer responsibility exactly once to existing caller cleanup on normal return
(including owner+error). Preserve the ORIGINAL panic while terminating resources
already owned, honoring existing CloseAndJoin/borrowed handle contracts. Do not
swallow panic into nil, call manual test cleanup a fix, invent native facilities,
add a public Runtime authority, or weaken the original92 controls. Full cause,
references and exact re-review requirements are in findings.json.

## Actual execution, scope and STOP

| Reviewer execution | Ordinary | Race |
| --- | ---: | ---: |
| Original80 + author9 + unchanged originalReviewer3 | 92 PASS | 92 PASS |
| Granted original RestorePG | 7 PASS | 7 PASS |
| New panic/error control group | 1 PASS / 2 FAIL | 1 PASS / 2 FAIL |

All selected leaves had actual run/terminal events; no skip/unfinished/compiler
failure/race warning. Six Go commands actually wait returned.64 distinct logged
child PIDs have actual Wait/PipesJoined records across required92/PG7; the panic
group creates none. Every focal original call and resource terminal control joins;
the panic probe closes only its measured leak after observing failure. No pending
own Go/PG/callback. Complete result argv/PIDs/times/exit and raw under checks/;
actual-validation.json carries exact leaves/counts/source/Wait evidence.

On the new real blocker, no old319/global7/fmt/vet/Linuxcompile/extra native/other
panic expansion was started. Author/Main old319 and broad process/SQL evidence
remain separate; no author Green replaces Reviewer execution. Approval/acquisition
panic is established; other new unwind branches such as CloseAndJoin panic were
not independently expanded after this blocker and must be inspected with the repair.
All evidence/artifact/binary copies and new14 schemas retained; no cleanup/archive.
The unchanged PG fixture51934 portability note remains as originalR01 scope note,
not a new blocker, waiver or new environment authorization.

Native OS lock/ACL/safe crossprocess no-follow handoff/approved tool receipts/
process-tree proof, fullS413/rootS416 idle+postproof-failed ownership, realrestore/
E2E/human acceptance stay unknown/closed. No Task/G4/batch/global7/native/root PASS.
**STOP: sole Go/Reviewer RestorePG window RELEASED.** All6 commands truly joined;
source5 entry/exit matches exact frozen mirrors. Return new HIGH to original author,
then original Reviewer rechecks the new exact freeze; no self-repair or SSOT write.
