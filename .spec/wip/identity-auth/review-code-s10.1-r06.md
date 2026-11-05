# identity-auth S10.1 Code Review R06

- 日期：2026-10-25
- objectType：code
- Reviewer：identity-auth-S10.1 原非作者独立 Reviewer
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：
  - R04 SHA256：`CA0AE21ED8A4B2AC61144CC165597BF3ED6DB3DFDD4D1054E7AF791EF0D8D3DE`
  - R05 SHA256：`B313C68492B2C0CFC8459372363FCC2DFEBB5B9295179DA8DBAA1E6FB71EB299`
- 定点结论：`IDENTITY-S10.1-REV-R04-F002` **CLOSED**
- 新 findings：**无**
- **Code Review Gate：PASS**

## 1. 固定对象与防漂移

五文件均为共享工作树中的 untracked 对象；本评审没有使用空 `git diff` 代替全文审查、固定哈希和实际测试。

| 文件 | 固定 SHA256 |
|---|---|
| `internal/identity/handler.go` | `FCCF0290BEE93E7730AD8EE8B65A07B8BFC475D0BEA14C5A3AB2372D2C49C36C` |
| `internal/identity/password_reset.go` | `926E4507580F608AB37D13ACA88EDF8295DCC50BAE541168655326B8D33724FA` |
| `internal/identity/password_reset_test.go` | `046BF47AD5804AA3E1B5308C0AB3A8458F9AD245C8017F094C563BAF874C3040` |
| `cmd/han-server/main.go` | `38419E8708CE1FACAFC164D344D3D9A9F5E673D873529CD61810C2A6E63C6166` |
| `cmd/han-server/main_test.go` | `77062DAE8D69251EF731AF548D0A63294C4D5648E61BE8EF3F9958EC9D26085C` |

开始审查及落报告前均核对：

- 分支、HEAD匹配；
- 五文件、R04与R05报告哈希匹配；
- R06路径此前不存在；
- 评审期间五文件未漂移。

## 2. 两轮实际检查范围

### Round 1：五文件直接审查

- 全文检查 production builder新增 coordinator/Handler构造与依赖注入；
- 检查两条 exact POST route及404/405/Allow；
- 检查 Origin、Session、CSRF、Access、Handler与coordinator授权边界；
- 检查 Idempotency-Key、稳定成功/error envelope与secret safety；
- 检查缺 durable/security依赖的fail-closed路径；
- 检查新增测试是否真实调用 `newProductionBusinessHandler` 和 production router；
- 回归R03三项、R04-F001及全部 `TestPasswordReset`。

### Round 2：真实消费者与唯一 authority

有界追踪：

- S10.3建立的唯一 `PasswordResetStore`/Gate及Login、PasswordChange、Session消费者；
- S10.4既有permit/operator/worker/lifecycle；
- production prepared Mailer、Audit、IDs、Clock、Hasher、Idempotency keyring/lock/TTL及SMTP budget；
- production startup在listener前的worker construction检查；
- HTTP Authorizer、OriginGuard和router短路顺序；
- tasks.md 874-908、Plan R对账、design r44、stories r16、test-plan r20；
- test seam全部赋值点，确认仅存在于 `main_test.go`。

没有扩展为无关全仓评审。

## 3. `IDENTITY-S10.1-REV-R04-F002` — CLOSED

原问题：三文件能力只在单元测试中可达，唯一 production composition没有构造/reset Handler或注册两条公共REST route。

### 3.1 唯一 production builder 已构造真实 coordinator/Handler

`cmd/han-server/main.go:2014-2257`：

- `newProductionBusinessBuilder`只构造一个 `*identity.PasswordResetStore`；
- 同一Store注入Session，并继续被Login、PasswordChange和password-reset coordinator消费；
- `buildPasswordResetRecovery`基于同一Store构造唯一permit、operator和worker；
- `buildIdentityHandlers`调用 `newPasswordResetHandler`；
- `newPasswordResetHandler`先要求Store/permit/operator/worker全部非nil，再构造真实：
  - `PasswordResetCoordinator`
  - `PasswordResetHandler`
