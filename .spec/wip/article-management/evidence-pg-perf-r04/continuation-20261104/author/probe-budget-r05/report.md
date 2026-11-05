# R05 作者交接：probe budget + CLOCK-R04-F001

**SOURCE / LOCAL / PG STOP**。2026-11-04 16:06:47.304659 +08 最终查询无可见 Go/test 进程，各命令及测试 child 均有真实 wait/exit；Main 可接手。整个步骤没有装载 User DSN、没有执行 PG 查询/DDL/DML，没有创建真实 Tnew、schema 或运行性能入口。不以本次离线证据声称 backend=0、真实 SQL/增长通过或原 finding 独立 CLOSED。

固定 main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；仅改 `test/integration/article/pg_perf_harness_test.go`。最终 H 为 **E97318D68C8BE670DF11A196D469A3D91F5BAC979A1CF65EFDDCBAE4F09DAFC7**。完整五文件哈希/前像摘要/快照/STOP 在 [source-freeze-r05/manifest.json](source-freeze-r05/manifest.json)；W/R/N/docs 四哈希保持原值。精确 [r04→r05 diff](harness-r04-to-r05.patch) 与 [r02→final 累计 diff](harness-r02-to-final.patch) 均已保存，r04 固定 manifest 与五快照另核未漂移。

G0 完整消费 budget-r02（`4B5D2F9A0BE894E2D478ECAB0B6E4661DF72E0BB140FFD0909735EF5D924208A`）、Lead 采用和原 Reviewer verdict；后消费 CLOCK-R04-F001 trace。Role/实施 Skill/规则/go-service 复用本会话有效全文。输入摘要在 [adopted-inputs.json](adopted-inputs.json)；`before/` 为编辑前真实 r04 前像和旧资源摘要。本轮授权是源实施与离线自测，计划 PASS 不是代码或执行 PASS。

| 合同/消费者 | 当前 H 实现与验证边界 |
|---|---|
| 15/5/20 policy，旧缺字段 5s | H:1474。三个毫秒字段全部显式正确且绑定 `resource_probe_adoption_source/resource_probe_adoption_sha256` 才采用；部分/null/空/错误/退役 detection-envelope 拒绝。全缺字段保留原 5s、5s tick，不追溯产生新 20s 语义。原 clock/resource 身份验证保留。 |
| 总机会、发布、终态 | H:1696/1706/1730。请求入口起算，含可取消单飞等待，deadline 取 15s、旧成功 START+20s、真实 owner/调用者更早边界。使用未截断的 `time.Now/time.Time` 单调期限；UTC 仅取证。完整及时成功才更新，失败/取消/late 不刷新；request 的取消 hook 可独立取消 root，hook 都由 collect 停止或等待完成。 |
| root observer 与独立 expiry | H:2106/2131。专用资源 pool，root/diskCtx/h.group；单 observer 与 coalesced tick，不按 phase 结束取消它。独立 expiry 不持采集单飞锁，每次准入也自行核时间。 |
| 初始化、准入、seed/clone、统计 | H:1246/1903/3169/3184。初始准确 SQL 与 pre-DDL 同步 guard 统一入口；unknown 不准入。pool acquire、migration、point/operation 检查资源状态。seed CopyFrom 保留全部行，拆为最多 1000 行有限批并同步核 guard，clone 表边界同步核；不做物理预分配。cache 读取只记录旧观察区间/等待，不续新鲜度或混入业务样本。 |
| D1 与关闭 | 原准确 `SELECT pg_database_size(current_database())`，全库/分档旧账及卷预留不变；触及 cap 停新操作，超限原值保留失败，安全失败取消 root。H:1090 先 cancel，按既有 5s/更早 hard stop join；不赠送 15s。join 后核终态，防收尾错误漏报；非合作保留 inventory 并让当前 owned child exit86，资源 pool/probe 未归还也不进无界 Close。 |
| CLOCK-R04-F001 | H:3711/3737/3785。授权 map 不改；per-harness `resource_actual_baseline` 单列真正校验文件及 receipt 的 path/origin/tier/Bytes/filelen/SHA、授权状态、首登/复用状态和来源 manifest。摘要来自实际参与校验的字节；普通及最终 race 验首登、同 policy 复用、后续绑定 receipt 加载，并通过真实 fingerprint writer 落盘、重新解析核对三 harness。 |

