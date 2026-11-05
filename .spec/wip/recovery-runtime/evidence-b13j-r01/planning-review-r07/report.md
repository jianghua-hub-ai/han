# B13J 规划评审 r07 — r03 微修窄口径确认

- 角色：add-reviewer（非作者）。只写本目录两个文件；未改 `tasks.md`、Plan、`framework.md`、作者证据或 r06 证据。
- 对象：登记轮 r03 微修（`tasks.md` S4.24b 卡 5 个 hunk），只核 `B13J-R06-F001`/`F002` 是否 CLOSED 与是否引入新不一致。
- **verdict：`PASS WITH MINOR`**
- 边界：不构成实施授权；不登记或推进任何卡；不解除 `blocked-upstream`；非人的签收。

## 1. 结论

两条 R06 finding 均 **CLOSED**，5 个 hunk 全部落在 S4.24b 卡内，未发现新的阻断项或新的不一致。新增 0 条 finding，仅 4 条 observation（均不阻塞，见 §4）。按约定口径给 `PASS WITH MINOR`。

## 2. 输入实测

| 文件 | 字节 | SHA256 | 结果 |
|---|---:|---|---|
| `.add/project/knowledge/tasks/tasks.md` | 857169 | `B5AC1D1D5A57A94872511ABC6B7F2F73DFBD22FB37EC5D460B5D5FDADC1EBA4F` | 与 Lead 一致；3226 行 |
| `.add/project/knowledge/plans/ir-01-content-community-batches.md` | 504671 | `EC24892E46D5CB1C668F8F7E13443BC60F64F6A53A82812DBAEA30C79D9D37B9` | 本轮未改（与 r06 相同） |
| `.add/project/knowledge/decisions/framework.md` | 182488 | `39BD3A797B0FA2F5CDF944ACCF4129BB0B7F7A23FABC743B7E5CCC9F7B9BD704` | 未改 |
| `registration-round-r01/author-report-r02.md` | 11370 | `592376924EA7551244D1D7D8F853CE0CF2306A594EB0021A9FB5B48291635410` | 已追加 §7 r03 微修 |
| `registration-round-r01/author-index-r02.json` | 13868 | `E11CEC80B511B50AC390A1EDD08AE743B6A391B35E4A51A025CBF0E50E0E03FC` | 已追加 `r03_microfix`；JSON 合法 |
| `planning-review-r06/report.md` | 18374 | `6E00D8B576F3C0B01442DEE6341753EB7A5B2155B8D7E6342FA7FFC43D611291` | 未改 |
| `planning-review-r06/index.json` | 22680 | `1B97015BD8D54FA5B7D9FFBA7C1467543B9E83F49563F78DC8A7331ECF17AB82` | 未改 |