- coordinator复用现有production依赖：
  - pool/transactions；
  - `PostgresPasswordResetRepository`；
  - 同一 `PasswordResetStore`；
  - `security.PasswordPolicy`；
  - prepared password hasher；
  - crypto temporary-password generator；
  - prepared Mailer；
  - prepared idempotency keyring、lock与TTL；
  - composition SMTP budget；
  - 同一Audit Store、IDs与Clock。

没有第二Store/Gate/operator/worker、第二composition root、包级global或service locator。

### 3.2 两条公共REST route已通过唯一router可达

`cmd/han-server/main.go:2057-2073`、`:2671-2806`：

- reset Handler进入production router的固定第八个handler槽位；
- 两条route均只接受exact POST：
  - `/api/v1/admin/users/{id}/password-reset`
  - `/api/v1/admin/administrators/{id}/password-reset`
- 缺ID、额外path segment及其他未知path返回稳定JSON 404；
- 已知path非POST返回稳定JSON 405及 `Allow: POST`；
- router拒绝发生在Handler前，拒绝测试中reset业务调用为零。

`internal/identity/handler.go:890-969`保持：

- 从受保护context读取principal与trace；
- 要求精确一个 `Idempotency-Key`；
- 解析member/admin目标role及合法UUID；
- 成功响应只输出：
  - `user_id`
  - `temporary_credential_expires_at`
  - `delivery_status`
- 不输出temporary password。

### 3.3 安全middleware与授权合同正确

`cmd/han-server/main.go:2457-2487`：

- reset route固定为 `OriginGuard → HTTPAuthorizer(AccessGeneral) → Handler → coordinator`；
- HTTPAuthorizer继续承担Cookie/Session/CSRF及onboarding访问状态；
- Handler把解析出的administrator principal传给coordinator；
- coordinator既有管理员授权、自重置禁止、目标role/状态/initial-admin保护没有修改。

production composition测试覆盖：

- Origin拒绝；
- Session缺失；
- CSRF拒绝；
-受限/onboarding principal拒绝；
- 缺失Idempotency-Key；
- 非法目标输入；
- 全部断言reset service零调用及稳定error envelope/trace。

固定identity专项继续覆盖coordinator级管理员授权和目标资格，因此middleware拒绝与业务授权两层均有实际回归证据。

### 3.4 缺durable/security依赖时fail closed

`newPasswordResetHandler`在构造Handler前要求：

- Store/Gate；
- permit；
- operator；
- worker。

`NewPasswordResetService`继续验证Mailer、Hasher、Policy、IDs、Clock、Audit、Idempotency keyring/lock/TTL、SMTP budget等依赖。

新增失败矩阵逐项移除Store、permit、operator、worker、Mailer、Hasher、IDs、Clock和keyring，均不能获得Handler。production `runComposed`还在listener前检查runtime worker实际构造结果；因此不存在“缺worker但route/listener已开放”的路径。

### 3.5 新测试确实经过production composition/router

成功与安全测试通过 `newArticleProductionHandlerWithTestSeam` 调用：

1. production配置加载与prepared dependency构造；
2. `newProductionBusinessHandler`；
3. `newProductionBusinessBuilder`；
4. real Store/permit/operator/worker/coordinator/Handler构造；
5. production OriginGuard、Authorizer和router；
6. 最终HTTP请求。

测试service seam只在真实coordinator成功构造之后替换其调用端，用于避免真实数据库/SMTP副作用并观察Handler输入输出；它没有绕过production builder、Handler、middleware或router。

`articleProductionTestSeam`及其 `passwordReset` 赋值点只存在于 `main_test.go`。真实 `runComposed` 不设置该unexported seam，环境变量或公共输入也不能启用它，因此测试替身不会进入production authority。

### 3.6 关闭链未回退

