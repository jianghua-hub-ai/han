# S2.4 受审补充精确对账（2026-11-04）

已在原S2.4卡与B11唯一当前指针采用受审addendum §6，不新建任务、批次、文件写集或依赖。S2.4仍active；86卡与74 completed / 3 active / 9 todo不变。Backend Feynman继续Lead的SOURCE/LOCAL/PG窗口，本次没有等待或介入其实施，也不冻结其四test的哈希。

## 依据与真实责任

- addendum：15122 B / `2C3F874B43653D4E31C4FD65AA21AAB91CA795386413235E829AD4057B96C774`；初交接15094 B仅过时长度，按现场当前值登记。
- map：25810 B / `383215FA893DC9AC0A8CD8777B4B71C22C1BA411DD79DEF864FBC03B4CE95AD7`。
- 非作者Halley / add-reviewer：实际ID `01a1a495-be38-7201-a1d7-5c251cf16900`由Lead提供；同目录上两级的 `test-plan-review-r01/report.md` 12664 B / `ECBDC101942E62CD7DE14167889EBA1A0413D737CCF1CF6DF4802BED9C5DFDAA`，`verdict.md` 1903 B / `6A406752E1BDCBBDC3D3E5C8C03CC0BC379D74ACC6DC3DD195E155C3E630F1B1`。已实读两份结论；PASS / 0 finding仅预算、timeout与计数delta，Lead本轮明确已实读接受并授权登记。
- 原r04仍76409 B / `EEB026E5FEE5AB0FE207BD9ABC66C3E2F9227EF5112F226286BF5E6EAEB6FC17`；原评审、旧证据和相邻原planner/handoff/verifications不回写。18份固定元数据/旧Planner/评审对象摘要核无漂移，详见scope-baseline及verification；不把这个集合称为Lead的8旧证据检查。

## 实际替换

| 项目 | 当前原卡及B11指针 |
|---|---|
| medium外层timeout | 75m→80m，仅收尾余量，实际共同90min不延长 |
| 磁盘 | 1/5.5/8.5/15 GiB，明确GiB=2^30；1073741824/5905580032/9126805504/16106127360 bytes；校准归small、共同/分档累计不因进程/runid/重试重置，PGDATA真实余量及预留照核 |
| 集合与inventory | A/B/C共41逻辑/42展开schema场景单元，small9/10、medium32/32；校准K按真实调用实物另记。42+K与未来51+K仅无重试、每单元恰建一schema时的条件式，不是新硬总数。逐scene/variant/attempt/schema和实际库内清单对账，失败/重试/半成品/自适应调用全部留存 |
| large/canonical | 本轮D8+canonical明确NOT RUN；优先small/medium准确闭环和非作者独立Review，避免额外60min；large命令仅保留接口，不虚构已创建schema |
| 保留采样与时间 | 20 TIMED、15 SINGLE、6逻辑正确性；每次warm-up≥10s与真实活跃测量≥30s、TIMED三重复/每点≤120s、SINGLE单次≤60s、软N500；X01两fixture、D04-100及N02/W02-c4不减。共同small+medium90min、large独立60min、145min停新点、150min取消在途硬停不变 |
| 关闭条件 | 当前修订实现/检查、预算内校准+完整small/medium、实际inventory/库内与资源关闭证据、large原因、批准docs片段、非作者代码/执行证据独立Review及Lead同revision Gate继续必需 |

这些都是规划/任务元数据对账，不是作者G0实现缺陷核销、Go/PG执行、性能、资源关闭、容量/SLO、E2E或整卡PASS。原S1.4继续独立承接部署后E2E，S1.5/S8.2及其他finding不借此关闭。没有为收尾扩大Backend写集或运行窗口。

## 范围核证与交回

可复查命令（仓库根目录，只读文件/Git，不运行Go或DB）：

```text
python .spec/wip/article-management/evidence-pg-perf-r04/continuation-20261104/planner/addendum-reconcile-r01/verify_reconcile.py
git diff --check -- .add/project/knowledge/tasks/tasks.md .add/project/knowledge/plans/ir-01-content-community-batches.md .spec/wip/article-management/verification-s1.5.md
```

Python核证PASS：Tasks除S2.4与S8.2一条F005接收外的字节相等，Plan除B11当前指针外相等，verification原字节为当前精确前缀；全部86任务标题/checkbox及统计未变，S1.5整卡逐字节未变。Tasks/Plan原UTF-8无BOM、CRLF保留；verification原UTF-8无BOM、LF保留。只追加的F005与本次S2.4分开保存前像、结果和出处，不回写旧Planner证据。

F005具体接收及43/43接收时点/verification_context追加后摘要边界见相邻 `../f005-reception/registration.md`。最终Lead现场复核及Backend后续实施/Review归其各自证据；本角色完成本轮有界登记后STOP。新增证据保留供本卡交接追溯，未授权清单前不删除或归档。
