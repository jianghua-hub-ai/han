# PGPERF r04 evidence summary

Run: `r02-20261104T042816-2f7e888f`  
Started: 2026-11-04T04:28:19Z  
Generated: 2026-11-04T04:28:59Z

All figures are observational on a shared development machine. They are not target-environment SLO evidence and do not imply capacity approval.

> Capacity limits: small <= 1.0 GB / medium <= 5.5 GB / large <= 8.5 GB / total <= 15 GB. 2026-11-04 本轮用户允许采用建议值，Lead START 明确采用；原 r04 small=0.5/total=14 未修改，新值待本轮独立 Review。单位为 GiB（2^30 bytes）。

| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |
|---|---|---|---:|---:|---:|---:|---:|---:|---|

## Seeds

| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |
|---|---|---|---:|---:|---:|---:|---:|
| han_pgperf_r04_sr2t1_9ad25377c3b60e75 | small | r2t1 | 300 | 0 | 15 | 43 | 20 |
| han_pgperf_r04_sr2e02_9ad25377c3b60e75 | small | r2e02 | 300 | 0 | 15 | 46 | 25 |
| han_pgperf_r04_sr2d02_9ad25377c3b60e75 | small | r2d02 | 300 | 0 | 15 | 46 | 19 |
| han_pgperf_r04_sr2r05_9ad25377c3b60e75 | small | r2r05 | 300 | 0 | 15 | 40 | 18 |
| han_pgperf_r04_sr2n04_9ad25377c3b60e75 | small | r2n04 | 300 | 0 | 15 | 47 | 21 |

## Schemas

- `han_pgperf_r04_sr2t1_9ad25377c3b60e75` tier=small case=r2t1 retained=true
- `han_pgperf_r04_sr2e02_9ad25377c3b60e75` tier=small case=r2e02 retained=true
- `han_pgperf_r04_sr2d02_9ad25377c3b60e75` tier=small case=r2d02 retained=true
- `han_pgperf_r04_sr2r05_9ad25377c3b60e75` tier=small case=r2r05 retained=true
- `han_pgperf_r04_sr2n04_9ad25377c3b60e75` tier=small case=r2n04 retained=true
