# S1.5 Components CodeReview R02：F001复审

2026-10-28。原实际独立Reviewer Erdos / add-reviewer；作者 Dalton。objectType=`code`，续接同一development-delivery。技术修订/metadata包来自 **Lead交接**，本报告另列实际独立核验，不写成用户逐项直接交接或人的接受。

**有界 CodeReviewGate PASS；AM-S15-COMP-R01-F001 CLOSED，无新增findings。** 仅复审r03的F001最小修复及相关消费者/检错与回归。full S1.5、S8.2、Case18真实层及人的确认保持原blocked/pending，不因本局部PASS关闭。

## 1. 固定修订与保护范围

实际核 Han/main、HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。原 [R01报告](review-code-s1.5-components-r01.md) SHA=`D88CCCC9B6448E154266B00E480E2D998F763E64D87E8660463D2A2795A01938`，原失败链、对照日志及Temp见证全部保留，未覆盖或清理。

| 固定文件（均在 `test/contract/article/`） | r03实际SHA256 |
|---|---|
| `rest_contract_test.go`，唯一变化 | `CEBA105C2CBF70B91CB5DFF93D9D4FC37077153A7EDB4F9C2806C29E650E9533` |
| `internal_ports_test.go`，不变 | `DC6791D4793ABE67FE17B83736FC77E55C347859B13A4FB5071E6AC51BF64F25` |
| `projection_contract_test.go`，不变 | `D7585EE66538EECA4364BF01CC2BE3A4AC69D98C9818560D6533457E8A572033` |
| `replay_status_producer_test.go`，不变 | `B519DDE178CB3F4357560681490FD00639549E8CA0B369E2EFB01785488D8FDC` |

证据根 F=`C:\Users\J00010385\AppData\Local\Temp\han-s15-candidate-c63dac5fc52d4ea0af1d51efb9c76c18\fix-r03`；本角色唯一新Temp写目录 `F\review-r02`。实际核当前178项与 `fixed-178-r03.json`、`fixed-inputs-178-r03`全一致；相对R01固定178仅REST hash变化，其余177相同。另定向枚举cmd/internal/test/migrations/testdata中的Go/SQL/JSON，加module与migrations README，实际集合178，无增删；contract目录恰原四Go文件，无虚拟Reviewer第五文件落地。

source map SHA=`E88DC26D92259E3C147F644E27FDE4E4079899A3428B9A7D74FB8A16E70D470E`；rest-only.patch SHA=`4BDED77D53A90D879BB7562373EE869A2C8481D0AA260FC3195B34F067976005`。由R01前像及当前REST独立重建diff，去BOM/换行规范化后与原patch相同，实际3add/2del。没有把规范化文本比较冒称原始字节SHA相同。

r18/r21/stories/Pauli规划Review沿原固定SHA；复用R01已完整取得的Reviewer Role、add-code-review/current delivery、checklist/abstraction/安全红线、coding/no-hardcode/observability/tracing及唯一启用go-service适用规则。本轮只刷新必要delta，不整库重审、不重审规划/身份/恢复，不以readchecks产生共享root写窗口授权。

## 2. Round 1：实际修复与原finding

F001原问题：N/Nplus1 fixture长度依赖生产 `article.MaxArticleWriteBodyBytes`，实现上限漂移时测试预期跟着变化。r03 `rest_contract_test.go:503` 新增局部独立 `const wireContractBytes = 262144`；`:513–514` 两个fixture仅改为使用此测试合同值。其余wireAssertion、真实Handler调用、400/零下游、成功请求字面值核对和四top结构不改。

该修复独立固定design §2.4.4/r18 §4.7.1规定的N，而非读取/反射生产常量。不在父组提前比较常量或Fatal，不用静态数值检查替代两个真实请求。未更改生产handler上限、解码顺序、HTTP合同或其他测试；没有为复审新增共享见证文件。

