# identity-auth design r22 — TP-S03-R04-F001 修订复审

**日期**：2026-10-23  
**角色**：非作者独立 Reviewer  
**结论**：**PASS**  
**原 finding**：`TP-S03-R04-F001` — **CLOSED**  
**开放 findings**：HIGH 0 / MEDIUM 0 / LOW 0  

## 1. 固定对象

| 项目 | 固定值 |
|---|---|
| 对象 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| revision | `identity-auth-design-ir01-r22` |
| 要求 SHA256 | `FDF9D5A6EBD05E8CE7AAE34550145EADB6469EB6E75393BB6BFB091B4E8E0A9D` |
| 开始 SHA256 | `FDF9D5A6EBD05E8CE7AAE34550145EADB6469EB6E75393BB6BFB091B4E8E0A9D` |
| 写报告前 SHA256 | `FDF9D5A6EBD05E8CE7AAE34550145EADB6469EB6E75393BB6BFB091B4E8E0A9D` |
| 状态 | `draft`；评审待完成、批准为“未批准”（design:4、38-39） |

本复审只判断 `TP-S03-R04-F001`。原规划评审中的 F002/F003/F004、tasks/Plan 的其他问题及其修订均不在本报告结论范围。

## 2. 两轮实际检查范围

### Round 1：r22 直接修订与反向残留检查

1. 核对 revision、状态流转和整改表：
   - r22 明确把外部部署材料与真实演练限定为 production observability readiness Gate，并规定 S3.2/B4 按自身 Gate 关闭（design:75）。
   - 整改表明确三层串行且各自闭环，S3.2/B4 不等待 S3.5/S3.6（design:120）。
2. 全文反查所有 `S3.2`、`S3.5`、`S3.6`、`B4` 及“等待/不得关闭/不得宣称/生产就绪”组合，未发现要求 S3.2 或 B4 等待 S3.5/S3.6 才能关闭的相反残留。
3. 核对 §6.8、§8、整改响应和确认记录：
   - S3.2 在代码、适用测试、独立 Review 与 Lead task Gate 满足后可关闭；B4 在三卡关闭及独立批次 Review 后可关闭；依赖保持 `S3.2 → S3.5 → S3.6`（design:2497）。
   - S3.6 通过前禁止宣称 production observability readiness、真实部署/采集/告警演练通过或需求整体完成；该限制不反向阻塞前三层各自关闭（design:2499）。
   - 当前 F005、S3.2、B4 仍未关闭；这是当前事实，不是等待 S3.5/S3.6 的前置条件（design:2530）。
   - r22 仍为 draft，历史 r21 checkpoint 不自动批准 r22，且没有伪造新 checkpoint/decisionId（design:2650、2702、2748-2756）。
4. 未发现代码、runbook、Dashboard、alert rules、receiver、部署、采集或演练已完成的错误声明；正文明确否定这些完成事实（design:2526、2530）。

### Round 2：上游合同与真实消费者反向核对

1. **architecture r23**：
   - 应用 composition、0.0.4 conformance 与真实 scrape 部署 Gate 分离（architecture:560-564）。
   - Dashboard、告警、receiver、runbook、采集与 30 天留存须实际部署并演练后才能宣称生产就绪（architecture:566）。
   - 可观测性部署材料保持 provider-neutral，绑定实际 backend/receiver 并演练前生产就绪 Gate 关闭（architecture:883-892）。
2. **ADR-0022**：
   - 采集与 30 天查询属于生产就绪 Gate（framework:618-622）。
   - 真实 scrape 是部署 Gate，不能替代应用协议 conformance（framework:626-634）。
   - 实际 receiver/排班/runbook 未配置和演练前生产就绪 Gate 关闭（framework:661-667）。
   - 外部 collector/scraper/Dashboard/Alertmanager/receiver 由部署验收 Gate 证明，未装配不得宣称生产就绪（framework:669-681）。
