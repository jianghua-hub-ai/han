# PGPERF r04 evidence summary

Run: `r02-20261104T043052-194b5408`  
Started: 2026-11-04T04:30:54Z  
Generated: 2026-11-04T04:33:59Z

All figures are observational on a shared development machine. They are not target-environment SLO evidence and do not imply capacity approval.

> Capacity limits: small <= 1.0 GB / medium <= 5.5 GB / large <= 8.5 GB / total <= 15 GB. 2026-11-04 本轮用户允许采用建议值，Lead START 明确采用；原 r04 small=0.5/total=14 未修改，新值待本轮独立 Review。单位为 GiB（2^30 bytes）。

| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |
|---|---|---|---:|---:|---:|---:|---:|---:|---|

## Seeds

| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |
|---|---|---|---:|---:|---:|---:|---:|
| han_pgperf_r04_sr2n04_491ae4adc7b34312 | small | r2n04 | 300 | 0 | 15 | 43 | 17 |
| han_pgperf_r04_sr2x08_491ae4adc7b34312 | small | r2x08 | 300 | 0 | 15 | 45 | 18 |
| han_pgperf_r04_sr2plan_491ae4adc7b34312 | small | r2plan | 300 | 0 | 15 | 45 | 18 |
| han_pgperf_r04_sr2n3a_491ae4adc7b34312 | small | r2n3a | 300 | 0 | 15 | 161 | 34 |
| han_pgperf_r04_sr2n3b_491ae4adc7b34312 | small | r2n3b | 300 | 0 | 15 | 908 | 158 |

## Schemas

- `han_pgperf_r04_sr2n04_491ae4adc7b34312` tier=small case=r2n04 retained=true
- `han_pgperf_r04_sr2x08_491ae4adc7b34312` tier=small case=r2x08 retained=true
- `han_pgperf_r04_sr2plan_491ae4adc7b34312` tier=small case=r2plan retained=true
- `han_pgperf_r04_sr2n3a_491ae4adc7b34312` tier=small case=r2n3a retained=true
- `han_pgperf_r04_sr2n3b_491ae4adc7b34312` tier=small case=r2n3b retained=true
