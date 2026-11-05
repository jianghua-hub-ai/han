# S4.4 / ISSUE-article-management-0001 — fixed180 r02 execution

**Stopped. Authorized integration list/normal/race all completed successfully. No repository source/test changes, Git actions, cleanup, independent CodeReview or Issue/task closure.**

| Run | Actual result |
|---|---|
| integration list / tag compilation | exit0; exactly TestArticleReplayPostgres; no DSN injected, no test body/PG execution |
| normal, count1, timeout8m | exit0; 24 run nodes / 24 pass nodes / 16 terminal leaves; fail0, skip0, missing0; package elapsed13.362s |
| race, count1, timeout8m | exit0; 24 run nodes / 24 pass nodes / 16 terminal leaves; fail0, skip0, missing0; package elapsed15.536s |

The four groups/thirteen required named mappings expand to the actual sixteen terminal leaves in both runs. Each mode has sixteen different committed/retained schemas, for thirty-two distinct schemas total. Each case logged endpoint loopback5433/version180006/identity preflight before CREATE, committed CREATE before the separate full0001..0009 migration transaction, and observer/pool/workers shutdown with acquired0. No DROP/RemoveAll was executed.

Frozen manifest: SHA256 `18270024690941D7A20294FE78C56696FCFD8B19414B07823ED3E5E8368D2071`.

Unchanged sole test: SHA256 `B65FAE92A2AE95E291C8CB2CC136769E22209C4E888546CEDF43CC68727F659A`.

Executed runner: SHA256 `C3052EF1E1BE921F63CA0B4C1F0457CF8935058B3D7089E877F82E43C46AA67D`.

All180 live paths matched Lead's flat manifest before/after each run and at final collation. The complete Lead snapshot and overlay158 were independently verified. Execution used the approved direct Go path with the full180 guards; the overlay was not passed to Go. Exact native Go argv and all raw hashes are in each mode's argv.json/result.json and EXECUTION-STOPPED.json.

## Runtime evidence

`case-evidence-normal.json` and `case-evidence-race.json` index every leaf, retained directory, committed schema and all retained file hashes. Each mode contains29 Ready wire/independent SQL/fresh Facts records (58 total), including the exact execution partitions, ref/digest/completed_at, current/input_digest, operation/audit rows and immutable xmin/ctid images. Python cross-checks the retained SQL fields against the actual public Ready; canonical encoding/digest and strict decoder remain the shared Go authority.

- Counts: real3/0, precompleted0/3 and mixed2/1; zero-replay repeat/rebuild with stable complete Ready and row images.
- Partial failure: one real deletion/audit committed, two targets still present, evidence0; full retry binds actual2/1 only.
- Crash: retained child stdout contains the actual post-realReplay3/0/prePublish handshake. Parent SQL shows all physical completion and no evidence before exit; the fixed executed parent assertions verify native nonzero exit23, output/process join and child backend0. First retry evidence is actual0/3, then stable.
- ACK: actual PGStore.Publish commits and an independent backend observes it before the wrapper drops ACK; executed assertions verify ReadCurrent first, no Fail/replay/second Begin/Publish/ref/clock, one durable2/1 evidence. No real network-fault claim.
- Winner/lagger: retained execution records contain real2/1 and real0/3; final public readback references immutable winner2/1 with no extra evidence/write.
- Operation/Audit: separate legal real fact changes; stale then actual0/3 republish/current switch, old immutable row retained.
- Refresh: retained before/new-owner-current SQL checkpoints demonstrate legal generation/current advance; fixed executed assertions verify actual independent NOWAIT, pool availability, old Publish zero-write and old ReadCurrent stale after the barrier. No cross-source atomic snapshot claim.
- A→B→A: final real execution0/3, chosen originalA2/1 with the same ref/digest/time; only two evidence rows.
- Legacy: synthetic legal gen0/tokenNULL compatibility preimage; actual public read/reuse zero-write assertions pass; drift then gen1.
- Ambiguity: synthetic legal sameFacts2/1 and0/3 histories, currentB; actual stale read and replay/publication reject with failed/evidence_unreadable, three immutable rows preserved and no currentA.

ACK/order/barrier/NOWAIT and exit23 outcomes are supported by the fixed tests' executed assertions plus the retained images/marker; they were not additionally printed as a standalone event trace. No tokens/DSN/secrets were logged.

## Runner correction and retained r01 failure

r01 remains untouched: fixed runner95 exited natively2 before Go/DSN/schema because its enumeration accidentally counted two noninput READMEs (182 instead180). The exact argv/raw stdout/raw stderr/flat180 and runner95 copy remain under ../execution-r01/.

Lead authorized the precise mechanical correction: base go.mod/go.sum/migrations/README.md, then recursive .go/.sql/.json/.mod/.sum only in cmd/internal/test/migrations/testdata. `runner95-to-r02.diff`, `approved_runner.fixed.py.txt` and `runner-change-and-inputs.json` preserve the exact change/new SHA. Offline child-only GOTOOLCHAIN=local/GOPROXY=off/GOSUMDB=off/GOTELEMETRY=off, list-no-DSN, normal/race User-HKCU process injection and all path/hash/result guards remain.

This PASS covers only the authorized controlled-public-Journal/real Finalizer/PG/FileInventory/Store/Provider/decoder subset. No real FileJournal/HMAC/POSIX/Unix, actual old backup, double-namespace/MarkConsumed or global recovery reopening PASS. S4.4 fullclose, CodeReview, Issue acceptance, Case18/S8.2 and other existing gates remain separate.
