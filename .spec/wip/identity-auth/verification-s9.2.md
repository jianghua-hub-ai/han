# canonical 用户删除跨模块事务：Lead 核证记录

日期：2026-10-28。原任务：`identity-auth-S9.2`。本记录由 Lead 汇总实际工作树、原始日志及 SQL 留存证据，不代替独立 Reviewer 的报告。

最终核证结论：作者与非作者 Reviewer 的真实 PostgreSQL 普通/race、专项 vet，以及 Lead 项目七项检查均通过。Lead 核对原四项前置、两文件成果、真实执行与独立评审后，判定原卡关闭条件满足，于 2026-10-28 决定关闭且仅关闭 `identity-auth-S9.2`；任务状态以本轮同步后的原 tasks SSOT 为准。

## 固定范围与产出归属

- 工作区固定为 `D:/AID/VSCode/Codex/Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。仅新增原任务的两个 integration 测试文件；原有 181 项源码输入及生产实现保持基线字节。
- TestEngineer Zeno `01a18020-4347-7b72-9f08-5661c319d33a` 产出只读测试准备；Backend-Dev Rawls `01a18028-4625-7bf3-9824-9409764e641b` 实施并自测；非作者 Reviewer Lovelace `01a18046-3223-7e42-a7d3-d55a47063bde` 独立评审。Task-Planner Ptolemy `01a17ca6-756c-7de1-b524-0665af10739f` 负责原卡元数据，不作关闭决定。
- `canonical_user_delete_test.go`：1259 行，SHA256 `2C88891DCD6F2E7F2B1B256492679FC6F4341F2F0816451D38E5841E3FE9FA20`；`canonical_delete_faults_test.go`：456 行，SHA256 `AADA00B5314FD6CE312B84547E9DF20FEDE09CA97432A8822A0EA68264F8BF99`。
- r02 完整输入清单共 183 项，SHA256 `03594FB126EB3E2493A14876E33AE5C1475E2F66C908BDB3D484B6E61CCBB73E`；其中 Go overlay 是 161 项，不代表全部输入。集成执行 cwd 为完整冻结快照 `C:/Users/J00010385/AppData/Local/Temp/han-s92-lead-guoi9beq/fixed183-r02`，迁移与数据也绑定该快照。
- r01→r02 在 PG 执行前完成两项 Lead 接缝修正：每 fixture 固定 lifecycle/verifier，保存逐阶段 11 表脱敏行像。候选版本、差异、静态日志均保留；编译转录错误不计产品 RED，未修改生产文件。

## 实际执行

| 执行者 | 检查 | 结果 |
|---|---|---|
| 作者 | PostgreSQL smoke `Daily/Zero` | 1 leaf / 3 节点通过 |
| 作者 | 原命令筛选 `^TestCanonicalUserDelete`，`-count=1 -timeout=15m -json` | 64 leaf / 76 节点通过 |
| 作者 | 同一筛选加 `-race`，`-count=1 -timeout=30m -json` | 64 leaf / 76 节点通过 |
| 作者 | `go vet -tags=integration ./test/integration/crossmodule` | 退出 0，无 DSN |
| Lead | gofmt、module verify、全仓 vet/build/contract/race/unit，固定顺序 | 七项退出均 0，gofmt 输出空 |
| 独立 Reviewer | 普通/race/专项 vet | 普通与 race 各 64 leaf / 76 节点通过，0 fail/skip/extra；vet 退出 0，三 child 实际 wait |

作者四个 child 均实际 wait 结束，每次前后核对 live/snapshot 全部 183 路径及哈希。Go 子进程关闭下载；User-scope DSN 仅注入集成 child，未进入 argv、证据文件或输出。每 fixture 在 CREATE 前核实际 literal loopback:5433、PostgreSQL 180006、用户/数据库匹配；CREATE 单独提交，九项迁移另事务提交，保留约束。

## Lead 现场复核

Lead 独立解析原始 Go JSON，核预登记的全部 leaf/父节点 run/pass 恰好各一次，无 fail、skip、遗漏、额外节点。smoke/普通/race 共保留 129 个随机 schema、1091 个 case 文件；每 case 有资源关闭记录，普通/race 共四个 barrier owner join。未删除 schema 或证据。

逐个核验文件 SHA256、leaf/schema 绑定、11 表完整性和敏感列哈希替代；962 份阶段像共 10582 个脱敏 rows 摘要重算一致。关键拒绝/故障回滚及按实际阶段顺序对应的提交前 observer、caller 与提交后像，共 242 对 11 表 raw/redacted 摘要及行像一致。不能将多次出现的同名 stage 直接覆盖后跨事务比较。Lead 核证脚本的格式/阶段匹配修正仅涉及 Temp 工具，未改变测试、原始执行或生产实现。

另现场读取全部 13 个成功 leaf 的最终脱敏像：目标已删除，email/phone 清空，目标会话为零；当前临时凭据失效，历史 consumed/superseded 保留；用户名/电话 reserved；零文章或最后 eligible orphan 完成后邮箱释放一次，仍有 deletion_pending 则保留。恢复 review 为 kept_deleted；真实转移生成两个 pending recipient。源测试还执行内容/版本/无关用户行不变、真实 run 锁、native completion proof 同一 caller transaction、非法输入无写入及合法重试断言。

raw_image_sha256 是 PostgreSQL 原始全像摘要；脱敏文件不能还原凭据原值。原始秘密字段的相等依赖执行时内存严格比较，Lead 可重算的是脱敏 rows 和文件摘要，两者不混称。

独立 Reviewer 的三个 child 与作者执行分开，128 个新 schema 无复用，1082 个 case 文件全部保留。Lead 另解析其普通/race 原始 JSON，核各 64 leaf/76 节点唯一 run/pass、实际 wait、全 183 输入前后匹配、预检/九迁移及资源收束；逐 954 份阶段像、10494 个脱敏表摘要重算通过，并另核 240 对拒绝/回滚及提交镜像。Reviewer 自身按更完整的序列核对共 360 对镜像，属 Reviewer 来源，不冒称为 Lead 的 240 对。两者的最终判断一致。

## 证据索引

- [准备与采用范围](evidence-s9.2/prep-r01/index.json)，SHA256 `652DA80EA6CCD440F10556DDCC4F415519684FCB2038F43399E2DFBE41968014`。
- [准备认领的两阶段 CAS 与 Lead 范围核验](evidence-s9.2/start-metadata-r01/index.json)，SHA256 `AE7D3DF943B6031E58318B5C26455231E511343F1DFEED2168922BD740D933F4`。
- [r01/r02 代码与 183 输入字节绑定](evidence-s9.2/code-r02/index.json)，SHA256 `5D2A42BC083922EC22A1F05E5D010B12A7FAFCA83EAE2E5B85D5300574CE6589`。
- [作者真实 PG 原始日志、保留 SQL 像与 Lead 原始核验](evidence-s9.2/author-exec-r01/index.json)，SHA256 `DBCD60049474EF8134B050FAE90DDC38475B0EE5ACF03013C3877FB0E820C829`，1133 个 payload，未复制编译/模块缓存。
- [Lead 七项检查](evidence-s9.2/lead-manifest-r02/index.json)，SHA256 `962AA0A430BF0E00C471903D22629FD677E806938D926115CA418C7B6A550D8E`。
- [独立评审](review-code-s9.2-r01.md)：Code Review Gate PASS，0 finding；报告 SHA256 `4AD92EE874C7FC2A688A5AABDDBBD6581D8873C650CD29C8BEB0126D320BB051`。
- [独立运行与读取记录的完整归档](evidence-s9.2/reviewer-exec-r01/index.json)，SHA256 `219BCCD421D29EC1D5388AD8E03AA9FFFD20C35C33FA8EC953D4EF7F60E94EDB`，1150 个 payload，含最终报告副本及 Lead 核证；原 Reviewer 索引 SHA256 `DCCE5581EE0559C7A939F75CFBB0493991D52B53957E18F5605ED4E412BD6685`。
- [最终台账 CAS、前后像及 Lead 范围核验](evidence-s9.2/close-metadata-r01/index.json)，SHA256 `9CAAA696D734B3199BAB688E8CA3043F1CEA205CEA2E0AD24F4D81E7F440D484`。Lead 现场回读确认：原 S9.2 已 `[x]`，总数 59 / 45 completed / 5 active / 9 todo；其他 58 卡、其他 34 张 identity 卡及全部 article 卡逐字节保持，Plan 仅原 B9A 本卡子句变化；四 requires、两文件、原验收命令、完整关闭条件和后续关联不变。Tasks BOM+LF、Plan BOM+CRLF 保持；183 输入、其他七份受保护文档、main/HEAD 与空暂存区核验一致。两文件 whitespace 检查按现有 CRLF 处理，未改行尾或 Git 配置。所有 Agent 已结束当前任务并释放。

## 本任务的证明边界

64 个 leaf 分为 Daily、Recovery、Retention、Rejection、Rollback、Concurrency 六组。真实 Audit INSERT 主键碰撞产生 23505；其他故障是在真实 SQL 成功后返回注入错误，证明 caller 回滚责任，不冒称数据库引擎或磁盘故障。

controlled maintenance/readiness 是明示组件 seam，不是生产 Gate；RecoveryRunStore 仍为真实锁读权威；native proof 由真实 Article completion 产生，fixture 固定 provider 仅映射并在原 tx 核验。phone 沿用 r45 已接受的路径 B 行存在/reserved/未释放限制，不新增 retained_phone。ProtectedInitial 场景被合法的管理员权限规则先拒绝，不能宣称单独命中后续 initial-target 分支。标识占用只证明 SQL UNIQUE，不证明完整注册 API。

本结果不覆盖真实 Journal/旧备份/global recovery/MarkConsumed、HTTP/Cookie/CSRF、完整注册、SMTP 投递、浏览器、部署、容量或生产就绪。原 S9.1 的作者自审来源保持不变。本卡按自身真实 PG 与独立 Review 条件判断，不倒挂未完成的 article S4.4；其真实 Unix/旧备份恢复边界仍单独保留。
