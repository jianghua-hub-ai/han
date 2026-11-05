# PGPERF r04 evidence summary

Run: `r04e-20261103T161337`  
Started: 2026-11-03T16:13:40Z  
Generated: 2026-11-03T16:26:10Z

All figures are observational on a shared development machine. They are not target-environment SLO evidence and do not imply capacity approval.

> Capacity limits: small <= 1.0 GB / medium <= 5.5 GB / large <= 8.5 GB / total <= 15 GB. 2026-11-04 Lead 授权调整，原因=测量窗口写入流量未计入 r04 §8 静态估算；r04 原文未修改。

| Point | Tier | Mode | Concurrency | N | P50 ms | P95 ms | QPS | Errors | Exploratory |
|---|---|---|---:|---:|---:|---:|---:|---:|---|
| W01-typical | small | TIMED-3REP | 1 | 76428 | 1.117 | 1.978 | 849.175 | 0 | false |
| W01-near-limit | small | TIMED-3REP | 1 | 49741 | 1.643 | 2.777 | 552.614 | 0 | false |
| R01-limit20 | small | TIMED-3REP | 1 | 23032 | 3.806 | 6.089 | 255.899 | 0 | false |
| R04-10rows | small | TIMED-3REP | 1 | 231601 | 0.520 | 0.727 | 2573.260 | 0 | false |
| R04-empty | small | SINGLE-RUN | 1 | 336921 | 0.000 | 0.549 | 11230.477 | 0 | true |
| R04-all-hidden | small | SINGLE-RUN | 1 | 3127 | 9.115 | 13.511 | 104.209 | 0 | true |
| R04-last-page | small | TIMED-3REP | 1 | 173892 | 0.549 | 0.988 | 1932.116 | 0 | false |
| D04-100 | small | SINGLE-RUN | 1 | 100 | 1.174 | 1.911 | 836.607 | 0 | true |

## Seeds

| Schema | Tier | Case | Articles | Hidden | Near-limit | Seed ms | ANALYZE ms |
|---|---|---|---:|---:|---:|---:|---:|
| han_pgperf_r04_sw01a_34c16d68b5170de3 | small | w01a | 1000 | 325 | 10 | 190 | 53 |
| han_pgperf_r04_sw01b_34c16d68b5170de3 | small | w01b | 1000 | 325 | 10 | 158 | 41 |
| han_pgperf_r04_sr01a_34c16d68b5170de3 | small | r01a | 1000 | 0 | 10 | 225 | 63 |
| han_pgperf_r04_sr04a_34c16d68b5170de3 | small | r04a | 10 | 0 | 1 | 19 | 11 |
| han_pgperf_r04_sr04b_34c16d68b5170de3 | small | r04b | 0 | 0 | 0 | 0 | 0 |
| han_pgperf_r04_sr04c_34c16d68b5170de3 | small | r04c | 1000 | 1000 | 10 | 126 | 45 |
| han_pgperf_r04_sr04d_34c16d68b5170de3 | small | r04d | 1000 | 0 | 10 | 122 | 36 |
| han_pgperf_r04_sx01a_34c16d68b5170de3 | small | x01a | 0 | 0 | 0 | 0 | 0 |
| han_pgperf_r04_sx01b_34c16d68b5170de3 | small | x01b | 10 | 0 | 1 | 21 | 12 |
| han_pgperf_r04_sd04s_34c16d68b5170de3 | small | d04s | 200 | 0 | 10 | 22 | 14 |

## Schemas

- `han_pgperf_r04_sw01a_34c16d68b5170de3` tier=small case=w01a retained=true
- `han_pgperf_r04_sw01b_34c16d68b5170de3` tier=small case=w01b retained=true
- `han_pgperf_r04_sr01a_34c16d68b5170de3` tier=small case=r01a retained=true
- `han_pgperf_r04_sr04a_34c16d68b5170de3` tier=small case=r04a retained=true
- `han_pgperf_r04_sr04b_34c16d68b5170de3` tier=small case=r04b retained=true
- `han_pgperf_r04_sr04c_34c16d68b5170de3` tier=small case=r04c retained=true
- `han_pgperf_r04_sr04d_34c16d68b5170de3` tier=small case=r04d retained=true
- `han_pgperf_r04_sx01a_34c16d68b5170de3` tier=small case=x01a retained=true
- `han_pgperf_r04_sx01b_34c16d68b5170de3` tier=small case=x01b retained=true
- `han_pgperf_r04_sd04s_34c16d68b5170de3` tier=small case=d04s retained=true
