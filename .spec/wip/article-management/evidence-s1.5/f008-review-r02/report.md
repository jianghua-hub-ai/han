# S1.5 `S15-RE-F008` 更正包 r02：非作者独立复审

- 性质：非作者独立复审；AI 代理评审，**非真人签字**。
- 复审对象：`.spec/wip/article-management/evidence-s1.5/f008-allow-assertions-r02/`。
- 上游裁定：`review-code-s1.5-f008-producer-r01.md` 判 `S15-RE-F008` NOT CLOSED（仅证据记账：叶子数 33/实际 34、耗时与日志不符、缺 gofmt 日志）；代码修复成立。
- 工作区：`D:\AID\VSCode\Codex\Han`；分支 `main`；`HEAD` `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 复审时点：2026-11-05。
- 授权边界：只读；允许一次 targeted normal、一次 race、`gofmt -l` 与 `go vet`；唯一写入为本目录 `report.md`、`verdict.md`。
- 执行方式：按用户明确要求由主会话直接复审，未 spawn 子 Agent，因此没有 `add-reviewer` 子 Agent 的 actualID；本文件是主会话的独立复核记录，不是作者自评。
- 权威边界：本轮只关闭 `S15-RE-F008` 的证据记账问题；不宣称 S1.5 整卡关闭，不替代人的业务接收、独立评审签字或后续授权。

## 1. 结论摘要

`S15-RE-F008` = **CLOSED（有限定）**。

R01 的三项记账阻断均已消除：normal 日志实为 34 个叶子且 0 FAIL/0 SKIP；1.326s/3.027s/39.076s 与保存日志逐字一致；`gofmt.log` 已保存为空输出，本次独立复跑也为空。`Allow` 精确断言与 `mainOrphanAllow` 映射同 `main.go` 的 production `allow` 字面量一致；`index.json` 记录的全部源文件与日志 SHA256/字节均与实际复算一致。

保留限制：`gofmt.log` 为 0 B，文件自身不能证明命令来源，需结合索引命令/exit、文件时间序与本次独立复跑理解；race 未能在本机复跑（无 CGO/C 编译器），仅审阅作者保存的 race 日志；full-package 未复跑，仅审阅 32 B 的包级 `ok` 日志。

## 2. 五点核定

### 2.1 34 叶子 / 0 FAIL / 0 SKIP 是否与 normal 日志一致

**结论：一致。**

`f008-targeted-normal.log` 实际统计：

- subtest `=== RUN`：34
- subtest `--- PASS`：34
- `FAIL`：0
- `SKIP`：0
- 顶层 `--- PASS`：1
- 末行：`ok  	han/cmd/han-server	1.326s`

测试循环本身也能独立算出 34：5 个 action（list/detail/publish/transfer/delete）× 8 个 variant = 40，扣除 list/detail 各 3 个不适用的 origin/csrf/precondition（共 6），即 34。`index.json` 的 `leaves.targeted_normal_subtest_pass=34`、`fail=0`、`skip=0` 与日志一致；`report.md` 的“34 个叶子 PASS、0 FAIL、0 SKIP”也一致。

### 2.2 1.326s / 3.027s / 39.076s 是否逐字取自日志

**结论：是，逐字一致。**

| 来源 | 日志中的原文 |
|---|---|
| `f008-targeted-normal.log` 末行 | `ok  	han/cmd/han-server	1.326s` |
| `f008-targeted-race.log` 末行 | `ok  	han/cmd/han-server	3.027s` |
| `full-package-normal.log` 全文 | `ok  	han/cmd/han-server	39.076s` |

`report.md:44-46` 与 `index.json` 的 `runs[*].duration` 分别写作 `1.326s`、`3.027s`、`39.076s`，与保存日志逐字相同，没有 r01 中“摘要取自另一次执行”的不一致。

文件时间序也支持“单轮执行”的自述：normal 日志 mtime `2026-10-31T23:21:20.857`，race `23:21:31.818`，full `23:22:12.637`，go-vet `23:22:14.337`，gofmt `23:22:14.457`，`index.json` `23:22:32.599`；与 1.326s、3.027s、39.076s 的先后和量级相容。该时间序只是旁证，不是密码学证明。

本次独立复跑 normal 的命令报告 `ok ... 1.414s`（wall 3.828s），与保存日志的 1.326s 属不同次执行的正常波动，不构成 r02 的记账不符；r02 的 1.326s 确实取自其保存日志。

### 2.3 `gofmt.log` 是否真为保存的 gofmt 运行证据

**结论：可接受为“保存的空输出日志”，但属有限证据。**

- `gofmt.log` 为 **0 B**，SHA256 为 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`（空文件哈希）。
- `index.json` 记录命令 `gofmt -l cmd/han-server/main_test.go`、exit `0`、`log: gofmt.log`、上述 SHA256 与 0 B。
- 本次独立执行 `gofmt -l cmd/han-server/main_test.go`：exit `0`、耗时 `0.148s`、输出 0 行，复现了“格式干净”的状态。
- 0 B 正是 `gofmt -l` 在文件已格式化时的自然输出；文件 mtime `23:22:14.457` 位于 full 测试结束 `23:22:12.637` 之后、`index.json` 生成 `23:22:32.599` 之前，且与 `go-vet.log` 同批。
- 限制：0 B 文件内部没有命令行、时间戳或 exit code，单凭文件本身不能证明它一定由 gofmt 生成；只能结合索引元数据、时间序和独立复跑接受。若要求取证级 provenance，应补充带命令/退出码的 wrapper transcript；这不是 F008 的代码缺陷，不阻断本轮 CLOSED。