3. **test-plan r06**：
   - §4.2.2 将 I-OBS-DEP-01～06 与应用级 I-F005-13～22 明确分离，且全部真实部署场景当前为“计划/未执行”（test-plan:175-186）。
   - §9.1 分别定义 S3.2 application composition、部署材料和真实验证，并要求代码评审明确“应用实现结论与 deployment readiness 是两个 Gate”（test-plan:369-377）。
   - 执行顺序为 application composition → 部署材料 → 真实 backend 验证（test-plan:381-393）。
   - 当前不得关闭 F005/S3.2/B4 是因为实现和验证尚未完成，不是永久要求等待 S3.5/S3.6 才允许关闭应用 task/batch（test-plan:414-421、432-436）。
4. **当前 tasks/Plan 消费关系（只读理解）**：
   - S3.2 当前关闭条件仍要求完成 I-F005-13～22、适用 Gate、独立代码 Review、关闭 F005，并经 Lead 现场核验；S3.5/S3.6 登记或 mock 均不能替代（tasks:350）。
   - S3.5 依赖 S3.2，S3.6 依赖 S3.2 与 S3.5；现有依赖仍为 `S3.2 → S3.5 → S3.6`（tasks:365-368、405-446）。
   - 因 r22 已允许 S3.2 先按自身 Gate 关闭，原 finding 所述“保持 S3.2 未关闭则 S3.5 无法启动”的合同循环已消除。

## 3. Finding 复审

### `TP-S03-R04-F001` — CLOSED

**原触发**：application composition 已完成而 S3.5/S3.6 未完成时，design r21 禁止关闭 S3.2/B4，但 tasks/Plan 又要求 S3.2 关闭后才启动 S3.5，形成不可满足循环。

**修订证据**：

- design:2497 明确 S3.2 与 B4 可分别按自身 task/batch Gate 关闭，不等待 S3.5/S3.6。
- design:2499 将 S3.6 前的禁止声明精确限定为 production observability readiness、真实部署/采集/告警演练通过及需求整体完成。
- design:2530 保持 F005/S3.2/B4 当前未关闭，同时明确该事实不是等待 S3.5/S3.6 的关闭前置。
- design:2697 在整改响应中再次固定三层职责和 `S3.2 → S3.5 → S3.6`。

**判断**：修订采用了用户已决定的推荐方向，消除了循环，没有削弱 architecture r23、ADR-0022 或 test-plan r06 的 production observability readiness Gate。

## 4. Gate 结论

### 设计修订 Gate：PASS

- `TP-S03-R04-F001` 已正确修复。
- 未发现 HIGH、MEDIUM 或 LOW 新 finding。
- 本 PASS 只证明 r22 对 F001 的设计合同一致；不表示 F005、S3.2、B4、S3.5、S3.6、部署或 E2E 已完成。
- S3.6 真实部署、采集、告警与演练通过前，仍不得宣称 production observability readiness 或需求整体完成。

## 5. Task-Planner 接力条件

**可交 Task-Planner，但须先完成 r22 精确对象软确认。**

接力时必须：

1. 以 revision `identity-auth-design-ir01-r22` 和本报告固定 SHA256 为新设计来源，不借用 r21 checkpoint 批准 r22。
2. 保持 `S3.2 → S3.5 → S3.6`，并使 S3.2/B4 关闭条件与 design:2497-2499 一致。
3. 不因本复审关闭 F005、S3.2、B4，亦不据此授权编码、测试执行、平台/provider 选择、部署、E2E、Git 或发布。
4. 原规划评审 F002/F003/F004 仍须各自修订和复审；本报告不对其作通过判断。

## 6. 限制与剩余风险

- 按授权未运行 Go、部署、E2E 或任何平台校验。
- 未评审 r22 之外的设计内容正确性，也未复审 tasks/Plan 的 F002/F003/F004 修订。
- 未验证真实日志留存、Prometheus scrape、告警时序、Dashboard、receiver、升级链或 runbook 演练；这些仍由 S3.5/S3.6 Gate 承担。
- r22 精确对象尚未获得新的软确认；历史 r21 accepted checkpoint 仅为历史证据。

## 7. 复审条件

若 design.md 哈希、revision 或 §6.8/§8/§9/确认记录发生变化，或 Task-Planner 将 S3.2/B4 再次绑定到 S3.5/S3.6 完成后才能关闭，须重新复审 `TP-S03-R04-F001`。
