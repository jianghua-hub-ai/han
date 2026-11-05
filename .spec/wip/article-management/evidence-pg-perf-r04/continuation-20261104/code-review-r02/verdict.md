# S2.4 原18finding candidate复审 R02 verdict

**Candidate Static Gate: CHANGES REQUIRED。14项有静态整改落点、核销PENDING；4项仍有实质缺口；正式CLOSED 0。**

对象是 `lead/source-candidate-r02/manifest.json` 的Lead点时candidate（2026-11-04T04:29:59.3592855Z），不是author freeze或live最终代码。五snapshot bytes/SHA独立复核一致，R01 base64摘要核对后离线差分留档。

| 文件 | 已审candidate SHA256 |
|---|---|
| harness | `96291E78CBB194B2581A455354746917D8C5B376F40512FDF5D0BC08294CBC9A` |
| write | `7F1AD7DE6A3B5C28B663B960D3A59075C8CFD09839FA591EB518F822925EAD7D` |
| read | `E4A7A9C5A973E023E0B103C35882155578854C5251CB9F8F2CC00A47058B5A40` |
| notice_delete | `A6C749537A5A89CC030EF068B8911EE18852ABAF1B923DDC141B251DD69D6E5E` |
| docs | `D47605C85226A618680A9BA455557236F4D202A7FB3BA9C4FB388D9049650C70` |

仍OPEN/PARTIAL：

- **F002 HIGH**：有phase context，但worker Wait无界、复合helper仍用长h.ctx，不能证明121s/累计1s实际有界退出；精确phase时轴/零成功等证据门仍不足。
- **F008 MEDIUM**：CREATE即内存登记已改，但校准仅第三调用Finalize，初次创建后失败仍可能没有实际inventory。
- **F009 MEDIUM**：初始完整批门已恢复；后续批按[2,1,1]插recipient，违反每notice两种各一的真实约束，完整N03仍受阻。
- **F015 MEDIUM**：新notice分两条autocommit且只有一个recipient；同candidate五hash的directed-r02已23514 FAIL，晚到/同worker断言未跑通。

其余F001/F003/F004/F005/F006/F007/F010/F011/F012/F013/F014/F016/F017/F018的具体修复落点与待核证据逐项见report.md；不能当正式核销。

同版directed-r02整体exit1/44.156s，四子项PASS不等于命令通过；N04后锁链/读计划/1k/10k drain均前错NOT RUN。第一次directed的竞争timeout/连接和goroutine失败保留。较早负例/局部GREEN五hash不等于candidate；后续pg-append GREEN只作Lead提供的新修订线索。

Lead已告知live H/N继续变更，本报告不审新字节、不外推：待作者STOP/newfreeze，只补审这两文件candidate→最终delta及关联证据，另三不变则复用。最终还须作者freeze对应、定向raw/exit/故障资源证据齐全及Lead复核live hash/命令/范围。

**本轮完整新校准/small/medium：NOT RUN（12:32:04.586115条件门已过，后续绿灯也不恢复资格）。Execution Evidence Review：PENDING；S2.4仍active。** 原起点/elapsed/累计磁盘不重置；不拼旧证据成完整性能PASS。Reviewer只做离线取证，未跑Go/PG、读凭据、改源/候选/任务/旧报告，未Git写/清理/下级委派。交回Lead后STOP。