## 3. Round 2：消费者、行为敏感性与关闭依据

复核真实路径 `restWire → wireAssertion → component/serve → Create/UpdateArticle → decodeArticleWriteJSON → port`；已有helpers沿R01同版完整审查复用，补读当前wireAssertion及未改的 `handler.go:22/:820` 上限/有界读取/长度拒绝/UTF8/JSON链。负控仅在个人Temp副本将生产常量262144改为262145，当前四测试和其他编译输入保持r03固定版。

个人overlay含156个Go输入，除handler一行数值外均匹配r03 snapshot；没有改断言。生产mutant SHA=`910652ED2F532CAB36064BAFF5FA15CE626D4645AFE11FB1FEBFDCAB6D513186`；个人overlay SHA=`A6723AD8E1A1FC1AC682EAC17B758BC5BA9517F53C934EEDF74B83AEF0959873`。该语义变体与原R01暴露问题一致，按新178输入重绑，未误用旧REST断言overlay。

**独立实际负控结果**：WireAndStrictJSON共24run、20leaf，exit1；Create/Nplus1、Replace/Nplus1分别实际HTTP201/200而预期400，在`:525`的真实HTTP断言失败。其余18节点pass，包括两个N叶；共6个fail节点包含父层，不冒称6个独立缺陷。不是编译/环境失败，不是父级提前Fatal，证明修复已检出原盲区。

原源码普通/race中Create/N、Create/Nplus1、Replace/N、Replace/Nplus1四个实际叶全部pass；Nplus1请求在现源被400拒绝，沿原断言核零下游。原R01“286节点仍GREEN”的漂移对照链保留；新修订对同类漂移实际RED，满足原复审条件。

| 原finding | 本轮状态 | 依据 |
|---|---|---|
| AM-S15-COMP-R01-F001 / MEDIUM | **CLOSED** | 独立定值fixture；真实N/Nplus1四叶现源GREEN；同上限漂移两个Nplus1叶真实RED、两个N仍pass；178固定/177保护及相关完整组件回归通过 |

新material findings：无；新方向决定/必改风格建议：无。原三文件与R01其余消费者/层次结论复用，没有由这3add/2del重新签全业务或外部层。

## 4. 本角色实际命令与集合

仓库根执行，所有child按键排除HAN_TEST_POSTGRES_DSN后才读取其余env值；未查询/打印DSN或secret，无PG/SMTP/service/network/Browser。每命令meta保存argv、exit、时刻、raw SHA及178项before/after，均与固定r03相同。无需重复Manifest7。

```text
go test -tags=contract ./test/contract/article -run '^(TestArticleRESTContract|TestArticleInternalPortsContract|TestArticleProjectionContract|TestArticleReplayStatusProducerContract)$' -count=1 -json
go test -race -tags=contract ./test/contract/article -run '^(TestArticleRESTContract|TestArticleInternalPortsContract|TestArticleProjectionContract|TestArticleReplayStatusProducerContract)$' -count=1 -json
go vet -tags=contract ./test/contract/article
go test -overlay <个人Temp>/wire-mutant/overlay.json -tags=contract ./test/contract/article -run '^TestArticleRESTContract$/^WireAndStrictJSON$' -count=1 -json
```

| 个人label | exit | 实际结果 |
|---|---:|---|
| normal | 0 | 4top、23groups、16Endpoint分支；286run/pass、236leaf、0fail/skip |
| race | 0 | 同286/236/0fail-skip，完整run/leaf集合与普通相同 |
| vet | 0 | stdout/stderr空 |
| wire-mutant | 1 | 24run/18pass/6fail/20leaf/0skip；仅两个Nplus1叶失败、两个N叶通过 |

逐行解析JSON核真实命中，不把父节点/包pass重复计leaf。四top分布不变：REST182/149、Ports62/53、Projection15/11、Producer27/23；23组actual leaf与r03 case_map一致，16Endpoint实际分支保留。个人与作者r03精确四top及全包普通/race的完整run集合一致。