完整命令/版本/起止/PID/真实出口/raw 摘要见各 `result.json`，汇总 [command-index-r05.json](command-index-r05.json)。只有 `current_five_hashes=true` 是最终版；旧成功不冒同版。当前版实际结果：

| 命令目录 | exit / 实际耗时 / 结果 |
|---|---|
| `race-budget-final` | 0 / 33.954s；race×3，243 pass 事件、0 fail/skip，包含子测试。精确受控 0→14.9→15→20→29.9、及时 19 成功分支、队列/父取消/失败、expiry、D1 caps、两类 owner、关闭、F001 及 r04 兼容。20ms/10ms 独立控制测试是明确缩放的离线 fixture，生产字段仍固定 20000/15000ms，不冒充 OS/真实 SQL 证明。 |
| `build-core-final` → `core-race-final` | 0 / 5.812s；0 / 10.906s。当前二进制在新 sandbox 执行原 core race×3，57 pass 事件、0 fail/skip；旧相对 unit-fixtures 落点只在新 sandbox，未写旧目录。 |
| `vet-final / format-final / list-final` | 均 0；1.829s / 0.125s / 4.437s。gofmt diff 空。普通中间运行另保留，最终版全部所选断言在 race 组实际执行。 |

行为 RED `red-policy-r01` exit1/3.656s：旧解析器静默接受只有 timeout 的不完整新预算。`local-evidence-r01` exit1/1.812s 是补强测试误把 readJSON 的 bool 当 error，属于编译失败，已修正，不能算行为 RED。所有中间日志/失败/二进制保留。非合作关闭不只核非零：父测试读取不可变 fail-stop JSON，核其 PID=真实 child、exit86、UNKNOWN、root canceled 和 inventory，再核真实 wait；它仍是预期失败证据，不是资源采集成功。工具链记录 Go1.27.0/windows/amd64；最终二进制摘要及构建版本见 [binary.json](binary.json)。

只读 PG **入口已编译、未执行**：`TestPgPerfR05_ReadOnlySQL`，可按 `natural-three / controlled-six / total-fifteen-failure / earlier-freshness-failure` 精确选子项。复用同源 collector/SQL；不经过 newHarness，不需 policy Tnew，不建库/schema。入口代码在连接前拒绝非授权库，设置 `default_transaction_read_only=on`，连接后将核该设置及 loopback:5433/PG180006/han_perf_s24；本轮没有取得这些现场 PG 结果。必须显式设置已有 `HAN_PG_PERF_EVIDENCE_DIR` 为新增绝对证据目录，禁止默认回写；DSN 仅由未来获准的外层安全注入。自然值与 pg_sleep 注入独立标记，负例保留 `accepted=false` 并真实测试失败，不转成 parent PASS。当前任何调用此入口的 PG 授权都尚未授予，也无结果。

原 global base=1406932671、small 80B/原 Window/SHA、原 5400s window 记录、旧 policy/ledger 哈希均未变，原目录仍只有 small；没有真实 medium 首登或 receipt 写入。枚举候选未恢复。原 30s/10s×3、121/60/tail1 及 phase 主体文本未削减；本次未验证真实 SQL>5<15、真实 SQL timeout/连接/backend归还、PG 增长/leak/九语义、完整校准/small/medium/large/full bench，也未验未来平台。它们均 **NOT RUN/PENDING**；旧批次 NRU 保留，S2.4 不关闭。请原 Reviewer 独立核 combined H 的 budget delta 与 CLOCK-R04-F001，再由 Lead 决定后续只读证据/准入；作者不核销原12项或宣布运行 Gate。

## 仅供后续授权的只读入口 usage（本轮未执行）

仍需原 Reviewer 对固定 combined R05 的独立结论、Lead 对实现的采用、Main 的 Manifest checks 与明确只读 PG/LOCAL 窗口。当前 SOURCE/LOCAL STOP、PG 关闭不变。若需修源，先报本 freeze 的适用边界，另建 revision；不边审边替换 E97318D6 版本。只读入口本身不要求新源码补丁，不创建/读取新的执行 policy/Tnew，也不取得 Growth/leak/九语义/完整性能资格。

