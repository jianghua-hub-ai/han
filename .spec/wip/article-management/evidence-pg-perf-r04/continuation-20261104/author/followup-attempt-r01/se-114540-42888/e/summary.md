# PGPERF r04 evidence summary

Run: `fu_sem_114540_42888`  
Started: 2026-11-04T11:45:43Z  
Generated: 2026-11-04T11:53:40Z

All figures are observational on a shared development machine. They are not target-environment SLO evidence and do not imply capacity approval.

> Capacity limits: small <= 1.0 GB / medium <= 5.5 GB / large <= 8.5 GB / total <= 15 GB. 2026-11-04 本轮用户允许采用建议值，Lead START 明确采用；原 r04 small=0.5/total=14 未修改，新值待本轮独立 Review。单位为 GiB（2^30 bytes）。

| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |
|---|---|---|---:|---:|---:|---:|---:|---:|---|

## Seeds

| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |
|---|---|---|---:|---:|---:|---:|---:|
| han_pgperf_r04_sr2t1_a2b11a24332b5e62 | small | r2t1 | 300 | 0 | 15 | 22460 | 19 |
| han_pgperf_r04_sr2e02_a2b11a24332b5e62 | small | r2e02 | 300 | 0 | 15 | 23612 | 21 |
| han_pgperf_r04_sr2d02_a2b11a24332b5e62 | small | r2d02 | 300 | 0 | 15 | 24577 | 18 |
| han_pgperf_r04_sr2r05_a2b11a24332b5e62 | small | r2r05 | 300 | 0 | 15 | 24511 | 20 |
| han_pgperf_r04_sr2n04_a2b11a24332b5e62 | small | r2n04 | 300 | 0 | 15 | 26229 | 15 |
| han_pgperf_r04_sr2x08_a2b11a24332b5e62 | small | r2x08 | 300 | 0 | 15 | 25659 | 14 |
| han_pgperf_r04_sr2plan_a2b11a24332b5e62 | small | r2plan | 300 | 0 | 15 | 25824 | 16 |
| han_pgperf_r04_sr2n3a_a2b11a24332b5e62 | small | r2n3a | 300 | 0 | 15 | 32667 | 33 |

## Schemas

- `han_pgperf_r04_sr2t1_a2b11a24332b5e62` tier=small case=r2t1 retained=true
- `han_pgperf_r04_sr2e02_a2b11a24332b5e62` tier=small case=r2e02 retained=true
- `han_pgperf_r04_sr2d02_a2b11a24332b5e62` tier=small case=r2d02 retained=true
- `han_pgperf_r04_sr2r05_a2b11a24332b5e62` tier=small case=r2r05 retained=true
- `han_pgperf_r04_sr2n04_a2b11a24332b5e62` tier=small case=r2n04 retained=true
- `han_pgperf_r04_sr2x08_a2b11a24332b5e62` tier=small case=r2x08 retained=true
- `han_pgperf_r04_sr2plan_a2b11a24332b5e62` tier=small case=r2plan retained=true
- `han_pgperf_r04_sr2n3a_a2b11a24332b5e62` tier=small case=r2n3a retained=true
- `han_pgperf_r04_sr2n3b_a2b11a24332b5e62` tier=small case=r2n3b retained=true
