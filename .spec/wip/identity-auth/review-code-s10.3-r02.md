# identity-auth-S10.3 代码定点复审 R02

- Reviewer：Gibbs（原非作者 Reviewer）
- 日期：2026-10-24
- objectType：code
- 工作区：`D:\AID\VSCode\Codex\Han`
- 分支 / HEAD：`main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`
- 原报告：`.spec/wip/identity-auth/review-code-s10.3-r01.md`
- 原报告 SHA256：`CE775BEF877E6078399857219C417024FE5E1329232306E1B8FCA2B87D8FCC86`
- 结论：**Code Review Gate: PASS**

## 1. 状态与权限边界

本轮只复审 `IDENTITY-S10.3-REV-R01-F001`、`IDENTITY-S10.3-REV-R01-F002` 及修订回归，不修改 13 个代码对象、Tasks/Plan、design/test-plan、既有报告或二进制。

PASS 仅表示本固定修订通过本次独立代码复审，不自动关闭 S10.3，不启动 S10.4/S11.1/B6D，不改变 S10.1 `implementation stopped` 或 `IDENTITY-S10.1-REV-R01-F002` OPEN，也不表示部署、E2E 或 production readiness。

`han-server.exe` 未获清理授权；开始核得 SHA256 `A92D062BC163A48277107F02D812F66AE77A213BBE707CB2CBFEB2F63EAF4145`、大小 18,137,600 bytes，本轮未修改或删除。

## 2. 固定对象与开始哈希

开始时报告路径不存在，分支、HEAD、原报告及下列 13 个对象均与固定输入匹配：

| 文件 | SHA256 |
|---|---|
| `internal/identity/password_reset_store.go` | `6C0136408EF62A7B5FC80C4C557AEBCB557457B772EB96E8DA90024C42D3315F` |
| `internal/identity/password_reset_store_test.go` | `62EAE06BD41E5F3CFA7AAB03B3721B8D3A964834A0E7C98B194181A2791827E6` |
| `internal/identity/login.go` | `FDB1ED13FB499A1219AE24884F053A3626567BB81CAAB56EC02B1FC197B6864C` |
| `internal/identity/login_test.go` | `FB5483FB93649F9B01B060D54F35B5CC7B7A3305704D19DFCA445A2C7400414F` |
| `internal/identity/login_integration_test.go` | `15C28D7BA7EE39499D970E8C7AE53F7CE28A1A450D2530FD7697BF8BE4D407BA` |
| `internal/identity/service.go` | `4371ED7348A7035BDB6CAA57CE55F8621FF41CDE5D8D7E5615D54789F88A4F40` |
| `internal/identity/bootstrap.go` | `AEF53946AB8C55A3CC03DAE03DC1CD0E42BE12553D9619D45C7F491F154CAAA5` |
| `internal/identity/bootstrap_test.go` | `C33C31A22C32DA0FFE57099C37C3E6EE4826F4F082359F2B7A7E2B1CB43E2B45` |
| `internal/identity/administrator_create_test.go` | `AC9FE098FA3F1F09FA4944BB5BD7653BF375B23C08ED1E63F13DCEE45683EEAE` |
| `internal/session/store.go` | `4B12462E29D46B730F0DD58E6A0BAA57EC1D91B99D9F43FC91179319FF88E679` |
| `internal/session/store_test.go` | `3D32E65D407F24211355B642DAB3933F1AB36E10B60CAFC7F039B2F9B8201E09` |
| `cmd/han-server/main.go` | `61ED0992F65CBD45CDEB524DE71C4D580A92A7AC1EB70BAF1F8A073DFC7D45F2` |
| `cmd/han-server/main_test.go` | `9E6B1C6E2E3BE747E3055CDBE275815BC7976D7E5D2FCEF848718C864776BFEB` |

13 个对象仍为未跟踪文件，因此本轮没有用普通 `git diff` 代替固定全文、函数体与真实调用面检查。

## 3. 两轮实际检查

### Round 1：原 finding 与修订对象直接检查

1. 复核 `internal/session/store.go:149-171`、`internal/identity/service.go:303-350` 的 constructor 与 typed-nil 判定：
   - 先处理 untyped nil；
   - 仅对可 nil 的反射 Kind 调用 `IsNil`，没有非法 Kind panic；
   - helper 未导出，未改变 constructor 签名、公共 REST/错误码或包依赖方向。
