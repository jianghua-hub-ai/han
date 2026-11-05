# B13E shared identity timestamp correction - author STOP

Original Zeno/add-backend-dev, existing S4.10 shared consumer owner. RECOVERY-B13E-G4-R01-F001 remains OPEN pending original Dirac independent R02 and Main. No new Task/design/API or author G4/finding closure.

SOURCE3 was frozen and sole LOCAL/identityPG RELEASED in release.json before this packet text/manifest work. Pending owned work0. Main may run independent Review concurrently with this Temp-only packaging. Fixed Han/main/HEAD6da8abbceb0fa197364810a92a8fa2d97b38ddca.

## Final object and smallest production change

- `internal/identity/recovery_execution.go`: `57118E4DDCC4097516403869F1D7D61F1A977DAA9C41895BF989ABAC3DE9E0E4`
- `internal/identity/recovery_execution_test.go`: `8953835A996CAD8DA93F1AAD8F6C41AEAB445E109FC5BBA278F15F38305B09E4`
- `internal/identity/recovery_execution_postgres_test.go`: `DB37374C4348CE044AAD7AE534E16DABBCB51541EC0B4B9A9C2244206A83250D`

finalization previously used reflect.DeepEqual on optional deleted_at pointers. Audit JSON and PostgreSQL JSON may represent the same instant with different time.Location objects. The sole production change computes nil-safe equality with time.Equal and retains the existing role/status/member/count/source checks. It does not round/truncate timestamps, normalize a fixture to avoid UTC, loosen all DeepEquals or alter any API/phase/Store authority. Source-before-after.json and diff/source3-full.diff give full bytes/scale; original before bytes and final snapshots are retained.

The unit regression calls the real finalization consumer using explicit complete read-port fixtures and verifies owned projection closure: both nil, UTC/Z/zero/nonzero-offset same instant accept; a real microsecond, nil mismatch, role/status, wrong member and missing account reject. The new P regression creates actual UTC/+00:00/nonzero session contexts, enters the real public Runtime recovery phase and calls actual Finalize/PrepareTrust. Default existing fixture behavior and clock remain unchanged; no timezone workaround is used as the fix.

## Failure evidence and current verification

Original G4 Red is the frozen Dirac report/two actual normal runs in red/: original Install -> same Store Bind -> Enter -> Finalize -> Prepare failed while time.Equal=true/DeepEqual=false. Initial unrelated compile diagnostic stays classified as such in that original report. The exact latest composition probe SHA86FB92AB45C99611D2B3BADC977E4DC1FE2C61E6248FC27DD42B1CF3632ED0A0 is copied byte-exact under oracles/ and used unchanged as a new virtual test-file overlay, production overlay0.

An additional compiled author Red on unchanged production6D80 executes11 real-finalization cases:10PASS/1 behaviorFAIL (nonzero-offset same instant). After the small production fix, the same11 names/expectations PASS. No compiler/setup/timeout/Skip was counted as Red. tdd-proof.json records the exact labels/marker.

Final selected ordinary/race results are equal by leaf name/result, all PASS/no Skip/unfinished/race:

| Exact scope | Normal | race |
|---|---:|---:|
| S410 unit incl. new11 | 88 | 88 |
| maintenance Auth unit | 32 | 32 |
| Installation Store binding unit | 2 | 2 |
| S410 affected Finalize/Trust/Audit/ACK/Panic PG incl. three timezone cases | 9 | 9 |
| actual maintenance Auth PG | 42 | 42 |
| original real Install -> sameStore -> Enter -> Auth composition | 2 | 2 |

The native unit aggregate is122, not the earlier commentary arithmetic120. No case or result changed to correct that total. The original composed E-X01 negative stays zero Auth writes; E-X02 now reaches actual Login/Resolve/Change/G+1, old Cookie refusal, ordinary refusal and both Audits. Both modes log the original endpoint marker. Current source3 fmt and native/integration static vet also pass. All selected Green result headers bind the exact current complete three SHA. No mechanical full-history or global seven rerun; unchanged old component evidence is not relabelled as this author's execution.

## Kept diagnostics and test-resource adaptation

First actual cross Go exit0/twoPASS had an author JSON postprocessor error because S413 physical receipts use a text format, unlike S410 JSON. Only the parser was repaired and existing unchanged raw re-read; no Go rerun was used to hide it. A one-off Python default-GBK read also failed, then explicit UTF-8 was used. postprocess-diagnostics.json preserves both facts.

First Auth-PG attempt hit the fixture guard requiring a private own D Temp signal before any business assertion/DDL. That failed raw remains a setup diagnostic, not a product Red. The runner then supplied the existing required B13E-S411 signal from Main's named synthetic identity resource to an own private D file; no source guard or assertion changed. Current Auth normal/race42 pass. The private signal/DSN is outside this packet and is not printed.

All PG jobs derive a fresh overlay from the CURRENT P source, adapting only the four named B13D->B13E test DB/user/schema literals. Every original literal matches once and inverse restoration is byte-exact. Original assertions, SQL algorithms and all new tests remain in the compiled prefix; overlay-*.json records current source/overlay SHA. The composition virtual probe is unchanged. No stale old-test overlay/cache replaces the new test source.

## Actual physical records and release

15 actual Go/gofmt commands and7 actual psql commands all have parent Wait, kernel Job-zero and both real pipe EOF/pump joins; all owned descendants are confirmed complete by Job-zero. Kernel accounting totals309 processes;49 optional observed PID values are not an exhaustive child/handle census. Runtime.Stop/callback and pool.Close acquired0 are asserted/logged by the original actual composition and Auth fixtures; Rows/Tx/pool resources return before test/command completion. See joins-summary.json and each raw/result.json. No claim about other Host processes.

106 exact schemas from this author's raw were physically re-read in7 batches of at most16, each its own bounded read-only RR transaction. Names equal the raw-created union; physical destination is PG180006/127.0.0.1:51935/han_b13e_identity, userhan_b13e and the exact new Main D data_directory. S413 schemas have24 tables, S410/Auth schemas23; actual run/Audit/Session counts and namespace presence are recorded. Every SQL command exits0 and joins. No historical-prefix sweep, DROP/cleanup or old DB reuse. All raw/schema/data/cache are retained; Main owns the cluster. physical-schema-receipt.json and physical/batch-* contain exact SQL and actual output.

The original17 batch sources plus6 documents remain exact at release (guard.json); no Git/stage/commit/push, other source/SSOT, original evidence, API/design/DDL source, cleanup/delete/move/archive or subdelegation write. Main's new whole-tree/frozen26 acceptance remains separate.

## Boundary

This proves the bounded software/isolated PostgreSQL seam, not real provider/ACL/approval subject, SMTP/native restore, full production Root/S4.16/Case18/E2E, capacity/P95 or human acceptance. The original synthetic external ports remain explicit fixtures. Existing finding/task/G4 decisions belong to the original independent reviewer/Main. Source/Go/PG stay stopped; only final Temp evidence sealing occurred after release.
