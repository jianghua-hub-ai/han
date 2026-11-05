# Article Admin2 独立 Design Review R02

日期：2026-10-27。Reviewer：原独立 Reviewer Pauli，非作者 Hubble。objectType：`design`。本轮沿同一评审上下文，复用已完整读取的 Reviewer Role、`add-code-review` design 分支及 R01 适用规则；未重路由、未加载后续 Workflow、未下级委派。

**结论：PASS（仅固定 Admin2 设计增量的有界复审）。原 MEDIUM `AM-DES-R19-ADMIN2-R01-F001`：CLOSED，设计修复成立；无新增 findings。** 此结论不代表 Admin2 代码已实现、Handler 已运行、测试/PG/root/E2E PASS 或人的 accepted/checkpoint；N001 仍是实施注意，不是第二项缺陷。

## 1. 固定修订与基线

工作区固定 `D:\AID\VSCode\Codex\Han` / `main`，本轮只读核验 HEAD 为 `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。

| 对象 | 本轮实际 SHA-256 |
|---|---|
| `.add/project/knowledge/design-workspace/article-management/design.md` r20 | `6884431B6164140E5593A8253AC20E8EB1C59C1AD71B86558B6D1A8C99BF037A` |
| 同目录 `stories.md` r09 | `A2BA7746F30B1C9B60DB4337995FC0A0E52DD55BFF984BAA784C20331D80C569` |
| 原 `review-design-r19-admin-query-r01.md` | `5BBD0D2341B1A54A2324220627FB2F1DCEBC6C72F074BBA32516FC04B0A20EC4` |

基线 manifest 为 `C:\Users\J00010385\AppData\Local\Temp\han-article-admin2-r20-20261027T150417378Z\baseline.json`。其 design/stories before-image 分别与 Lead 固定目录 `C:\Users\J00010385\AppData\Local\Temp\han-admin-query-lead-eb1adcd2a0d94f60b8848c9a459a43b6\` 中 `design-r19.md.txt` / `stories-r08.md.txt` **逐字节相同**，且各自匹配 R01 固定 SHA：

```text
design r19  19A06AE6FB78DB1C4F0F31A34FCFF789936AE467004CA660C2F211D14688C3CA
stories r08 9AAD29B0075C387195B99AC4F35D8AF8B5A5C1253D57E4D4FBF6CC85F3C3F361
```

design 由 2881 行变为 2894 行，stories 由 263 行变为 266 行；两者前后均无 BOM/CRLF。实际差异仅为修订/整改来源、HTTP 完整解码合同、N001 轻注、§8.6/Story8 验收及确认记录同步，未混入其他 Story/SQL/状态方案修改。

## 2. 两轮实际复审

**Round 1——修复 diff：**读取 r19→r20 与 r08→r09 全部实际 unified diff，逐项对照原 F001 的触发、最小修复及复审条件。主要落点为 design :1977–1993、:2001、:2728–2759，stories :211/:216；版本/来源及 pending 确认记录同时核对。没有重跑全特性审查。

**Round 2——限定接缝与新风险：**重读 `handler.go:679–719` 的原丢错参数助手/limit 转换、:595–628 的 sentinel 映射；对照 r20 的完整 map 读取与新入口局部错误转换。重读 R01 标准库探针源及同会话原执行证据，静态核对其 ParseQuery 报错路径与本次失败分支；本轮没有重新运行探针，也没有将探针视为 Admin Handler 执行。核五个非 root 依赖的全字节 SHA 与 R01 一致，确认没有需重新追查归属的变化；核 §8.6 的五生产/四测试承载合同没有扩大。

Dalton 的 S1.6 已获用户授权修改 `cmd/han-server/main.go`、`main_test.go`，与本设计复审并行。本轮**未读取或计算这两文件的当前哈希**，未审其代码，未要求保持 R01 root SHA；它们的授权变化不是本轮固定设计对象漂移。原五生产/四测试足够性仅按未改变的设计依赖合同保留，不声称已复核并行 root 实现或其适配结果。

## 3. 原 finding 复审

### `AM-DES-R19-ADMIN2-R01-F001` — CLOSED（design）

原依据：r19 指定按 `singleArticleQueryParameter` 的 presence 识别遗漏，而该助手的 `URL.Query()` 丢弃解析错误；显式非法 view 会变成默认 all，混合重复参数可能变成单值。

| 原缺口/触发 | r20 修复位置与实际合同 | 复审判断 |
|---|---|---|
| 参数被部分解析或误当遗漏 | design :1977/:1979：新 `ListAdminArticles` 在可信管理员授权后、typed/service 前，直接一次性 `url.ParseQuery(request.URL.RawQuery)`；任何 error 都丢弃可能返回的部分 map，包括未消费键的错误 | 错误不能进入 presence/default 分支；没有先过滤键、修复输入或继续部分成功 |
| 重新调用旧助手再次丢错 | design :1979：解析成功后只从同一完整 `url.Values` map 判断键存在、切片长度和值；明确禁止调用重新消费 `URL.Query()` 的两个旧助手 | 新入口接缝可实施，旧 member/orphan 行为保留 |
| 五类原始输入反例 | design :1983/:1993、stories :216：`view=%ZZ`、`view=hidden&view=%ZZ`、`view=all&cursor=%ZZ`、`view=hidden&limit=%ZZ`、`view=hidden;view=violating` | 全部优先整体 422，零 service/repository/identity；不采用部分 map，不回首页/默认值 |
| 整体解析失败与字段失败优先级不明 | design :1979/:1983–1993：整体解析失败统一既有 `422 article.validation_failed`，含 cursor 编码失败；完整解析成功后的 cursor 内容/绑定失败才为原 400，view/limit 仍为原 422 | 与本轮用户采用的错误归属一致，不将 cursor=%ZZ 同时承诺为 400 |
| 正常缺省/typed 零值被修复影响 | design :1984/:1986–1991：只有完整解析成功且无键才用默认；单个精确 view、原 Atoi/resolveLimit 语义及 typed 零值 all 保留 | 未把 HTTP 错误折叠进 typed 零值，也未增加生产数值默认或更严格整数词法 |
| Story/验证只测试枚举，没有真实入口要求 | stories :211/:216、design :2752：同步五反例/正常对照与零下游调用，要求后续经实际 Admin Handler 核 HTTP 输出 | 设计层同步完整；未要求尚未实施的测试先 PASS |

关闭依据是设计已消除原丢错复用合同，并给出足够明确的失败次序、错误码和验证责任；不是风险豁免，也不是判断尚未实现的 Handler 已经修复。

## 4. 新变化与边界

N001 的 design :2001 补充成立：现役 `MapArticleError` 在 :611–612 先按 `errors.Is(ErrArticleInvalidState)` 映射治理 409，r20 规定新读取非法状态转换为不包装/不连接该 sentinel 的独立低敏 internal cause，经原 `WriteArticleError` 输出 500，并保留旧公共映射/治理 409。此轻注明确了兑现原 500 的局部落点；本轮仅静态确认可实施，没有构造或运行新错误输出。

本轮完整解码发生在授权后，拒绝时不构造 typed 请求、不访问下游；不回显 RawQuery 或原始解析错误，未增加敏感日志。对未消费键中的编码错误也拒绝，是本次已明确采用的整体解码合同，不是未知键白名单或新业务需求。

五生产文件仍为 `internal/article/{query_service,handler,model,policy}.go` 与 `cmd/han-server/main.go`；四测试候选仍为 `internal/article/admin_query_test.go`、`internal/article/policy_test.go`、`test/integration/article/admin_query_test.go`、`cmd/han-server/main_test.go`。F001 落于新列表 Handler 的局部完整解析/转换，N001 落于新读取错误边界，均可在原范围承载；无新增 service、SQL、identity reader、迁移、共享映射或公开错误码修改。原 16 endpoint/S1.5 全部关闭条件、S1.6 三项独立接线、共享 main/main_test 串行 owner 要求未被缩减或代替。

**新增 findings：无。** 未新增必须代选的方案或方向决定。实际成果确认人/接收人仍 unknown/pending；用户/Lead 对修复方案的采用与独立设计 PASS 分别记录，不代签 accepted/checkpoint。

## 5. 本轮实际 Readset

| 读取对象 | 实际范围/用途 |
|---|---|
| r20 `baseline.json` | 完整 manifest；两个 before-image 全字节及摘要 |
| Lead `design-r19.md.txt`、`stories-r08.md.txt` | 全字节，与 author before-image 比对、作为 actual diff 基线 |
| 当前 design r20 / stories r09 | 全字节用于固定摘要/diff；逐段阅读全部实际 diff；另带行号读 design :1973–2005/:2728–2760、stories :207–219 的修复上下文 |
| 原 R01 报告 | SHA 核验；:61–93 与 :139–153，原 F001/N001、足够性及 Gate/限制 |
| R01 Temp `review-r01-readset-hashes.json` | 读取已有哈希登记，仅对下列五个非 root 源码作本轮比较 |
| `internal/article/handler.go` | 全字节 SHA；正文 :595–628/:679–719 |
| `internal/article/query_service.go`、`model.go`、`policy.go`、`internal/identity/article_reader.go` | 本轮仅全字节 SHA，与 R01 相同，复用原有界审查结果；未重读正文 |
| R01 Temp `review-query-parser-probe.go` | 全文，复用同会话 R01 的标准库解析证据；本轮未执行 |
| Role/Skill/适用 Rule | 复用当前会话已完整取得的同版有效正文；未额外扩读 |

五个非 root 源码实际摘要如下，均与 R01 一致，交回前再次核验：

```text
internal/article/query_service.go 9674130673CD05E8FA864B77528FDBF12C9AB8305AE6AD2EF7E2A85DB9136E7E
internal/article/handler.go A0921B0178551A8EA51772ECF9823B50AA693DD3B450FDD347257B6F77CCC7E8
internal/article/model.go 3B5D3C01BA5BC56B8E4486C7F4FFD35BAAF6735A1DD98668D5097D3E4A4E68B4
internal/article/policy.go B5A199340C18B505B8F2633715C9EC63B78910FB684332061CCE0258E12FC980
internal/identity/article_reader.go D9315AFA411CB8C4612459C8E97979BD2274780A7B7B32D78F3AB735C3F82B6B
```

未读并行 root 当前文件、Task/Plan、其他测试正文、身份文档/ADR-0025、架构/ADR 或其他特性；没有扩大消费者范围。Git 仅只读 branch/HEAD；Python 仅作摘要、字节/diff 和行号摘录。唯一仓库写入为本 R02，原 R01 及失败证据保持原样；未写其他仓库文件或执行清理。

## 6. Gate、限制与交回

| 项目 | 结论 |
|---|---|
| 固定对象与基线 | r20/r09 均匹配；author/Lead before-image 字节相同；交回前对象与所查非 root 依赖无漂移 |
| 原 F001 | CLOSED，设计修复；原 R01 CHANGES REQUIRED 历史保留 |
| 本轮有界 Design Review | PASS，无新增 findings，不扩称全特性重新评审 |
| 五生产/四测试边界 | 合同不变，保留 R01 足够性结论；不是任务登记、认领、代码完成或当前 root 代码评审 |
| N001 | 非阻断实施注意保留，未声称 HTTP 500 已实际验证 |
| 人的 accepted/checkpoint/承接确认 | pending，未代签 |
| Admin2 实现与验证 | 未实施；未运行项目测试、PG、服务、图渲染、部署/E2E；没有对应 PASS |

剩余风险与 R01 的运行证据边界一致：实际 Admin Handler 完整解析与零下游调用、字段错误优先级、真实 WriteArticleError 500/旧治理409、SQL/身份/分页/cursor/DTO/ETag/root 安全链及资源释放仍须在后续实施与相应验证中取得证据。并行 S1.6 代码正确性与整合结果不在本报告结论中，不为关闭本设计 finding 要求其预先完成。

若后续修改本次解码/错误合同、旧消费者行为或五生产/四测试边界，须以新固定修订返回有界复审；若只是按本设计实施，则按后续实际代码/验证范围独立取证。本报告交回原活动 Lead，由其现场核成果和下一动作授权，不启动实施或更新任务状态。

**stopped：Pauli R02 有界复审完成，停止。**
