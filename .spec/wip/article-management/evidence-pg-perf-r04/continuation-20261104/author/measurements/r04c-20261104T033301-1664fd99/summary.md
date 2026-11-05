# PGPERF r04 evidence summary

Run: `r04c-20261104T033301-1664fd99`  
Started: 2026-11-04T03:33:03Z  
Generated: 2026-11-04T03:42:28Z

All figures are observational on a shared development machine. They are not target-environment SLO evidence and do not imply capacity approval.

> Capacity limits: small <= 1.0 GB / medium <= 5.5 GB / large <= 8.5 GB / total <= 15 GB. 2026-11-04 本轮用户允许采用建议值，Lead START 明确采用；原 r04 small=0.5/total=14 未修改，新值待本轮独立 Review。单位为 GiB（2^30 bytes）。

| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |
|---|---|---|---:|---:|---:|---:|---:|---:|---|
| W01-typical | small | TIMED-3REP | 1 | 94519 | 1.034 | 1.743 | 1050.183 | 0 | false |
| W01-near-limit | small | TIMED-3REP | 1 | 48537 | 1.693 | 3.060 | 539.272 | 0 | false |
| R01-limit20 | small | TIMED-3REP | 1 | 24026 | 3.729 | 5.911 | 266.939 | 0 | false |
| R04-10rows | small | TIMED-3REP | 1 | 122672 | 0.590 | 1.286 | 1363.013 | 0 | false |

## Seeds

| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |
|---|---|---|---:|---:|---:|---:|---:|
| han_pgperf_r04_sw01a_86ac35efa75a35fe | small | w01a | 1000 | 325 | 10 | 132 | 40 |
| han_pgperf_r04_sw01b_86ac35efa75a35fe | small | w01b | 1000 | 325 | 10 | 159 | 55 |
| han_pgperf_r04_sr01a_86ac35efa75a35fe | small | r01a | 1000 | 0 | 10 | 129 | 46 |
| han_pgperf_r04_sr04a_86ac35efa75a35fe | small | r04a | 10 | 0 | 1 | 15 | 12 |

## Schemas

- `han_pgperf_r04_sw01a_86ac35efa75a35fe` tier=small case=w01a retained=true
- `han_pgperf_r04_sw01b_86ac35efa75a35fe` tier=small case=w01b retained=true
- `han_pgperf_r04_sr01a_86ac35efa75a35fe` tier=small case=r01a retained=true
- `han_pgperf_r04_sr04a_86ac35efa75a35fe` tier=small case=r04a retained=true