2. 直接检查并运行 typed-nil 行为测试：
   - `internal/session/store_test.go:36-66`；
   - `internal/identity/login_test.go:57-89`；
   - `internal/identity/bootstrap_test.go:217-249`。
   测试调用真实 constructor；Login/PasswordChange 通过实际 stub 计数证明 verifier、repository、Session 写入均未发生，不是 SQL/源字符串断言。
3. 检查拆分后的安全关键流程：
   - Login：`internal/identity/login.go:1083-1431`；
   - PasswordChange：`internal/identity/service.go:352-811`；
   - production composition 与 Article 组装：`cmd/han-server/main.go:1930-2483`。
4. 运行现有 Go AST 行数测试，并用独立只读词法/括号边界分析复核函数体范围与嵌套上界。

### Round 2：真实消费者与回归深审

1. 全仓定向核对 `NewStore`、`NewLoginService`、`NewPasswordChangeService`、`PasswordResetGate`、`CheckLockedUser` 的定义与直接调用点：
   - production Gate 实现仍唯一归 `identity.PasswordResetStore`；
   - test-only fake 仍仅三处；
   - `internal/session` 没有反向导入 `internal/identity`；
   - production root 仍只创建一个 `*identity.PasswordResetStore`，同一指针进入 Session/Login/PasswordChange。
2. 深审调用顺序与事务所有权：
   - Login 保持 begin → abuse → User `FOR UPDATE` → Gate → credential → Session → audit → commit；missing User 仍走 fake credential，未调用 Gate。
   - PasswordChange 保持 caller-owned transaction → User lock → Gate → credential/lineage → Session rotation/revoke → audit → commit；失败路径仍按既有事务内审计或 rollback 后审计收敛。
   - Session 保持 token 预读 → begin → User lock → Gate → Session/credential 重读 → 必要 revoke → commit → Principal；commit error/unknown 不产生 Principal。
3. 核 production fail-fast 与 composition：
   - Gate/Store 在 handler、router、listener、Serve 前构造；
   - handler factory 失败测试证明不会创建 route/listener/server；
   - Article 仍按 finalizer → worker → coordinator → handler 构造，只有成功后才把 worker 交给 runtime，未吞并 S10.4 所有权。
4. 复核 unchanged durable Store/classifier、schema/migration、公共合同和 13 文件边界；未发现第二 authority、认证旁路或第 14 个必需文件。

## 4. 原 finding 复审

### IDENTITY-S10.3-REV-R01-F001 — CLOSED

**证据**

- `session.NewStore` 使用 `nilDependency` 同时拒绝 typed-nil transactor 与 typed-nil Gate：`internal/session/store.go:149-171`。
- `NewLoginService` 与 `NewPasswordChangeService` 使用 `nilIdentityDependency` 拒绝 typed-nil Gate：`internal/identity/service.go:303-350`。
- 三组 typed-nil 专项测试均 PASS；Login/PasswordChange 还断言零 credential probe、零 DB repository/Session 副作用。
- production construction 顺序及 `TestRun_AWIRELIFE04ConstructorFactoryFailuresHaveNoRouteOrListener` 证明 factory failure 在 listener/routes/Serve 前收敛。
- 未引入 optional/no-op/allow-all/deny-all Gate、循环依赖、公共 API 或错误语义漂移。

**结论**

原 constructor typed-nil 漏洞已关闭，mandatory non-nil/fail-fast 合同在本固定修订中成立。

### IDENTITY-S10.3-REV-R01-F002 — CLOSED

**独立测量**

- Go AST 测试实测：
  - `Login` 7 行；
  - `ChangePassword` 12 行；
  - `newProductionBusinessHandler` 30 行。
- 本轮拆分涉及的 Login helper 最大 35 行，PasswordChange helper 最大 43 行，production composition/Article helper 最大 42 行，均不超过 50 行。
- Reviewer 另以只读、保留换行且屏蔽注释/字符串的函数括号分析核定相同函数边界；其保守 lexical brace depth 上界最大为 4，低于 profile 的 5 层限制。

**语义回归**

- 拆分未改变 User lock→Gate→credential/Session 顺序、事务 begin/rollback/commit 所有权、公共错误映射或 missing User 抗枚举。
- production singleton Gate/Store 指针、启动 fail-fast、Article composition 与资源交接保持不变。
- 未复制 durable classifier，未改变 S10.4 ownership、公共合同、schema/migration 或文件边界。

**结论**

三个原超长函数及本次新增/修改的安全关键 helper 均满足已启用 go-service 强制复杂度规则，且未发现拆分造成行为漂移。