未来获准后，从固定仓库根目录调用已保存的 `tools/frozen.test.exe`（SHA `C4670AA1471DAE47847EED0E28234111D4943899C0377B752C69C630162C2DB2`）。每个子项独立命令和新增 evidence root，外层保存完整 argv/PID、开始/结束/实际 wait exit、stdout/stderr、前后五 source hash及 binary hash。以下为命令模板，不是已经执行的命令；Main 应以已批准的外层 runner 捕获这些实物，不写回本 freeze/raw：

```powershell
$probeBinary = Join-Path (Get-Location) '.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/author/probe-budget-r05/tools/frozen.test.exe'
$probeMode = 'natural-three' # 每次仅选下表一个子项
$probeEvidence = Join-Path (Get-Location) ('.spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/lead/readonly-r05-' + [guid]::NewGuid().ToString('N'))
$probePreviousDsn = [Environment]::GetEnvironmentVariable('HAN_TEST_POSTGRES_DSN','Process')
$probePreviousEvidence = [Environment]::GetEnvironmentVariable('HAN_PG_PERF_EVIDENCE_DIR','Process')
try {
    $probeDsn = [Environment]::GetEnvironmentVariable('HAN_TEST_POSTGRES_DSN','User')
    if ([string]::IsNullOrWhiteSpace($probeDsn)) { throw '同名 User DSN 为空；停止，不猜凭据' }
    [Environment]::SetEnvironmentVariable('HAN_TEST_POSTGRES_DSN',$probeDsn,'Process')
    [Environment]::SetEnvironmentVariable('HAN_PG_PERF_EVIDENCE_DIR',$probeEvidence,'Process')
    & $probeBinary ('-test.run=^TestPgPerfR05_ReadOnlySQL$/^' + [regex]::Escape($probeMode) + '$') '-test.count=1' '-test.timeout=2m' '-test.v'
    $probeActualExit = $LASTEXITCODE # 外层必须保存，负例不改写成0
} finally {
    [Environment]::SetEnvironmentVariable('HAN_TEST_POSTGRES_DSN',$probePreviousDsn,'Process')
    [Environment]::SetEnvironmentVariable('HAN_PG_PERF_EVIDENCE_DIR',$probePreviousEvidence,'Process')
    $probeDsn = $null
}
```

不得打印 DSN、凭据或环境全集；模板只临时继承到本次进程/子进程，不改 User 配置。证据根目录中的 GUID 只是只读诊断标识，不是 Tnew。测试内每子项 owner 最多90s，外层测试2min只作失败上限；这不改变正式性能门或增加性能预算。若沿用 test2json，保持同一精确 `-test.run/-test.count/-test.timeout` 参数及真实 child 退出链。

| `probeMode` | 到达路径与判定要求 |
|---|---|
| `natural-three` | 三次原准确 SQL；分别保留自然区间、bytes、Acquire/SQL/volume/总机会，期望 exit0。不能把相同 bytes 当低延迟证明。 |
| `controlled-six` | 同次机会内先执行明确注入的 `pg_sleep(6)` 再走原 size collector；总观察须实际 >5s 且 <15s，期望 exit0。注入与自然样本分账，不称自然 SQL 本身超过5s。 |
| `total-fifteen-failure` | 同一 collector 内注入 `pg_sleep(16)`；应在请求总15s边界失败，期望真实 exit1、sample `accepted=false/injection_reached=true`。失败通常在注入阶段，不能称 size SQL 本体已运行并超时。 |
| `earlier-freshness-failure` | 先完整自然观察，再单列7s SQL年龄阶段，然后16s注入；第二次有效 deadline 应受首次 START+20s 更早限制，期望 exit1及完整到达/取消证据。若首采过慢、年龄阶段先过期、cap/身份/卷/权限等前置先失败，目标路径记 NOT RUN/阻断，不能仅按 exit1认定该负例成立。 |

每个实际新增 `readonly-<mode>-<pid>-<timestamp>/` 内保存 sample、diagnosis、expiry、resources 等；核实际截止、错误类型、`injection_reached`、AcquiredConns、group/goroutine和关闭结果。现有 closeProbe 查询排除探针自身，单独的 residual=0不能证明所有 owned backend=0；获准外层仍须在 child 真正结束后按本次唯一 application/run范围补最终只读 backend 探针。环境/初始化失败、不同故障、late 或关闭未知都不能当预期资源成功或负例达标。