### 2.4 `Allow` 精确断言与 `mainOrphanAllow` 映射是否与 `main.go` 的 allow 字面量一致

**结论：一致，且断言具备负向有效性。**

`cmd/han-server/main_test.go`：

- `:3401-3408` 仅在 `variant == "method"` 分支执行：
  - `:3402` 精确比较 `response.Header().Get("Allow")` 与 `mainOrphanAllow(t, action)`，不是宽松包含或只比状态码。
  - `:3405-3406` 同时断言 `events` 精确等于 `[]string{"trace"}` 且 `resolver.calls == 0`，失败信息明确。
- `mainOrphanAllow` 在 `:3411-3424`：
  - `list` → `GET`
  - `detail`、`delete` → `GET, DELETE`
  - `publish`、`transfer` → `POST`
  - 未知 action → `t.Fatalf`（不再静默返回 `POST`）

`cmd/han-server/main.go` 的 production `allow` 字面量：

- `:3613` list → `GET`
- `:3615` publish → `POST`
- `:3617` transfer → `POST`
- `:3653` detail/delete → `GET, DELETE`

四处与 helper 逐字对应；`detail` 与 `delete` 共用同一路径和同一 `allow`，不是映射遗漏。`:3668-3675` 在 `methodAllowed == false` 时先设置 `Allow` 并立即 `WriteError` 返回，早于 `probe` 与 `handler`；trace seam 在 `:3700-3706` 包裹路由。因此 `events == ["trace"]` 与 `resolver.calls == 0` 能证明 405 分支未进入 principal resolver、安全链或业务 handler。`list`/`publish`/`transfer` 的期望值互异，若断言退化或写错，至少一组会失败；未知 action 的 `t.Fatalf` 也避免了默认值静默通过。

### 2.5 `index.json` 记录的各文件 SHA256 是否与实际一致

**结论：全部一致。**

| 对象 | 给定 / 索引值 | 实际复算 | 字节 | 结果 |
|---|---|---|---|---|
| `index.json` | `410A65ADD91FCDDD3D30E54090FC0EB64890CC73990E0F55C88B13DEAC0FCC6E` | 同左 | 5564 | 一致 |
| `report.md` | `400E593A7A6B8BDEEA034731375B21AEFC283FA2E74818001C5B0393D5C63681` | 同左 | 5808 | 一致 |
| `f008-targeted-normal.log` | `309047C765243AF0D59A6B83FC14F5F2CCE02A9EFBFD6C60A8241484738E19A9` | 同左 | 7036 | 一致 |
| `f008-targeted-race.log` | `600C463A42C40AB3466A0B3E12AF425CAD9163BAD1B7109D78616925F9ACD726` | 同左 | 7194 | 一致 |
| `full-package-normal.log` | `C4B92BBD6064061F8D155CC942BC7A0226155FFB61C0A7734B3DE5B8EB937EDD` | 同左 | 32 | 一致 |
| `go-vet.log` | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` | 同左 | 0 | 一致 |
| `gofmt.log` | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` | 同左 | 0 | 一致 |
| `cmd/han-server/main_test.go` | `3720652EF4774011732455D4901ADFC7D98096F0BECED4F8479F73819F9A7ECC` | 同左 | 400312 | 一致 |
| `cmd/han-server/main.go` | `6D5373CA1535C41A5FA340D2B75834EB7E15AEEE39ABF9928BD0E0A210CF8727` | 同左 | 173317 | 一致 |

