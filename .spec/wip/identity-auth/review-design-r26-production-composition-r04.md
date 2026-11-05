# identity-auth design r26 production composition 独立复审

- **评审日期**：2026-10-23
- **Reviewer**：非作者独立 Reviewer（延续 r23→r24→r25→r26 finding 链）
- **对象类型**：design
- **结论**：**PASS**
- **Finding 统计**：CRITICAL 0 / HIGH 0 / MEDIUM 0 / LOW 0

## 1. 固定对象与边界

| 项目 | 固定值 / 结果 |
|---|---|
| 被审对象 | `.add/project/knowledge/design-workspace/identity-auth/design.md` |
| revision | `identity-auth-design-ir01-r26` |
| 要求 SHA256 | `D2D17BACE467E4BB71199EF2A344DD9EC2AB74E821EF5CF72D277AEE21D0DF02` |
| 开始 SHA256 | `D2D17BACE467E4BB71199EF2A344DD9EC2AB74E821EF5CF72D277AEE21D0DF02` |
| 原 r25 复审 | `.spec/wip/identity-auth/review-design-r25-production-composition-r03.md` |
| 原 r25 复审 SHA256 | `02F701228B8C5AC4109049EE7A4842E8E8647DC437A43C26374B8CC5EC05DF33` |
| 分支 / HEAD | `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca` |

本轮只复审 `IA-R24-PC-R02-F001` 在 r25 后剩余的 capacity 429 `Retry-After` 合同。design 与代码消费者全部只读；唯一写入本报告。共享工作树存在他人既有改动，本轮未回滚、覆盖、清理或格式化任何现状。

## 2. 结论摘要

r26 已完整关闭唯一残留：

1. `internal/security` Gate capacity typed error 是容量路径 duration 的唯一 owner；
2. duration 固定为严格正 `1s`，只表示最小客户端退避，不估算 permit 释放；
3. Service 只识别并沿既有 `loginRateLimitError` carrier 原样承载；
4. 现有 Handler 的正 duration 向上取整规则精确输出 `Retry-After: 1`；
5. abuse bucket 429 继续使用自身动态 duration，没有被全局固定为 1 秒；
6. startup fake Hash/Verify 在 listener 前 fail closed，运行期真实/fake Verify 与 temporary/permanent rehash 使用同一 capacity 策略；
7. body、日志、审计和 metric label 不披露 duration、permit、profile 或 Argon2 参数。

真实消费者能够在现有十八文件范围内承接该合同，不需要更改公共 code/body、ADR、architecture、schema、依赖或新增第十九个文件。未发现新增阻断项。

## 3. Round 1：设计对象直接检查

实际检查：

- revision/change log、评审整改台账、Abstract、公共错误表、启动顺序、过载控制、测试合同、十八文件映射、现状声明与确认记录；
- owner、固定值、carrier、Header、startup/runtime 分流、abuse 动态回归、敏感信息边界在全文中的一致性；
- fixed-fail、PG18 blocked、未实施/未部署/未 E2E、无 r26 checkpoint 和 production readiness 分层。

直接证据：

- `design.md:600,704` 固定 Gate error owner、`1s`、Service carrier、Handler Header 与 startup/runtime 分流；
- `design.md:876,885` 保持唯一 `request.rate_limited` code/message，同时区分 abuse 动态 duration 和 capacity 固定 `1s`；
- `design.md:2428,2431` 固定 non-blocking Gate、无排队/重试、`Retry-After: 1`、不写 abuse failure bucket及敏感信息限制；
- `design.md:2640` 覆盖 startup fake Hash/Verify、真实/fake Verify、temporary/permanent rehash、审计、metrics 和多动态 abuse duration 回归；
- `design.md:2761-2780` 保持十八个既有文件及各层责任，不授权第十九文件；
- `design.md:2894-2895,2954` 如实保留 fixed-fail、PG18 blocked、待独立复审和无 r26 checkpoint。

全文未发现把全部 429 固定为 1 秒、由 Handler/Service 重造 duration、把 1 秒解释为资源恢复估计，或提前声明 r26 已实现/已确认的相反残留。

## 4. Round 2：真实消费者与反向合同检查

实际反查：

- `internal/security/argon2.go` / `argon2_test.go`；
- `internal/identity/service.go`、`login.go`、`handler.go` 及 `login_test.go`；
- `internal/platform/errors.go`；
- `cmd/han-server/main.go` production fixed-fail；
- 十八文件存在性与现有包依赖方向。

