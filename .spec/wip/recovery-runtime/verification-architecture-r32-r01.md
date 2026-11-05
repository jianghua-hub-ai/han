# 全局恢复架构 r32：Lead 接收与详细设计交接

2026-10-29，固定 Han/main，HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca`。本轮沿既有 development-delivery 继续，依据用户“待确认或决策采用建议值并继续”的授权采用技术建议；没有重新取得或补写人的成果签字。

本轮完成 [architecture r32](../../../.add/project/knowledge/design/architecture.md)、[ADR-0027 r02](../../../.add/project/knowledge/decisions/framework.md#adr-0027) 及决策索引的架构闭环。默认关闭、首装可信证明、正常重启的历史关闭证明、完整恢复消费、请求/worker 排空与部署独占已进入正式合同。identity 保留唯一 RecoveryRunStore 和业务结果责任；框架提供机制，article 维护删除日志及其独立锚点，platform/root 消费同一准入结论。

## 独立评审与现场接收

架构作者为 Pascal（add-arch-designer，`01a17e3d-5550-7f91-bb7e-6e6fad3d082c`）；非作者评审为 Kuhn（add-reviewer，`01a1835a-8010-7a73-a8dd-3192719b6965`）。Rawls 的测试工程咨询只提供原验收约束，不计为评审或执行测试。

- [R01](evidence-architecture-r32-r01/review-r01/review-report.md)：CHANGES REQUIRED，MEDIUM `ARCH-R32-R01-F001`。真实 article Journal 没有独立 Anchor 写入能力，“仅补历史读取接口”的后继范围不能满足双 namespace 防回退。
- 原作者修订了锚定写者、耐久确认、运行中屏障/故障通知、受控 pending 协调及可信存量迁移；只改同三文档，未接受风险豁免。
- [R02](evidence-architecture-r32-r01/review-r02/review-report.md)：F001 在设计/ADR/index 层 CLOSED，新增 finding 0。启动 Reconcile 与恢复 ReplayAll 的真实调用链及 FD 承接条件已单列核查。
- [R03](evidence-architecture-r32-r01/review-r03/review-report.md)：最终状态及相对链接的元数据核对 PASS。三文件各仅替换一行，技术合同与版本不变。

Lead 全文读取全部实际增量、修订差异和三轮评审报告/receipt，并现场核对相关 Journal/worker/finalizer 消费者。最终三 SHA 与 R03 一致；新增链接有效、编码/LF 保持、ADR 实际采用及索引均为 1。188 个源码树文件及 go.mod/go.sum 并集 190 项、其他 30 项知识材料，以及本轮原始基线中除三文档外的 4,490 个文件均未漂移。原版本与失败报告按各自索引原样保留。

Lead 补充文件集合检查首次因基线分类不同触发断言：Main 的 188 项包含 go.mod/go.sum，作者的 188 项源码树包含另外两份既有 README。已用原始前像交叉核对并集 190 项，零新增/缺失/变化；未更改原基线。初始失败和更正依据均保留在最终接收材料中。

本轮没有改源码或执行 Go/PG/服务/HTTP 测试。PlantUML CLI/jar 缺失，图块配对与链接只作静态核验，语法检查/渲染 NOT RUN。技术接收仅覆盖架构合同和评审闭环；完整恢复实现、真实介质/CAS/迁移来源/备份与 inventory/应用 ACL/epoch 与 retirement/RPO-RTO/部署 E2E 仍各自未验。人的成果接纳未知，未调用 checkpoint 或代签。

## 详细设计的具体输入与完成条件

下一接收角色为 add-feature-designer，输入为最终 r32/ADR-0027 r02、identity design r49、article design r22、现役边界与测试计划，以及 R02 的消费者解释。实际接收人/Agent 尚未登记；本记录是可执行交接包，不把后续工作记为已经完成。

1. 固定 runtime/Store 全量只读投影、双 namespace 无副作用构造/读取、精确 published closure、genesis/安装绑定与版本化元数据；不给缺失证据或任意 caller run_id 授权。
2. 固定 article-owned Anchor 写入/受控协调/可信迁移的局部类型和精确写集；保持 tombstone/HMAC/S16/operation 幂等与独立 CAS 信任边界。无真实 provider 时可设计受控软件适配，不注册为生产 fallback。
3. 显式区分内部阶段许可、公共请求/readiness 和 ordinary worker claim。启动 Reconcile 可在公共屏障关闭时持内部有效准入；受控 Replay 的 Append 仅能精确重读已 committed operation，不能先开公共流量或授权任意新 tombstone。PublicReady 不能作唯一能力。
4. 固定唯一 root 的 recovery 命令/ENV、受控 allowlist、运行独占、全部 worker/request 排空、真实 join 与资源归还；缺少实际部署秘密/权限/主体时记录 unknown，不虚构值或放宽关闭条件。
5. 详细设计稳定后，由 TE 补测试计划、Planner 对账原卡与独立实施责任，再实施/独立评审/验证。不重开 S4.2/S10.5，不让测试卡持有生产写权；本轮未新增 taskID 或依赖边。

上述交接允许进入详细设计；尚未产生新的详细设计、Story、任务规划或实现。实际任务状态仍只取 [Tasks](../../../.add/project/knowledge/tasks/tasks.md)：本次现场快照 59 项，48 completed / 3 active / 8 todo；article S1.5/S4.4/S8.2 和 Case18 未因架构接收关闭。

最终对象哈希、检查、不可变证据索引及本接收记录的哈希见 [Lead receipt](evidence-architecture-r32-r01/lead-reception/decision.json)。
