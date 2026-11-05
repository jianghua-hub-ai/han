# Article S4.8 独立 CODE Review R01

结论：**CHANGES REQUIRED；新增 1 MEDIUM，0 HIGH**。原 Dirac / add-reviewer，非作者评审；objectType=code。本报告不以此前 Design/Mapping PASS 或作者自测代替独立代码判断。

**ARTICLE-B13D-S48-CR-R01-F001 — SQL 回滚后 Publish 仍返回未提交 evidence（MEDIUM，OPEN，交原作者 Noether）**

`internal/article/replay_publication.go:888` 在提交前给命名返回值 evidence 赋值；`:903/:906` 的错误返回保留该值。真实 evidence INSERT 后，current UPDATE 或 publication INSERT 被实际 SQLSTATE23514 拒绝时，调用者收到非零 ref/digest，但事务已回滚、该 evidence 未持久化。旧不可变 Issue 消费者 `test/integration/recovery/replay_evidence_test.go:506–508` 明确要求失败返回零 evidence；新回滚助手 `internal/article/replay_publication_integration_test.go:1379` 丢弃返回值，漏掉此原有断言。

独立实际 PG 探针普通/race **每模式健康 1 PASS、两条回滚 2 FAIL**，无 skip/unfinished/race 诊断。触发前确认 evidence INSERT 已发生；九表 rows/xmin/ctid 前后精确相同，NOWAIT 确认 run 锁释放，排除“未触达实际 SQL”或“回滚失败”。影响是返回合同回归和测试盲区；当前 provider 仍检查 err 并独立读回，**未建立数据库损坏或 readiness 绕过**。最小建议：原作者恢复失败时零 evidence，保留 unknown/原 cause 与独立精确读回协议，并在原测试中恢复结果断言；以精确修复 SHA、真实两回滚＋健康正控普通/race及受影响 required 叶复审。详见 `findings.json`、`probes.fragment` 和 `checks/probe-pg-return-{normal,race}/{stdout.jsonl,result.json,parsed.json}`。

## 两轮实际范围与直接证据

Round 1：完整审阅当前十源（6498 Python 逻辑行）、完整十源差量。四个新增文件的 diff payload 与已完整读正文一致；六个既有文件 diff 2980–3694 行完整读取。独立复核 before/after/live、文本差量重放及原两个 unit 文件前缀保护；原始字节 SHA 与换行保护单独核对，不把 normalized diff 冒称原始 CRLF。精确十源入口/退出均一致，见 `source10-entry.json`、`source10-exit.json`、`static-proof.json`。

Round 2：沿实际 Replay provider→Store→same pool/typed 配置→同一 Source journal/snapshot→Runtime opaque permit/Coordinator/Admission→controlled-open/Anchor 消费链，核独立 checkpoint verifier、first-use seal、普通读历史与双 revision、三表事务、ACKunknown readback 和旧 Issue 消费者。实际涉及原十源全量及 `readset.json` 列出的必要中性生产者片段。没有重审未变整库或接触并行 cmd5。

除 F001 外，在已审局部未建立新增缺陷：构造零端口 I/O、typednil/same Store pool、旧入口首次使用封存；17 字段 canonical 的三个独立 golden 重算；组合 FK/immutable；历史 prefix 可合法 head advance、read 间 drift 拒绝；普通 ctx required reads 与禁止动作计数；actual Runtime.Install/同受控 handle/独立既存 checkpoint verifier 与前后复核、真实 IO join；missing legacy 仅禁 Import 及其余动作正控均有对应代码与本轮实际覆盖。synthetic facilities/checkpoint 不冒充真实批准源；legacy now-reassert 不等于过去已有批准。

## 独立实际验证

所有 Go 均 GOMAXPROCS=1、-p=1、offline/local、serial one job、-mod=readonly；-vet=off，普通 DSN 为空。PG 只授权 Article 库；固定 0001/0002/0003/0005/0009/0011，独立新 schema 保留。完整命令/raw/逐叶/父 Wait 见 `checks/*`；旧新 required 逐名保护见 `required-leaf-protection.json`。

| 层/范围 | 普通 | race |
|---|---:|---:|
| Required U/C（新23＋owned旧31） | 54 PASS | 54 PASS |
| Required P（新52＋owned旧6） | 58 PASS | 58 PASS |
| 独立错误返回探针 | 1 PASS / 2 FAIL | 1 PASS / 2 FAIL |

五组依次为 PublicationCurrentEvidenceAtomic、CompositeFKImmutableCanonical、ExactHistoricalClosureAllowsRequiredReads、HeadAdvanceAndReadDrift、LegacyReassertionVersusTrustedImport：每模式 U/C 命中 **11/8/1/1/2**，P 命中 **13/1/8/2/28**。Required 无 fail/skip/unfinished，race 无竞争诊断；不将 required 全绿用于覆盖独立失败。旧 Issue 两顶层不在授权执行范围，**NOT RUN**；仅静态取得原返回值 oracle，owned 六 PG 叶已真实执行。

只追加 test overlay，原完整测试 prefix SHA/字节保持，未覆盖生产代码/改旧断言。实际 Article 物理预检及本轮 **122/122** 自有 schema 逐名 SELECT 均 PASS；无 glob/删除。八条有 result 的命令均真实父 Wait 返回；probe fixture 清理后两池 acquired=0，Rows/Tx/worker/action/journal 按实际 fixture 收口，不宣称所有 Host/别角色已 join。C 初始 race 编译遇磁盘满未运行产品测试，原 raw 保留；未清理，转自有 D process Temp 重跑成功，详见 `host-limit.json`。

## 边界与交接

固定 Han 工作区；main/HEAD6da8abb… 是本次授权定位，未用 Git 独立核分支/HEAD。十源及设计 r27/Article TP/Tasks 精确退出 SHA 见 JSON；本角色仓内零写、无源码修复/SSOT 采用/Taskclose。全局7、独立 vet/fmt/Linuxcompile、旧 Issue tops、具名 native 正控、完整 S4.13/root/S4.16/provider/批准 ACL/E2E、人的 checkpoint **NOT RUN/unknown**，不倒挂为本局部的新前置，也不冒 PASS。此前 Design/Mapping 结论原边界保留。

**STOP。交 Main 接收并回原作者处理 F001；本轮不签 CodePASS，不继续扩跑。**