工作区 `D:\AID\VSCode\Codex\Han`，分支 `main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；`git status --porcelain` 293 行。

## 3. R06 逐条判定

| R06 编号 | r06 级别 | r07 判定 | 依据 |
|---|---|---|---|
| `B13J-R06-F001` | LOW | **CLOSED** | `tasks.md:3174` 的 `gofmt -l` 现含 7 个 `.go` 路径，与写集 `:3163–:3169` **逐项相同**：无遗漏、无多余（我以集合比对实测，差异集为空）。 |
| `B13J-R06-F002` | MEDIUM | **CLOSED** | `:3182` 增加 `go env GOOS` 主机前置；`:3183`/`:3184` 要求 `-json` 含该测试的 `"Action":"pass"` 实际执行事件；`:3189` (5) 要求记录 `GOOS`、两侧各在对应主机执行、仅有 exit 0 而无执行事件视为未验证、不得关闭；`:3193` 停止条件禁止以空跑冒充通过。`"12 条"→"13 条"` 与实际 13 条命令行一致。 |

### 3.1 F001 细化核验

- 写集 7 个 `.go` 文件：`recovery_exclusive.go`、`recovery_runtime.go`、`main.go`、`internal/recovery/filelock_flock.go`、`internal/recovery/filelock_unsupported.go`、`recovery_exclusive_posix_integration_test.go`、`recovery_exclusive_unsupported_integration_test.go`。
- `:3174` 解析出的路径集合与上述 7 个完全相等（`Compare-Object` 双向差异均为空），且顺序与写集一致；证据报告 `:3170` 仍按「非代码产物、不占 7 文件写集」单列，未被误计入。

### 3.2 F002 细化核验（假通过路径是否消除）

- 原假通过路径：POSIX 主机上该测试文件被 `!(POSIX)` 标签排除，`go test -run '^Test…$'` 无测试仍 exit 0。
- 现有三重约束闭合该路径：① `:3182` 主机门禁（GOOS 必须为 windows 或其他非 POSIX，并明写 POSIX 上的 exit 0 不算通过）；② `:3183`/`:3184` 以 `-json` 的 `Action:pass` 事件为通过条件，无执行即失败；③ `:3189` (5) 把「记录 GOOS + 两侧分别执行 + 无执行事件视为未验证」写入关闭条件，`:3193` 再以停止条件兜底。三者叠加后，空跑无法满足任一层。
- 可失败性：断言链仍是「退出码 2 + `errors.Is(err, ErrUnsupportedPlatform)`」（`:3188`），不是只断错误非 nil；测试被实际执行时，若实现缺退出码映射或分类错误，测试会失败。
- 条数：`:3172` 标 13 条；`:3174–:3186` 恰 13 行；13 条命令去重后仍 13 条（无重复）。

### 3.3 改动范围（越界核验）

- `tasks.md` 1–3154 行：842938 B / `31B0BB3EDFD584FECD683B84758748D0BF6588CB2BFF728353FF0E8580C05152`，与冻结值相同 → 该卡之前零改动。
- S4.24b 卡：改后 3155–3193 行 8007 B / `1F44129480141226B2AEB2325715404D894305995B6E3CC335D4852743917FBB`。
- S4.22 卡：改后 3194–3226 行 6224 B / `A2F5EB15BFF6D2C19A29410DDCDF49D5AB9D680D187A4EB7CA26F3408E4EB351`，与改前 `3193–3225` 指纹相同 → 内容零改动，仅整体下移 1 行。
- 增量：+698 B / +1 行（对应新增 1 条 `go env GOOS` 与 5 处改写），与作者 `r03_microfix` 记录一致。
- 逐行比对 r06 时打印的卡体（3156–3192）：差异仅出现在 `:3172`、`:3174`、`:3182`（新增行）、`:3183`、`:3184`、`:3189`、`:3193`，其余各行（目标、写集 3163–3170、requires、`:3178`–`:3181`、负向 oracle 段、证据落点、责任人、不宣称边界）逐字未变；未发现越出两条 finding 授权范围的改写。

## 4. Observations（不转 finding）

- `O01` `:3182` 的 `go env GOOS` 被计入「期望每条 exit 0」的 13 条之内，但它的通过判据是**取值**而非退出码（任何主机都 exit 0）。该行内联注释与 `:3189` (5) 已给出真正判据，故不构成假通过，仅属口径不够严格；如后续再动此卡，可把它从「验收命令」清单移为独立「执行环境前置」。
- `O02` `:3184` 的 `-race` 在 `windows/amd64` 需要 CGO 与 C 工具链；若实施主机缺工具链，该条无法通过。建议实施时在证据中记录工具链可用性，或明确 race 在非 POSIX 侧的可选性（不得因此放宽 (5)）。
- `O03` 作者 `author-index-r02.json` 顶层 `changed_files`/`changes[]` 仍是 r02 终态（`after=856471/3225`、"12 commands"），r03 状态放在 `r03_microfix`；`author-report-r02.md` §7 已显式声明「§1 数值已被本节取代」。属**已披露的陈旧摘要**，不影响判定，建议下次合并轮统一。
- `O04` `author-index-r02.json` 的 `boundary.changed_existing_cards:false` 与 r03 实际改写了 S4.24b 卡文本仍不一致（同 r06 O05），已在 `r03_microfix` 中逐行披露。

历史项不变：统计行 `:46` 82 / `:47` 67+2+14 / `:49` 55 的差一仍在（O，未授权修复）；TAB@12631 与 0x0C@12676/12691 未改；`R05-F005`（ADR-0032 候选措辞）仍 OPEN，随 ADR-0032 下次修订收口。

## 5. 实跑命令与结果

| # | 命令 | 结果 |
|---|---|---|
| 1 | `git rev-parse HEAD; git branch --show-current` | `6da8abb…38ddca`；`main` |
| 2 | 哈希/字节 tasks、Plan、framework、两份作者证据 | 与 §2 表一致 |
| 3 | 打印 `tasks.md` 3156–3193 | 13 条命令、gofmt 7 路径、GOOS 门禁、`(5)` 与停止条件加强 |
| 4 | 解析写集 `:3162–:3170` 与 `:3174` 路径并 `Compare-Object` | 7 vs 7，双向差异为空 |
| 5 | 统计 `:3174–:3186` 命令行 | 13 行、去重后 13 唯一；`:3172` 标 13 |
| 6 | 字节分段 SHA256（1–3154 / 3155–3193 / 3194–3226） | `31B0BB3E…C05152` / `1F441294…917FBB` / `A2F5EB15…4EB351` |
| 7 | 增量算术 | +698 B / +1 行；S4.22 卡内容指纹不变 |
| 8 | `rg -n "验收命令（NOT RUN；[0-9]+ 条\|12 条\|13 条"`（tasks+Plan） | 仅 `:3172` 命中 13 条，无残留 12 条 |
| 9 | `rg -n --pcre2 "S4\.24(?!a\|b)"` | 退出码 1，零命中 |
| 10 | `rg -n "ADR-0032 r60"` tasks | 退出码 1，零命中 |
| 11 | 打印统计行 `:46/:47/:49/:51` 与 S4.22 卡头 | 未变（差一仍在，未授权修复） |
| 12 | 复核 r06 两份证据哈希 | 与 r06 记录一致（未改） |
| 13 | `git status --porcelain` | 293 行 |

未运行：Go 编译/测试、PG、服务、任何 Git 写操作。Windows/非 POSIX 侧的可执行性为静态判断。

## 6. 未覆盖范围

- 未在 Windows 或 POSIX 主机实际执行任何命令（含 `go env GOOS` 与两条非 POSIX 测试命令），故「空跑路径已消除」是**文本约束层面**的判定，不是运行证据。
- 未复核 5 个 hunk 之外的 r02→r03 字节级 diff（无 r02 物理快照）；范围判定基于前缀指纹、S4.22 卡指纹、行数/字节增量与逐行比对 r06 卡体。
- 未重评 r01–r05 已 CLOSED 项与 ADR/设计结论；`framework.md` 未改。
- 未核 Plan 其他批次与 `article-management-S8.2`/Case18 现状卡面（本轮未触及）。

## 7. 本报告自身格式实测

- 字节：9186；LF：95；CRLF：0；BOM：false；末尾：单 LF；控制字符：0；TAB：0。

## 8. 边界复述

本确认只覆盖「R06 两条 finding 是否 CLOSED、r03 是否引入新不一致」。`PASS WITH MINOR` 不构成实施授权，不登记或推进 S4.22/S4.23/S4.24a/S4.24b，不关闭 `pending-B13J-D04`，不解除 `blocked-upstream`，不代表人的签收。`R05-F005` 仍需随 ADR-0032 下次修订收口；`O01`/`O02` 建议在 S4.24b 实施与取证时处理。