## 5. 新 findings、待决定与建议

### 新 findings

无。

### 待决定

无新的架构、公共合同或文件扩围决定。

### 非阻断建议

R01 已记录的 singleton observation seam 同源断言建议仍可后续改进；本轮静态调用面、同一变量注入和运行测试相互印证，未将其升级为 finding。

## 6. Reviewer 实际命令与结果

Reviewer 自跑：

1. 报告不存在、branch/HEAD、原报告、13 文件及 `han-server.exe` SHA256/大小核对。
2. finding 专项与 AST 行数测试：

   ```text
   go test ./internal/identity ./internal/session ./cmd/han-server \
     -run 'Test(LoginPasswordResetConstruction_(RejectsTypedNilGateBeforeCredentialProbe|SecurityCriticalFunctionsFitLineLimit)|OnboardingPasswordChangePasswordResetConstruction_RejectsTypedNilGate|PasswordResetGateConstruction_RejectsTypedNilDependencies|ProductionPasswordResetWiring_CompositionFunctionFitsLineLimit|Run_AWIRELIFE04ConstructorFactoryFailuresHaveNoRouteOrListener)$' \
     -count=1 -v
   ```

   PASS。
3. 定向普通：

   ```text
   go test ./internal/identity ./internal/session ./cmd/han-server \
     -run 'Test(LoginPasswordReset|OnboardingPasswordChangePasswordReset|SessionResolvePasswordReset|PasswordResetGate|ProductionPasswordResetWiring|Run_AWIRELIFE04ConstructorFactoryFailuresHaveNoRouteOrListener)' \
     -count=1
   ```

   PASS。
4. 同一定向 `go test -race`：PASS。
5. `go vet ./internal/identity ./internal/session ./cmd/han-server`：PASS。
6. 13 文件 scoped `gofmt -d`：无输出。
7. 全仓定向 `rg` 核 constructor/call-site、Gate 实现/fake、反向 import 与旁路关键字。
8. Reviewer 独立只读函数边界/嵌套上界分析：所有目标满足 50 行/5 层限制。

所有 Go 子进程均先移除 `HAN_TEST_POSTGRES_DSN` 环境变量；Reviewer 未读取、输出或使用其值。

Lead 提供但未冒充 Reviewer 自跑的外部证据：新 hashes、finding tests、定向普通/race、三包普通/race/vet、scoped gofmt/diff-check 与 Manifest 七项均 PASS。第一次 Manifest 包装函数名与 PowerShell alias 冲突，仅完成 gofmt 后包装脚本失败；Lead 已以 `RunCheck` 完整重跑七项并全部 PASS，该次包装失败不是项目检查失败。

## 7. 未验证风险

- 未运行真实 PostgreSQL、真实 Gate SQL fixture、多连接锁序、commit actual-state/unknown 或 migration 混跑。
- 未运行 SMTP、部署、Host、浏览器或 E2E。
- Reviewer 未独立重跑全仓 Manifest 七项或三包全量普通/race；本轮独立证据限于上述 finding 专项、定向普通/race、三包 vet 与静态检查。
- 13 个对象均为未跟踪文件，历史差异不能由普通 Git diff 提供；本结论绑定 §2 固定 SHA。
- `han-server.exe` 仍是未获清理授权的未跟踪构建制品。

## 8. Gate

**Code Review Gate: PASS**

理由：R01 的两个 MEDIUM finding 均已 CLOSED，未发现新增阻断 finding，适用定向普通/race、vet、格式与静态证据齐备。最终任务关闭、后继启动及部署仍由 Lead 按 development-delivery 现场 Gate 决定。

## 9. 结束验证

- 结束 branch/HEAD 仍为 `main` / `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。
- 13 个固定对象结束 SHA256 与 §2 逐项一致，未发生对象漂移。
- `han-server.exe` 结束 SHA256 仍为 `A92D062BC163A48277107F02D812F66AE77A213BBE707CB2CBFEB2F63EAF4145`，大小仍为 18,137,600 bytes。
- 报告 scoped `git diff --no-index --check -- /dev/null .spec/wip/identity-auth/review-code-s10.3-r02.md` 未报告 whitespace error；退出码 1 仅表示新增文件与空文件存在预期差异，行尾转换 warning 不是 diff-check 缺陷，判定 PASS。
- 本轮唯一写入归属为 `.spec/wip/identity-auth/review-code-s10.3-r02.md`。
