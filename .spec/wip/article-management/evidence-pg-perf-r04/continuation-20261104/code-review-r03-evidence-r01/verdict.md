# S2.4 R03新增证据 verdict

**PARTIAL / PENDING。没有最终CodeReview、执行Review或性能PASS。**

源码仍为正式source-freeze-r02五hash；独立核snapshot/live以及8个新完成leaf的before/after，均一致。只追加证据复核，未重新源码返工，未Go/PG/凭据/Git/清理/委派。

- **新增CLOSED：F001、F003、F004、F018。** F008/F010保持CLOSED。原18项累计 **6 CLOSED / 12 PARTIAL或PENDING**。
- **R03-E001 CLOSED**：补正仅将旧W02 r2body归到正确run；旧diagnostics29+1=30，与旧resource/preclose/raw/probe/顶层清单逐项相符，旧文件未改。
- core构建0/6.719s，race3运行0/10.781s，57个子项pass；main源码构建的同路径binary在r03的sandbox CWD运行，并非新worktree。binary SHA `5B0EACA4CF170198E69F705C4E09E0AA2C80A59BB41466B9ABE9D4779054ACF8` 已独立核；运行后摘要与命令链的证据边界见report，不伪造当时binary摘要。
- measurement/seed/volume分别exit1。measurement还伴随acquire0/sql5000ms磁盘SQL超时，但已核到更早的完整10s warmup、measurement两条raw（1成功+1非context错误）、统一NONCONFORMING errors=1；F001按真实分支证据核销，绝非仅凭exit1。seed/volume命中各自预期故障及资源/inventory。**leak仅到前置disk SQL5000ms超时，注入未到达**，F005不能核销。
- 已全文读作者最终 `final-review-index.md` 污染勘误，并独立对照raw/时间轴：**F001仅核销统一测量错误拒绝功能；整次measurement混合失败不是干净负例PASS**。不以作者expected_outcome标记或exit1代替判断；累计6/12不变。
- 后追加 **pg-positive-safety-r01 exit0/27.188s**，取消release/join、W02Bodies真实存储双正文与c4短诊断通过；两个schema acquired0/backend0/goroutine回基线。**pg-positive-semantics-r01 exit1/13.235s**，freshT1首DDL前acquire0/sql5003ms超时，9个业务分支均未到达（首项预检失败，其余8项前错NOT RUN）。不把Go fail事件当业务均已执行。
- F002正式120s active/121s wall及累计tail实际整点未发生；F006/F007/F009等正式窗口条件保持PENDING。其余未运行组不以旧hash或Main普通检查补成通过。
- Main七check确为exit0/五hash无漂移；仅项目检查。旧13:29 STOP/probe保留为第一组历史，**最终positive-stop为13:35:47.482879+08**、SOURCE仍NO WRITE；latest positive probe覆盖六个r03 run，backend0、**4新schema、原baseline累计586055680 bytes**，五hash一致。保存probe/进程空hits与作者STOP不冒充Reviewer实时运行。

完整calibration/small/medium/large/canonical本轮 **NOT RUN（NRU）**，任务active，原窗口与资源基线不重置。保持STOP，不再为补证修改Go或盲目续跑；当前缺项与最小补证方向见 [report.md](report.md)。最终接收由Lead核实际树/命令/范围/所有权。

本轮完成，**STOP**。
