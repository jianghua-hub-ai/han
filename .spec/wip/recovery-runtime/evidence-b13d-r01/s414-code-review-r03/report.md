# S414 independent Code Review R03 ? STOP / new MEDIUM OPEN

Original HIGH `RECOVERY-B13D-S414-CR-R02-F001` is independently **CLOSED** on the exact current source: original approve/acquire panic cases pass in normal and race, preserve original panic identity, and observe `os.ErrClosed` with `manual_probe_close=false`. Original R01 F001/F002 remain CLOSED: image/lease negative controls launch zero children; healthy control actually launches and joins restore+verify, binds once. This is not an overall component PASS: new MEDIUM `RECOVERY-B13D-S414-CR-R03-F001` remains OPEN.

Two rounds: reused own complete original five-source/real Runtime review and R03 entire 214-line static delta; independently checked acquisition-local ownership, transfer/error/panic and single-close paths. Then traced actual public Runtime Enter/Begin to Run/image lease/cleanup/Stop and executed exact original six plus the compound exception sequence. No private phase producer or shadow algorithm was used.

| Current independent execution | exit | actual leaves |
|---|---:|---:|
| core102-normal | 0 | 102 PASS |
| core102-race | 0 | 102 PASS |
| compound-normal | 1 | 1 behavior FAIL |
| compound-race | 1 | 1 behavior FAIL |

All required leaves actually ran, with zero skip/unfinished/compiler failure/race warning. Current102 retains all old92 names, seven author unwind leaves and exact original panic3; ordinary/race leaf sets match. Full commands and raw streams are in `checks/*`; `actual-validation.json` and `leaf-retention.json` contain exact paths/counts/outputs. Single job, GOMAXPROCS=1/-p1, OS/path allowlist, offline/local Go and empty ordinary test DSN were actually applied. Only test source was overlaid: the byte-exact entire final57946 test prefix plus unchanged DBDEDF/24D710 fragments; compound adds one separate frozen fragment. No author assertion/regression was replaced.

## [MEDIUM] Terminal cleanup panic replaces the causal handoff panic

ID: `RECOVERY-B13D-S414-CR-R03-F001`. Location: `cmd/han-server/recovery_restore.go:520-526`, with propagated terminal panic from `closeAndJoin:442-444`.

After Begin1 and the actual second image acquisition, the handoff callback panics with marker A. The image owner's real software lease first completes CloseAndJoin, then raises marker B. Run's cleanup defer allows B to supersede A. Both modes record `primary_preserved=false cleanup_caught=true lease_terminal=true owner_closes=1 checked_fd_closed=true manual_probe_close=false Begin=1 children=0 Stop_ok=true driverClose_ok=true deploymentClose=1`.

The original TP Root row839 requires the main error to remain primary and cleanup to be secondary. This is an error-cause defect in the already-supported exception/owner flow, not a demand to select a native adapter or implement future Root. The resource terminal/FD/single-close assertions succeed, so this finding does not claim a resource leak, unsafe release or restore side effect. Severity is MEDIUM because the failed action's primary panic is hidden from a recovering caller and its root-cause classification/diagnosis is corrupted.

Minimum repair direction: retain any already-active action panic through actual owner cleanup, keep the cleanup failure privately secondary, preserve cleanup-only panic when no primary exists, and keep the R03 terminal FD cleanup/exactly-once transfer. Original author owns the fix. Do not weaken old or new probes. Exact independent normal/race evidence and the unchanged frozen compound fragment are indexed in `finding.json`.

On this concrete finding, unrelated expansion stopped. **PG7 normal/race NOTRUN**, private resource/DSN not read, no SQL or new schema; old R02 PG7 evidence remains historical and does not count as R03. Old319, vet/fmt/Linux/global7 were not rerun. No overall component/Task/G4/global/native/root/E2E acceptance. Following repair, rerun affected exception controls/current102 and separately authorized PG7 on the new exact freeze; retain original finding chain.

All four own Go Popen.wait calls returned (0/0/1/1). Raw records contain 70 child Wait/pipe observations for 44 distinct test-child PIDs, all waited/pipes_joined true. Compound launched zero children; actual lease terminal/FD close/public Stop/driver Close checks returned. No pending own execution session or callback owner is claimed outstanding; no other-lane PID/Temp was inspected, and no all-host/native process-tree join claim is made. Go/RestorePG window is released with this STOP.

Exact source5 entry/exit hashes are in `source5-entry-exit.json` (all stable); complete own provenance/readset and prior reports are in `final-readset.json`. Native facility/ACL/no-follow safe handoff/real native receipt and process-tree proof remain gated; long-idle RootS416 owner, fullS413/real restore/provider/E2E and human acceptance remain unverified. No source/SSOT/Git writes, production overlay, cleanup, network install or subdelegation occurred. All evidence and fixtures retained.

| frozen source | entry = exit SHA256 |
|---|---|
| cmd/han-server/recovery_restore.go | FDC2792E93A638F45917C097D8CAE6DEBC22918159C1495F7D31148987534423 |
| cmd/han-server/recovery_restore_test.go | 57946F39EAE83C8273A66C15E62C3AAA83E0CB47FA4C2CD6A03A967A76E41259 |
| cmd/han-server/recovery_exclusive.go | D74537835601F20CA1B270AB21EED0E1179E61CDB7B59B0FFDB98E052E095B9B |
| cmd/han-server/recovery_exclusive_test.go | 7984D8EE2E6295EF6BDC490B7FA20C3A322863097B02934B3E71180D8D8A40EF |
| cmd/han-server/recovery_restore_integration_test.go | AFE7AE9162BFFA06672B23266E9478BAB8DCA4007ACCAE36F674F3D6FFB983FD |
