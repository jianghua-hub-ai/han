# R09 leak执行证据独立结论

**局部Gate：PASS。S24-CODE-R01-F005：CLOSED（原最低合同的泄漏有界收敛缺陷）。** 没有新增阻断finding；离线原件核证111项均成立。实际Go35312自然 **exit1 / FAIL** 保留，未称Go测试GREEN。

固定对象`author/followup-attempt-r01/le-113346-18724/`；R09 H C3DFDC66…D89B8B99、nonrace binary71B3EF47…705521D5、policy7C94C0DA…A33E04、grant C8A20432…EE2FEC实际复算匹配。五snapshot/live摘要及launch前后同版。

真实beginOwned→SELECT成功后的遗留marker→cleanup前ownedTx=1 guard已到达。冻结源码在该guard后必经实际Rollback及连接Release，再核ownedTx为空/acquired0；stdout没有reclaim错误，pre-close返回成功，业务/disk/probe Close都完成。共享原5s截止内实际关闭16.0514ms，worker joined、goroutine2→2、残余backend0；12个owned客户端自然wait齐全，外部精确app查询为空。

物证没有单独逐tx Rollback结果JSON或tx_after数值；本次关闭依据是固定必经控制流、真实guard、成功waitReturned谓词、实际pool Close与外部空集合的联合证据，不伪造缺失字段。此前已接收的取消worker子条件保留，本次补齐原F005欠缺的真实泄漏负例。

旧84个schema全保留，仅新增并登记`sr2leak_336a400e1247dcd8`。静止after SQL 2,105,743,039B；相对本次准确before增1,490,944B；仍按原origin1,406,932,671B累计698,810,368B，small余374,931,456B。先前+24KiB预检拒绝保持FAIL/无Go/无新schema；新外层只把历史snapshot与实时准入分开，不重置基线或放松live cap/volume/inventory硬门。本次实际snapshot drift=0，不冒称执行验证了非零漂移分支。

累计原18项：**7 CLOSED / 11 PENDING**。CLOSED：F001/F003/F004/**F005**/F008/F010/F018；PENDING：F002/F006/F007/F009/F011/F012/F013/F014/F015/F016/F017。其他finding状态未动，E001保持CLOSED。

完整calibration/small/medium本attempt永久NRU，旧完整性能及large/canonical NRU保持。N9尚未授权、未接收；本结论不批准后续运行、不改任务、不关闭S2.4。唯一Tnew及145/150停新/硬停不动。Reviewer未运行Go/PG/Git或作者helper；最终actual STOP和报告哈希见index.json。
