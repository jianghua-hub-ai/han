# S414 R03-F001 Temp-only preparation — STOP

原作者继续有界实施方法；17项同版完整方法/规则核SHA后复用，刷新R03 finding/report/receipt/精确compound fragment及普通/race实际失败raw，Main reception的当前事实已核。没有新Route、G0或Temp验收门。

仅新建自身OS Temp包，含可应用的两份完整candidate、原两源码before、标准unified diff与后继probe计划。仓内原5前后SHA完全相同（FDC279/57946等）；仓内SOURCE写0、Go0、SQL/PG0、controlled process0、DSN/秘密ENV读取0、Git命令0、网络/清理/委派0。未读其他lane Temp、活动SSOT或design。Dirac窗口不受影响。

候选语义：Run defer直接recover已有主panic，完成原image.closeAndJoin后原值panic；cleanup panic在局部边界捕获为secondary，原值不格式化。secondary通过tool私有mutex存储、泛化安全Error以及可drain的私有takeCleanupFailure保留；error causes可errors.Is，非error panic值也可私有取回。无主panic时cleanup-only panic仍以原值外抛；普通action error+cleanup error仍以原顺序errors.Join，private呈现/unknown/no replay不变。原获取即owned、正常owner+error转交一次和checkedFD defer不改。没有新增公共API/权威、SDK/provider、fallback、协议或第六源码。

新增作者8个planned叶：error/非error不可比较slice主panic × cleanup返回error/panic(error)/panic(slice)共6；另有正常returned error+cleanup error合并cause对照，以及真实child完成后cleanup-only panic。每叶保留Begin/child/Stop/FD/borrowed handle/owner close恰一次断言，compound6还核私有secondary取回、errors.Is/opaque身份、canary不出现在Error文本、二次drain为空。原测试只添加group注册并追加补测，未改旧断言。新测试无新增import或跨package依赖。

三份原Reviewer fragment DBDEDF/24D710/613CD0 原字节保留；其第三份历史STATIC header也未改。Red overlay基于原57946完整test，Green overlay基于最新candidate完整test，均仅test映射，生产overlay0。未来先在未改生产前执行exact compound普通/race真实可编译Red，再应用原两源码并跑保留旧102+Reviewer compound1+作者8（静态预期111，尚未确认）、7PG每mode以及获授检查。所有Go/进程/PG动作等待Main后续grant，不追加确认请求。

纯文本diff roundtrip核对一致、函数静态长度≤50；未运行Go编译、gofmt、vet、测试或Git apply/check。因此候选不是修复PASS，当前MEDIUM不由作者关闭。native Gate、rootS416长期owner、任意提前panic provider的未知join边界、真实restore/E2E与Task/G4保持原限制。实际资源正确性仍须后继执行和独立复审。

STOP：短包交Main调度后继SOURCE/Go/RestorePG窗口，不继续仓内动作。