`index.json` 自身只登记源文件和 run 日志，未自登记 `report.md`/`index.json`；上表对这两份文件使用用户给定的对象哈希核对，其余全部按索引登记值逐项复算。执行测试后再次复算 `main_test.go` 与 `main.go`，哈希和字节均未变化，支持“测试运行没有改写被测源码”的自述。

## 3. 实跑记录

| 命令 | exit | wall / 命令报告耗时 | 结果 |
|---|---|---|---|
| `go test ./cmd/han-server -run '^TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity$' -count=1 -v` | `0` | wall `3.828s`；go 报告 `1.414s` | 34 个 subtest PASS；0 FAIL；0 SKIP |
| `go test -race ./cmd/han-server -run '^(TestOrphanDispositionProductionRoutes_RealHandlersEnforceSecurity|TestOrphanDispositionProductionRoutes_ExactPathsAndSharedAdapter)$' -count=1 -v` | **NOT RUN** | 尝试 `0.093s` 后启动失败，exit `2` | `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`；本机 `go env CGO_ENABLED=0`，且 `gcc`/`clang`/`zig` 均不存在，未进入测试执行 |
| `gofmt -l cmd/han-server/main_test.go` | `0` | `0.148s` | 输出 0 行 |
| `go vet ./cmd/han-server` | `0` | `1.264s` | 输出 0 行 |

race 的保存日志另行审阅：`f008-targeted-race.log` 有 34 个目标 subtest `--- PASS`、2 个顶层 `--- PASS`（目标测试与 `ExactPathsAndSharedAdapter`）、0 FAIL、0 SKIP，末行 `ok ... 3.027s`。该日志支持作者的 race 结论，但本机未能独立复跑 race。未运行 full-package（不在允许范围），仅审阅其 32 B 日志。

## 4. 发现与剩余风险

本轮没有阻断 `S15-RE-F008` 的 finding。剩余为证据强度和范围限制，不是代码缺陷：

1. `gofmt.log` 0 B 不能自证 provenance；接受依据是索引命令/exit、mtime 顺序与本次独立空输出复跑。
2. race 未在本机复跑；保存日志有逐叶子记录，但日志本身不打印 `-race` 标志，`-race` 的事实依赖索引/报告记录。
3. 保存日志没有命令行、时间戳或 exit code 头；单轮执行由文件 mtime 旁证。
4. `full-package-normal.log` 只有包级 `ok`，没有逐测试明细；full-package 未复跑。
5. `cmd/` 在 Git 中为 untracked，无法用提交 diff 固定基线；本轮以 SHA256/字节和当前文件内容固定对象。
6. 工作树本身不是 clean 状态（存在大量预存修改与 untracked 文件）；本轮未触碰这些内容，也不据此评价 F008 之外的变更。
7. 本轮只复审 `S15-RE-F008` 的测试侧记账；不覆盖真实 PostgreSQL/Session/SQL、Journal/旧备份、Case18 全局恢复隔离、应用角色 ACL 或部署 E2E。
8. 本轮为 AI 代理评审，非真人签字；不替代人的业务接收、独立评审签字或后续发布授权。

## 5. 写入与只读声明

- 本轮未修改任何源码、测试、tasks、Plan、ADR 或既有证据。
- 未执行 Git 写操作，未删除文件，未再 spawn 子 Agent。
- 工作区唯一新增写入为本目录下的 `report.md` 与 `verdict.md`。
- 复审对象在复核前后的哈希保持一致；测试运行只使用 Go 缓存/临时目录，没有改写工作区源文件。

## 6. 最终裁定

`S15-RE-F008`：**CLOSED**。R01 的叶子计数、耗时记账与 gofmt 日志三项阻断已消除；`Allow`/事件断言与 production 字面量一致；索引哈希全部可复核。保留上述证据强度与范围限制。

---

AI 代理评审，非真人签字。本裁定不宣称 S1.5 整卡关闭，也不替代人的验收或后续授权。