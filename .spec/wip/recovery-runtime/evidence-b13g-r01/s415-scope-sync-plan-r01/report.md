# S4.15/B13G 原卡返工范围精准同步

已在固定 main / HEAD `6da8abbceb0fa197364810a92a8fa2d97b38ddca` 按与 Nash 实际 ID 精确匹配的 grant 各一次写入原 Task/Plan。沿原 development-delivery 返工，复用本会话已全文读取且同版本的 Role、batch-plan、delivery、task-execution、multi-agent；不新 Route，不调用 task-decompose 新建/拆分，不增加 Planning Gate。

原卡12文件=原10+runtime.go/runtime_test.go。本轮repair4仅recovery_workers.go/_test.go、runtime.go/_test.go；两项HIGH的真实normal/race失败、Main采用原因及作者actual-release已注记。F001固定首次shutdown context同步只读查询与先bind notice后forward cancel；F002ordinary健康pause在最终授予锁内回原changed等待，保留request拒绝/真实fault/失权/取消错误。详细技术采用来自Main grant；设计补充由Linnaeus另持有，本Planner不代写、不宣称设计完成或PASS。

`beforeafter/validation.json` 保留其他73整卡逐字节SHA、319正式edges有序全等、9批原安排归一后字节相等、7前置和5风险及全部原命令代码块不变。Task仍[-]，60/4/10；B13F和全部旧前置未重开。两文件完整字节inverse和实际edits重放均通过。Tasks UTF8BOM/LF/FF2、Plan UTF8BOM/CRLF保持。before与Main snapshot、grant的SHA完整一致；before/after及精确diff在beforeafter，所有实际变更为可重放字节载荷`edits.json`。

CodeGate/wholeG4仍CHANGES_REQUIRED；无修复/测试PASS，不关闭Task，不替独立Reviewer或人的接纳签字。SOURCE/Go/cache/SQL/PG/service=0，未读写源码、未运行Git/服务或清理/子委派。SOURCE12仅登记所有权边界，不授源码写权。metadata已RELEASED，STOP，交Main现场核diff/status；本返回不代其现场复核。