个人stdout SHA：normal=`A79A0E26CDF3CFC8A8245B64565C823CD7FC25793B67C6CF29361003873F1070`；race=`74997F7EF9A7FF425D7D9CD6C444E90C31870A494B822B227FF347ADB5E9DCF6`；wire-mutant=`272458B88F0DD8FCF8D8DC0EE410C7B3A387AFF644F17044769E0D244152C89C`。全部个人stderr与vet stdout均0字节，SHA=`E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`。

个人raw目录 `F\review-r02`，`raw-index.json` SHA=`72899BD34D6E6D8F5175D71BF0A485F5D80DB0DFBD0FF98B98AB65BEB8F32A86`；`raw-run-verification.json` SHA=`CD9546705FA0B850A5CE4E7A0225C4D97F05C19144D4F2812EADF6954B8E4BA0`；`overlay-verification.json` SHA=`F9EFB2FA5C01D5866771EE458385390603AA7DE30EE0CE9A5B12DC01E4626330`；`source-review-binding.json` SHA=`8D04C11190EED88E90964578DC499A167B9C7FC9E51FCE5BE560ABC4E62CDC4E`。源码集枚举/四文件存在性另留source-set-verification.json。hash-only的178绑定不等于178份全文审读。

## 5. 作者/Lead证据核证及未验边界

实际读取stopped/G0、F001-author-fix、actual-readset、修复diff、新snapshot/index/case map、三负控和对应raw。独立复算stop声明84个artifact SHA全match；raw-meta-index为19runs/57个raw-meta，SHA全match，19份meta均before=after=当前178。作者全包/精确四top普通/race四份JSON实际286/236/0fail-skip；集合与个人相同。

三条作者156Go overlay均按完整r03 snapshot重绑，四断言SHA不变：m1只改model/policy暴露作者version，两key leaf编译后FAIL；m2只改ReadStatus Ready→Unavailable，Ready20 leaf编译后FAIL；m3只改wire上限，同个人负控实际两Nplus1失败/N通过。三份实际源diff和overlay未知/额外变更均核过，未将脚本/quote/行尾字节对比等操作失败算RED。原R01问题RED及全部旧材料未删除覆盖。

Lead **新r03** Manifest回执实际SHA=`8904DA4630D4441B7EBE06C15A86FD24C49190CBE46AFCC3CFE7B06F37E61047`，核format/modules/vet/build/contract/race/unit原顺序全exit0、format空、178 before=after当前修订。它是Lead执行/本角色核回执，不冒称个人执行；没有套用r02旧0DC回执或重复跑全七项。独立源码枚举也确认178项集合无增删。

本轮实际readset严格限F001 delta、wire helpers/真实decoder、固定输入/原R01与r03证据及新Manifest回执。其余三测试、旧root、端口/proof、投影/producer及原同版规则沿R01已取得结论复用，不展开全库、身份历史、未来E2E或Task/Plan全文。

本PASS仅关闭F001并通过本次固定组件复审。真实审计INSERT失败、应用身份ACL、实际PG/source ReadCurrent/Publish、全16生产root新专项、真实Session SQL、Journal真卷/旧备份、全局恢复隔离/完整恢复/S16/二次核验/MarkConsumed后开流、Host/部署E2E/容量与人的确认未因此取得。full S1.5全部12requires、S8.2/Case18阻塞原样保留；没有placeholder/skip或局部GREEN外推fullPASS。

唯一仓库新写入为本R02，R01原件/对照证据保留；未修改被审源码、生产/identity/root、metadata或他人报告，无Git写/清理/委派。个人schema=0，无后台服务；全部个人日志/Temp生产变体保留。

**stopped：F001独立复审完成并CLOSED，交Lead核结果与后续门禁，不关闭任务、不代签人的接受。**
