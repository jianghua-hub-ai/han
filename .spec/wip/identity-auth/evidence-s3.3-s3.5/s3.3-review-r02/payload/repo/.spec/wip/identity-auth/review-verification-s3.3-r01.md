# identity-auth S3.3 核证成果独立评审 R01

- 日期：2026-10-28；活动：同一 `development-delivery` 中测试证据与直接 code seams 的非作者评审。
- 工作区：`D:/AID/VSCode/Codex/Han`，`main`，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`；依据实际未提交字节，不只依据 HEAD。
- 作者：Test-Engineer **Bacon**，Session `01a18077-e5df-7240-baac-bdee83a21685`。本 Reviewer 为 Lead 本轮受派的独立非作者，未再委派、未运行 Go、未替作者修复。
- objectType：**code**（定向测试证据及其实际生产接缝）；design/ADR 只消费本轮安全验收与豁免边界。
- 固定报告：`.spec/wip/identity-auth/verification-s3.3-r01.md`，SHA256 **`F9F8D228BC1B092CCA709E01AB51557971FA896D9C6DE58FC2A6DD4D37AB98D8`**。接收冻结信号时 native final 待回是历史状态；Lead 随后正式通知 Bacon 已 native **completed/stopped** 回交，报告 SHA 不变。作者证据 index SHA 为 `A0BE45DCF9F96BCE02784108D299E8BD2CD7AF9FCA1E946284509B102424B0C3`，本轮封存时另核字节。
- **报告与 raw 的可信性：核对通过，限本报告所列执行、断言层次和限制。未发现把 stub、SQL spy、历史 PG 或浏览器豁免虚报为本次真实验证。**
- **S3.3 覆盖充分性：NOT PASS / EVIDENCE REQUIRED；STOP closure。** 两项 MEDIUM 未豁免验证缺口仍 OPEN，分别对应作者 G01/G02。它们是验证缺口，**不是已复现的产品 Bug**；不因 Go/race 绿而批准全覆盖或关闭。
- 真实浏览器：**WAIVED / NOT RUN**，不是 PASS。S3.4/S3.6 及产品安全合同不变；S3.5 已冻结成果不改写。

## 1. 固定输入与成功判据

成功判据是：冻结作者报告与选择集/raw；独立核真实完成、精确节点与源码版本；按原验收分别判断每项证据能证明什么、不能证明什么；给未豁免缺口稳定 ID；唯一仓内写入本报告，最后 stopped 交 Lead。

原验收来自 `test-plan.md:193/195/196/199` 的 **I-AUTH-01/03/04/07**，原 S3.3 卡 `tasks.md:442`～`:458`，design 同源/会话合同及 `:3359`～`:3360`，已接受 ADR-0025。B0.1、S1.4、S3.2 在本次原读 Tasks 中均为 `[x]`，不因此推导 S3.3 已通过。

ADR-0025 仅豁免真实浏览器环境准备、连接和执行；Cookie/Origin/Fetch Metadata/CORS/CSRF、认证后轮换与旧 Cookie 失效、持久防暴力合同保留。两个原计划 browser 文件均未创建，本 Reviewer 未创建它们。G01/G02 不能由浏览器豁免覆盖，也不是把后续 E2E/部署验证倒挂为新前提。

| 关键契约文件 | 固定 SHA256 |
|---|---|
| identity design.md | `D57BB2152A93F6F651165EE7DBE1913B28E9E8719B6991B1FDD36E8B7DBAB7FE` |
| identity test-plan.md | `0052894D3C4B58F644749F8337BB264FD2768DD26A31C7EA0A361F0F61D939BD` |
| decisions/framework.md | `4FA647D562B869A4D5F09241244937F5E7066A6837965C766E39B6B7099315A2` |

行号以本轮冻结读版本为准；Tasks/Plan 的原读版本与受控 S3.5 元数据写者另见交回保护收据，不把它们永久绑定为旧整文件 SHA。

## 2. 两轮实际审查

### Round 1：合同、测试断言与执行证据

先从原验收抽取预期，再核当前测试与生产代码，未用实现反推验收：

- Cookie 固定 `__Host-han_session`、Secure/HttpOnly/Lax、Path=/、无 Domain；检查真实序列化入口、Handler success/failure 与摘要参数断言。
- Origin/Fetch Metadata 在认证前拒绝；有 Metadata 时只接受 same-origin；CORS Guard 的拒绝与允许分支均清除 ACAO/ACAC；写请求 CSRF 要求、缺失/伪造拒绝及下游零调用。
- 成功登录轮换、认证失败/审计失败保持旧会话；追到实际 SessionStore revoke/insert、caller commit/rollback 与下一次 ResolvePrincipal 的数据库谓词，检查测试是否真正串接这些行为。
- 三维桶 keys、读改写、来源/账号退避分类，区分 memory store、SQL lock spy 与真实 PostgreSQL 并发/重启证据；检查代理 Header 解析的实际 provider 与 23 个相关子例。
- 对 normal/race 原始 JSON 全流独立解析；对比预登记 top/leaf、源文件及测试体、每节点 run/pass、package 结束状态、wait/exit、原始流 SHA、执行前后 183 项 guard 和作者落盘的白名单环境记录。没有重跑 Go。

### Round 2：真实直接消费者、历史适用性及作者矩阵

- 追 `http_security.go` 的 Cookie/Origin/CORS/HTTPAuthorizer → `session/store.go` 的摘要/CSRF/事务 resolver → `identity/login.go` 的 Session/abuse 编排 → Handler 和 production router 的真实包裹次序。
- 逐项对照冻结报告 `:53`～`:63`，确认它把 HTTP 序列化、stub 编排、SQL 形状、真实解析器和持久行为分开；特别复核其 G01/G02 不属于豁免。
- 仅按真实引用补读 S3.2 r06、S1.6 作者自审报告及 `login_integration_test.go:481`、`test/integration/identity/postgres_test.go:1035` 的相关片段。核历史 SHA 和测试语义，不执行 helper/连接数据库。
- 独立核对新增 finding 是否源自原验收、是否存在可复用的等价证据；本次未发现能关闭 G01/G02 的等价来源。作者报告准确保留这些缺口，未据此判定产品实现失败。

复用本会话已取得且同版本的 Reviewer Role、add-code-review、主 delivery、checklist/abstraction、task-execution/multi-agent 及适用 go-service/coding 规则；本轮重新**全文读取产品安全红线**。历史证据有界调查使用 add-research；不外查、不联网。

## 3. 当前 raw 的独立核验

作者证据根目录 **A**：`C:/Users/J00010385/AppData/Local/Temp/han-s33-verification-r01-01a18077/`。

选择集 `A/selection.json` SHA256 为 `8DDC6B2F9E8880A5481C5CF1911AB79BB4F6404D0A46ADF7D06DF55DBB7FAB7B`。其登记时间早于两次 launch；26 个测试体均仍存在于当前对应文件，所在行及文件 SHA 与 source183 一致。

| 检查 | normal | race |
|---|---|---|
| actual wait / exit / timeout | true / 0 / false | true / 0 / false |
| 精确 top / leaf / 不同节点 | 26 / 84 / 97 | 26 / 84 / 97 |
| package 结束 PASS | 4 | 4 |
| fail / skip / missing / extra / 重复 run-pass | 均无 | 均无 |
| 执行前后 183 项及当前源码一致 | 是 | 是 |
| stdout raw SHA256 | `E295BFE5099D690617FBE45BBFD6E80039423171D1D47F991460BB2E635E3B34` | `F28103C70394F37473CC1EFEB2A51BE9F8ABA4FBB0C56A3E5E4921C254D41116` |
| stderr | 空 | 空 |

package 为 `han/internal/platform`、`han/internal/session`、`han/internal/identity`、`han/cmd/han-server`。84 leaf 包含无子例的 top，不相加为 110 个用例。每个节点恰有一次 run、一次 pass，结束记录为 pass；两次 argv 对同一 top 名称并集锚定，含 `-count=1`，仅 race 多 `-race`，没有 integration tag。

仅读取作者已经落盘的白名单环境 JSON；未读取运行环境、注册表或 DSN。该记录无 `HAN_*`，GOENV/GOWORK/GOPROXY 均 off。以上核验是 Reviewer 解析已有证据，不是 Reviewer 独立重跑，也不证明未断言的安全行为。Lead 的 `lead-s3.3-raw-verification.json` 是另一份现场核证，与本 Reviewer 脚本和结论分别保留。

## 4. Findings（未豁免验证缺口）

### [MEDIUM] IA-S3.3-R01-F001：缺少实际登录轮换后旧/新 token 的后续授权及回滚证据

- **状态/分类**：OPEN；关键验证缺口，映射作者 **G01**；STOP S3.3 覆盖充分性/closure，不声称已复现产品 Bug。
- **位置/对象**：冻结作者报告 `verification-s3.3-r01.md:58`～`:59`、`:78`；`internal/identity/login_test.go:347/:763`；`internal/session/store_test.go:332/:563/:613`。当前源码版本见 §6。
- **违反/未满足依据**：I-AUTH-04 明确要求有效旧会话再次登录成功后旧 Cookie 下一请求失效，失败时旧会话保持；I-AUTH-01 要求库中仅摘要。design `:3360`、原卡剩余核证与 ADR-0025 保留这些合同。
- **真实触发场景**：active 用户携有效会话再次登录；成功后分别携旧 token 和新 token 发下一受保护请求，或认证/审计失败回滚后继续使用旧 token。
- **现有证据不足**：Service 测试使用 Session/事务替身的 oldSessionActive 标记；Handler 用 service stub 输出 Cookie/CSRF；Session “rotated old token”等六例预设同一个 ErrNoRows，未消费先前 Rotate 的真实状态。Create 只核 recording executor 摘要参数，没有真实库读回。各局部绿不能等价证明实际 Login→SessionStore→提交→后续 resolver 的行为。
- **历史证据排除**：S1.6 `postgres_test.go:1035` 同时撤销 session **并停用用户**，不能隔离证明 active 用户因本次 login rotation 而拒绝旧 token；也没有认证失败/审计失败后旧 token 仍允许的这组闭环。
- **影响**：目前不能排除跨层接线、SQL/事务语义或回滚回归使旧 token 仍可用、新 token 不可用或失败登录误撤销旧会话；不能确认实际持久存储仅摘要。这是风险未被验证，不是已证明存在该行为。
- **最小补证方向**：在原卡内由获授权执行者使用真实现役 Login/SessionStore/解析链与隔离 PG fixture，固定 active 用户并读回摘要、旧撤销/新提交；旧/新 token 后续请求分别拒绝/允许；认证失败与审计失败回滚后旧 token 仍可用。可先新增 test-only 材料建立失败/覆盖证据；不得由 Reviewer 改源码或自行运行 PG。
- **复审条件**：Lead 给新测试/输入完整清单、固定 SHA、精确新增断言与实际 raw/结束记录；本 Reviewer 对原 F001 和相关回归复审。需要 PG 时必须另获真实执行授权，不能把只读 G0 当作已授权运行。

### [MEDIUM] IA-S3.3-R01-F002：缺少三维持久桶的真实并发原子性、退避及重建后有效性证据

- **状态/分类**：OPEN；关键验证缺口，映射作者 **G02**；STOP S3.3 覆盖充分性/closure，不声称已复现产品 Bug。
- **位置/对象**：冻结作者报告 `:60`～`:61`、`:79`；`internal/identity/login_test.go:691/:741`；生产合同 `internal/identity/login.go:730/:782/:868/:902`；相关 PG 构造在 `login_integration_test.go:481`。
- **违反/未满足依据**：I-AUTH-07 原本就要求单账号多来源、多账号单来源、账号+来源、代理 Header 伪造及重启场景下三类持久桶原子计数，账号维度仍统一失败；ADR-0025 未豁免此项。不是新增容量或真实部署要求。
- **真实触发场景**：多个来源并发攻击同账号、多个账号并发来自同一来源、特定账号+来源独立触发；事务中途失败、实例/进程重建后继续判断封禁和退避。
- **现有证据不足**：`:691` 重建的是 protector、共用同一 memory store，仅驱动单账号多来源局部路径；`:741` 只核 pgx.Tx spy 的 advisory-lock→read 次序。source 429/Retry-After 用 service stub 给出。相关历史 PG login fixture 明确注入 abuse stub，没有生产三桶 SQL 的并发计数、部分失败全回滚和重建持久状态断言。
- **影响**：不能排除并发丢增量/半提交、source 或组合桶不能独立触发、重建后封禁丢失、source/subject 对外分类漂移等风险；内存算法和 Go race 通过不能判定 DB 锁与事务行为。
- **最小补证方向**：原卡内获授权的 test-only 改动分别驱动三维组合与分类，使用同步屏障和真实 store 断言无丢计数、事务失败全无或全有、持久状态在实际实例/进程重建后继续退避；账号维度保持统一失败，来源维度按既有合同允许 429。只用已批准的合成测试阈值，不选择生产阈值或扩大豁免。
- **复审条件**：提供新固定测试/版本和三维逐断言映射、实际 store/并发/重建证据及回归；原 Reviewer 复审 F002。无 PG 执行授权前相关动作 STOP；spy/memory 改名为 integration 或再次全仓绿均不足以关闭。

CRITICAL/HIGH 产品缺陷：本轮未复现。其他独立 LOW/MEDIUM 报告内容缺陷：未发现。上述两条是已由作者准确披露、经本 Reviewer 独立确认的剩余验收缺口，不要求作者把真实限制改写成 PASS。

## 5. 风险分层与历史结论

| 原验收 | 本次足够支持的局部事实 | 不能外推的部分 |
|---|---|---|
| I-AUTH-01 | Cookie 序列化、Handler 200/CSRF、recording executor 只传 SHA-256、固定熵 reader 编码 | 浏览器策略豁免；真实库存储读回未补，归 F001 |
| I-AUTH-03 | Origin 三负例下游零调用；精确同源正例；Guard 两分支清除 ACAO/ACAC；CSRF stub 门禁+真实摘要比较代码+router 次序 | 不是浏览器执行；没有真实网络或 PG 零行快照。当前下游零调用证据只支持受测 HTTP 接缝 |
| I-AUTH-04 | Service/Handler 编排、caller 事务归属、真实 store SQL/摘要的局部形状 | 实际轮换状态的后续授权与失败回滚，F001 OPEN |
| I-AUTH-07 | memory 算法局部路径、锁调用次序、实际来源 provider 的 XFF 忽略/解析/规范化 | 真实三桶并发/退避与重建持久性，F002 OPEN |

S3.2 r06 的历史报告 SHA `0739133B9674B9B03EE76F0E4D0D0D3F81F6BA3D3AA1A40F7A7DEA057DB6EE67` 与作者引用一致；其 handler/login/main 等历史 SHA 已与当前不同，原 Reviewer 明示 PG 是 Lead 上游证据、自己未复跑。S1.6 作者自审报告 SHA `EFBDE5D35FD198888CB4CD40624C5947C1FE9E0486A4E112847BF9D62D40C485` 与引用一致，三个相关测试输入当前 SHA 与其记录相同；但既有用例混合撤销/停用，且本次未获得那次完整外置 raw/wait/UTC，不扩大为当前 login rotation 或独立 PG PASS。

同源历史 Manifest 七项只作项目检查支持，**本轮 NOT RERUN**，不补 F001/F002。真实浏览器为 WAIVED / NOT RUN；当前 PG/integration、SMTP、平台、部署均 NOT RUN。本轮没有网络/安装/运行 DSN/PG 操作，没有清理、Git mutation 或新的 task/Case/Run。

## 6. 当前源码与测试版本、实际 readset

| 文件 | 当前 SHA256 |
|---|---|
| `internal/platform/http_security.go` | `3676B4C64BCF115C43845DC69D77F6875F3A7BBCD67DA6A91920B84594C3CFDA` |
| `internal/platform/http_security_test.go` | `BF3DCEC73440B687ED2A1C68F2B30D0FCEA52A6BF9AF8300E82449B8499E5002` |
| `internal/session/store.go` | `4B12462E29D46B730F0DD58E6A0BAA57EC1D91B99D9F43FC91179319FF88E679` |
| `internal/session/store_test.go` | `3D32E65D407F24211355B642DAB3933F1AB36E10B60CAFC7F039B2F9B8201E09` |
| `internal/identity/login.go` | `FDB1ED13FB499A1219AE24884F053A3626567BB81CAAB56EC02B1FC197B6864C` |
| `internal/identity/login_test.go` | `FB5483FB93649F9B01B060D54F35B5CC7B7A3305704D19DFCA445A2C7400414F` |
| `internal/identity/handler.go` | `EC4533A6CAA6819255BF9FC7D19EEE3AEEAA90B9E3A9128A5D24B5017D1F1EA3` |
| `cmd/han-server/main.go` | `6B45B5509A2D01C9F5DF0393CA00CAD5C8430353B5F849BF6E71BA3A47C52AC4` |
| `cmd/han-server/main_test.go` | `B53DD34D57AD6A2797E9FC080FD90034FC52CB26675686CB075D3153F193A9BC` |

实际精读范围包括 platform 的 Cookie/Guard/Authorizer 与六项选择测试，session Create/Rotate/ResolvePrincipal/CSRF/SQL 及九项选择测试，login 的 issuer/abuse/session 编排及九项选择测试、Handler 来源 provider/Cookie 返回，main production 包裹与两项 router 测试。26 个选择测试体/源哈希另由脚本逐一比对。其余 183 源文件只做完整字节保护，不声称全库语义评审。

本轮独立证据目录 **R**：`C:/Users/J00010385/AppData/Local/Temp/han-s33-review-r01-34df0bf58e/`。

- `payload-index.json`：相对路径/源路径/SHA256/字节数，可复制核验。
- `verify_raw.py`、`raw-independent-check.json`：Reviewer 自写解析器与实际核验结果；不调用 Go/数据库。
- `readset.jsonl`、`reviewed-readset.json`：实际读取区间与适用方法；加载请求和输出截断不冒充全文件审阅。
- `author-frozen-report.md`、`lead-freeze-signal.json`：冻结对象与 Lead 正式版本信号，分开记录冻结接收时 pending 的历史状态和随后 completed/stopped 的正式回交。
- `before.json`、`after.json`、`protection-check.json`、`metadata-guard.json`、`s33-card.before.raw`：源码/安全契约与 S3.3 原卡保护、原读及交回 Tasks/Plan SHA、受控元数据差异。
- `historical-applicability.json` 与被引用快照：只验证历史可适用范围，不制造本次 PG 执行记录。

## 7. 交回与复审

两轮对当前固定对象已完成，作者报告可信且分层准确；S3.3 仍有 F001/F002 未豁免验收缺口，**STOP closure**。没有新增豁免、产品合同改变或实现修复建议采用决定；不替人 accepted/checkpoint，不改任务状态。

Backend Erdos `01a18086-c383-7b72-bcfe-75635bd6c600` 当前仅获 test-only G0 只读与 Temp 权限；本评审不扩大为源写入或 PG 授权。后续补证若形成 184/185 等新输入集合，必须由 Lead 逐文件固定版本与权限，不把本次 source183 哈希直接冒充新版本；再交本原 Reviewer 沿 F001/F002 复审实际断言、raw 和相关回归。

**stopped，交 Lead。唯一仓内新写入为本报告；S3.5 R01～R04 与原独立 index 的 37 项 payload 均未改写。**

## 8. 交回保护收据

封存取证时间（UTC）：`2026-10-28T03:29:28.128471+00:00`。183 项源文件、五份 S3.5 材料和其他四份关键文档均与本轮初始字节相同。S3.3 卡 SHA256 前后均为 `486581BD23D56280476072C4CB0AE20023DB2CF4DC34E0CDC73887BBA95C8517`；59 卡中只有 S3.5 卡改变，其余 58 卡逐字节一致。

| 元数据 | 本轮原读 SHA256 | 已知 CAS 后实际 SHA256 | 实际变更范围 |
|---|---|---|---|
| Tasks | `22C4796844643688FEA82925966656CF51C6CD0FA2885190ADFBC712BAE7ACDA` | `E63E9805F4A51EFCD3CA8C13F65068C3BEA251E7D58EEF07012EA48E6DD95487` | S3.5 卡及 derived metrics |
| Plan | `C9EACB04A51545DB77715C32B6F76C7B234C6407325CF61F87AFF9C66FBD6EBD` | `F54B37B090CE2B25DE7C5210375E81CB99A0AD45E4CD22D8F034590A6AC84C70` | B5C 行及其专属 Pack |

以上与 Lead 指定的新摘要相符，逐 diff 区域核对未发现未知额外变化；这些受控 S3.5 收口不改变本 S3.3 的 F001/F002 或 STOP closure。本文不声称未来元数据永远不变。作者报告和正式 index 也分别保持上列 F9F8… / A0BE…，native completed/stopped 由 Lead 正式通知。