- `handler.go`哈希与R03/R04/R05一致；
- `password_reset.go`、`password_reset_test.go`哈希与R05一致；
- 全部 `TestPasswordReset` 普通/race通过；
- same-key预分类、accepted/unknown settlement、parent-cancellation evidence context、secret清空及三消费者Gate回归均通过。

综上，R04-F002的“production不可达”触发条件已经消失，且没有用第二authority/root或测试fake进入production来规避，结论 **CLOSED**。

## 4. 新 Findings

**无。**

未发现五文件production wiring引入新的正确性、安全、权限、幂等、secret泄漏、durable状态机、composition ownership或测试真实性缺陷。

## 5. 实际命令与结果

以下均在仓库根目录实际执行：

| 命令 | 结果 |
|---|---|
| `go test ./cmd/han-server -run 'Test.*PasswordReset.*(Route\|Composition\|Wiring\|Origin\|Authorization\|Method\|NotFound)' -count=1` | PASS |
| 上述route专项加 `-race` | PASS |
| 七个production wiring/security/fail-closed测试的精确正则普通测试 | PASS |
| 上述精确集合加 `-race` | PASS |
| `go test ./internal/identity -run '^TestPasswordReset' -count=1` | PASS |
| `go test -race ./internal/identity -run '^TestPasswordReset' -count=1` | PASS |
| tasks 896所列三包消费者精确正则普通测试 | PASS |
| 同一三包消费者集合加 `-race` | PASS |
| `go test ./internal/identity ./internal/session ./cmd/han-server -run 'PasswordReset' -count=1` | PASS |
| 同一三包集合加 `-race` | PASS |
| `go vet ./internal/identity ./internal/session ./internal/mailer ./cmd/han-server` | PASS |
| `go build ./cmd/han-server` | PASS |
| 五文件 `git diff --check -- ...` | PASS；五文件untracked，故只作补充检查 |
| 五文件 `gofmt -l -- ...` | PASS；无文件输出 |
| `rg`检查coordinator/Handler/routes及test seam赋值点 | production构造/route存在；seam赋值仅在测试 |
| `Get-FileHash -Algorithm SHA256`、分支及HEAD核对 | 全部匹配 |

Lead提供的Manifest全量PASS作为交付背景保留；本表仅登记Reviewer本轮独立执行的命令。

### 执行副作用说明

Reviewer执行任务验收命令 `go build ./cmd/han-server` 后，Go在仓库根目录生成未跟踪的 `han-server.exe`。因本轮明确禁止删除/清理，Reviewer未擅自移除。该文件不属于固定五文件revision，五文件哈希未变化；后续是否清理由Lead按授权处理。

## 6. 未验证风险与限制

- 未读取、输出或使用 `HAN_TEST_POSTGRES_DSN`。
- 未连接真实PostgreSQL或SMTP，未部署、未执行E2E。
- 未验证真实PG事务锁、commit acknowledgement loss、多连接竞态、migration混跑、真实provider受理、worker crash/restart、容量或production readiness。
- HTTP测试使用production composition/router与安全middleware，但以test seam替代最终数据库/SMTP coordinator调用；真实coordinator逻辑由identity专项普通/race独立证明，二者的真实外部系统联通留给后续集成/E2E。
- 共享工作树其他大量修改/未跟踪文件不在本结论范围，未被回滚、覆盖、格式化、stage、commit或push。

## 7. Gate

| 项目 | 结论 |
|---|---|
| R03 F001/F002/F003 | **CLOSED，未回退** |
| `IDENTITY-S10.1-REV-R04-F001` | **CLOSED，未回退** |
| `IDENTITY-S10.1-REV-R04-F002` | **CLOSED** |
| 新 findings | **无** |

**Code Review Gate：PASS**

本结论仅覆盖固定五文件task-level revision及允许的真实消费者范围，不替代后续B6C-3 integrated Review、真实PostgreSQL/SMTP、部署、E2E或production readiness Gate。