反查结果：

1. **Gate error owner 可实现**  
   `internal/security` 已是 Argon2 hasher owner；在同一文件增加稳定 typed capacity error及其 `1s` duration，不需要新包、依赖或文件。`identity → security` 的依赖方向与设计一致。

2. **现有 carrier 可复用**  
   `login.go:1790-1815` 已有 `loginRateLimitError{retryAfter,cause}` 与 `LoginRateLimitRetryAfter`。credential Verify、fake Verify及两类 rehash Hash 的错误均汇入现有 credential-error helper；增加 `errors.As` 识别 capacity typed error即可原样承载 1 秒。

3. **Handler 可精确输出**  
   `handler.go:215-221` 已对正 duration执行秒级向上取整。输入恰为 `1s` 时输出精确 `Retry-After: 1`，无需改变公共 Handler API或 `platform.WriteError`。

4. **动态 abuse 429 不冲突**  
   `login.go:1769-1779` 继续从 `LoginAbuseDecision.RetryAfter` 构造同一 carrier；`login_test.go:659-687` 已证明 `1500ms → Retry-After: 2`。capacity 分支只增加另一个 carrier 来源，不会把 abuse duration 改为固定值。

5. **所有运行期调用路径可到达同一映射**  
   未知用户 fake Verify、真实/temporary Verify、temporary/permanent rehash Hash 均通过现有 credential verifier error路径返回 Service；capacity 分支可在写 abuse failure bucket前进入事务失败/审计和最终 Handler observation路径。

6. **startup 与运行期边界清楚**  
   `NewLoginService` 的 startup fake Verify本来就会在不匹配或 error 时拒绝构造；r26 只要求保留 typed cause并在 listener 前 fail closed，不生成 HTTP。production `composeBusiness` 仍固定失败，故设计未把待实施合同误写为现状。

## 5. 原 Finding disposition

| 原 Finding | 结论 | 依据 |
|---|---|---|
| `IA-R24-PC-R02-F001` | **CLOSED** | r25 已关闭 benchmark/profile、单一 Gate、资源包络与调用路径问题；r26 又固定 capacity duration owner=`internal/security` typed error、值=`1s`、Service 原样承载、Handler 输出 `Retry-After: 1`，并保留 abuse 动态 duration及 startup/runtime 全路径验证。 |

finding 链保持为：

- r24：F001 HIGH；
- r25：核心风险已修复，收窄为 capacity Header MEDIUM；
- r26：唯一残留关闭。

## 6. 新 Findings

无。

## 7. Gate 与接力结论

### 设计 Gate：PASS

- `IA-R24-PC-R02-F001`：**CLOSED**；
- 新 findings：0；
- r23→r24→r25→r26 production composition finding 链当前无开放项；
- 可进入 r26 精确对象结果确认及获授权后的 test-plan/tasks 来源同步；
- 本 PASS 不替用户确认 r26，不构成编码、测试执行、部署、E2E、checkpoint 或 Git 操作授权。

production observability readiness Gate 未被削弱：S3.2 application composition、S3.5 provider-neutral 部署材料、S3.6 真实部署/采集/告警演练继续分层；Argon2 目标环境 benchmark/profile 报告、内存/延迟预算和采用记录仍是 production readiness 前置；S3.6 前不得声明 production observability readiness 或需求整体完成。

## 8. 限制与剩余风险

- 按授权未运行 Go test/race/vet/build、真实 PostgreSQL 18、benchmark、部署、scrape、告警或 E2E；
- 本轮只证明 r26 设计及当前消费者可实施，不证明 Gate、typed error、factory、carrier 或 Header 已编码；
- Backend-Dev/Lead 的既有 PASS 仅为历史背景，未外推为 r26 实现证据；
- production `composeBusiness` 当前仍 fixed-fail，PG18 专用验证仍 `blocked-for-execution`；
- r26 尚无精确对象 checkpoint，历史 r22 accepted 不自动批准 r26。

## 9. 后续条件

1. 用户或实际责任方另行完成 r26 精确对象结果确认；
2. test-plan/tasks 若同步来源，必须使用 r26 revision/object digest并补入 capacity `Retry-After: 1` 全路径矩阵，不能只替换版本号；
3. 后续实施仍限十八文件，若需要公共 API、ADR、architecture、schema、依赖或第十九文件，必须停止并升级范围；
4. 实施完成后独立核对 typed error、carrier、Header、动态 abuse 回归、审计/metrics、全项目检查与 PG18 独立 Gate。
