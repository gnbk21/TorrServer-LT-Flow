# Five-repeat streaming comparisons

Checkpoint: `ff5df2fa`; [CI run 38069527761](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/38069527761).

These are original-byte, controlled Linux native peer/HTTP tests, not Android decoder playback. All 160 trials completed byte-integrity checks. Each row contains five rotated runs. Values below are medians; five trials are too few for precise population tail estimates. Server read blocking is not automatically a player-visible stutter.

Stable baseline: `MatriX.145.Flow-v1.0.1`, source `55c16dc399b8f9b14bd5ae76b62450af027c6b81`; published checksums and provenance were verified before running.

Stable executable SHA-256: `8ff5d9f4263da477b3aba4d78ffdae4409d750208792c7ca819c936ea19c195c`.
Candidate executable SHA-256: `b7211a2e717bbd8189c0c3232a83c88a0bca38cad6321267d1f99deabf258f0b`.

Matched settings: 120 Mb/s generated media, 60 seconds, RAM cache, 512 MiB cache, 32 MiB startup reserve, 4 MiB pieces. Healthy, outage, burst and mixed-peer scenarios each use the same bytes across compared variants. Profile and transport cohorts ran separately; CPU timings across cohorts must not be compared.

## Profiles

| Scenario/profile | Passed | Startup ms | Blocking over 250 ms, total ms | Seek P99 ms | Peak RSS MiB | CPU seconds | Native duplicate MiB |
|---|---:|---:|---:|---:|---:|---:|---:|
| bursts/previous-legacy | 5/5 | 5804.17 | 0.00 | 718.92 | 1159.01 | 12.10 | 0.02 |
| bursts/previous-adaptive | 5/5 | 5804.31 | 0.00 | 649.62 | 1159.70 | 12.33 | 0.00 |
| bursts/legacy | 5/5 | 5803.41 | 0.00 | 1086.22 | 1119.93 | 12.75 | 0.02 |
| bursts/adaptive | 5/5 | 5804.73 | 0.00 | 47.20 | 1117.30 | 12.98 | 0.00 |
| healthy/previous-legacy | 5/5 | 5803.92 | 0.00 | 502.52 | 1115.90 | 9.97 | 0.00 |
| healthy/previous-adaptive | 5/5 | 5803.00 | 0.00 | 1082.09 | 1117.12 | 9.79 | 0.00 |
| healthy/legacy | 5/5 | 5803.11 | 0.00 | 869.23 | 1117.53 | 10.32 | 0.00 |
| healthy/adaptive | 5/5 | 6003.44 | 0.00 | 387.97 | 1116.61 | 10.42 | 0.00 |
| mixed-peers/previous-legacy | 5/5 | 5805.27 | 0.00 | 286.07 | 1120.66 | 11.02 | 2.50 |
| mixed-peers/previous-adaptive | 5/5 | 5804.85 | 0.00 | 220.48 | 1121.21 | 11.02 | 7.09 |
| mixed-peers/legacy | 5/5 | 6004.27 | 0.00 | 351.86 | 1124.07 | 11.79 | 16.08 |
| mixed-peers/adaptive | 5/5 | 6002.29 | 0.00 | 911.65 | 1116.77 | 11.88 | 2.06 |
| outages/previous-legacy | 5/5 | 5405.91 | 14524.30 | 783.84 | 1118.12 | 23.38 | 0.00 |
| outages/previous-adaptive | 5/5 | 5407.36 | 16521.01 | 697.39 | 1119.05 | 23.43 | 0.02 |
| outages/legacy | 5/5 | 5405.77 | 18732.62 | 1081.41 | 1118.17 | 24.72 | 0.02 |
| outages/adaptive | 5/5 | 5406.64 | 15353.05 | 1081.37 | 1116.82 | 24.88 | 0.00 |

## Transport buffers

Native direct is 0 KiB; 64, 256 and 1024 KiB use the existing buffered HTTP reader. These runs all use the candidate Adaptive profile.

| Scenario/buffer KiB | Passed | Startup ms | Blocking over 250 ms, total ms | Seek P99 ms | Peak RSS MiB | CPU seconds | Native duplicate MiB |
|---|---:|---:|---:|---:|---:|---:|---:|
| bursts/adaptive-transport-0 | 5/5 | 5405.01 | 0.00 | 47.09 | 1116.67 | 26.17 | 0.00 |
| bursts/adaptive-transport-64 | 5/5 | 5405.36 | 0.00 | 656.40 | 1115.80 | 25.88 | 0.00 |
| bursts/adaptive-transport-256 | 5/5 | 5405.86 | 0.00 | 502.21 | 1117.20 | 26.06 | 0.02 |
| bursts/adaptive-transport-1024 | 5/5 | 5403.37 | 0.00 | 908.73 | 1117.41 | 26.14 | 0.02 |
| healthy/adaptive-transport-0 | 5/5 | 5604.11 | 0.00 | 325.39 | 1115.93 | 13.92 | 0.02 |
| healthy/adaptive-transport-64 | 5/5 | 5404.75 | 0.00 | 596.33 | 1116.14 | 13.87 | 0.02 |
| healthy/adaptive-transport-256 | 5/5 | 5404.00 | 0.00 | 46.00 | 1114.66 | 13.75 | 0.00 |
| healthy/adaptive-transport-1024 | 5/5 | 5403.43 | 0.00 | 561.35 | 1116.90 | 13.86 | 0.00 |
| mixed-peers/adaptive-transport-0 | 5/5 | 5605.80 | 0.00 | 638.31 | 1116.41 | 24.92 | 12.73 |
| mixed-peers/adaptive-transport-64 | 5/5 | 5606.43 | 0.00 | 419.27 | 1117.10 | 24.36 | 2.11 |
| mixed-peers/adaptive-transport-256 | 5/5 | 5606.10 | 0.00 | 465.02 | 1121.17 | 24.58 | 7.27 |
| mixed-peers/adaptive-transport-1024 | 5/5 | 5605.73 | 0.00 | 397.08 | 1117.94 | 24.54 | 5.39 |
| outages/adaptive-transport-0 | 5/5 | 6006.67 | 17260.76 | 1077.93 | 1114.72 | 18.50 | 0.00 |
| outages/adaptive-transport-64 | 5/5 | 6005.90 | 16690.78 | 559.93 | 1110.28 | 18.74 | 0.00 |
| outages/adaptive-transport-256 | 5/5 | 6005.34 | 12976.86 | 1071.50 | 1114.46 | 18.83 | 0.00 |
| outages/adaptive-transport-1024 | 5/5 | 6005.28 | 15903.31 | 840.63 | 1116.60 | 18.54 | 0.00 |

## Decisions and limits

- Healthy, burst and mixed-peer trials had no server reads blocked for more than 250 ms. Outage trials retained substantial blocking in every profile; passing byte integrity does not mean uninterrupted playback.
- Candidate Adaptive outage blocking had a 15.35 s median versus 16.52 s for stable Adaptive. Stable Legacy was 14.52 s; candidate Legacy was 18.73 s. The data does not establish that the candidate is universally faster.
- The 256 KiB transport had the lowest median outage blocking (12.98 s versus 15.90 s at 1024 KiB). Other scenarios and five-run variability do not support an automatic default change. The 1024 KiB default and Legacy default remain unchanged.
- Profile cohort candidate CPU medians increased about 5-8% in several cases. Representative profiling and final-head comparisons remain necessary before making a performance claim.
- These reports predate the subsequent v2/session/expiry corrections. Final acceptance must repeat the required comparisons for the release candidate. Eight-hour endurance, physical Android telemetry and Windows recovery checks are separate gates.

## Follow-up checkpoint: 0b9635b3

[CI run 38077497553](https://github.com/gnbk21/TorrServer-LT-Flow/actions/runs/38077497553) passed another 160/160 original-byte trials. The eight downloaded reports each contain 20 trials, five rotations, the same fixture SHA-256, 512 MiB cache and 32 MiB startup reserve. This is an additional checkpoint, not final-head release acceptance.

Candidate executable SHA-256: `839aa81d61bfd9b0484ad7201a59cf6665282e60608c792e0275fd8bfd3150bb`. Stable executable SHA-256 remains `8ff5d9f4263da477b3aba4d78ffdae4409d750208792c7ca819c936ea19c195c`.


### Profiles

| Scenario/variant | Passed | Startup ms | Blocking over 250 ms, total ms | Seek P99 ms | Peak RSS MiB | CPU seconds | Native duplicate MiB |
|---|---:|---:|---:|---:|---:|---:|---:|
| bursts/previous-legacy | 5/5 | 5406.17 | 0.00 | 1089.52 | 1160.33 | 23.78 | 0.00 |
| bursts/previous-adaptive | 5/5 | 5407.01 | 0.00 | 1091.15 | 1148.77 | 23.78 | 0.02 |
| bursts/legacy | 5/5 | 5403.57 | 0.00 | 47.11 | 1122.12 | 25.67 | 0.02 |
| bursts/adaptive | 5/5 | 5405.30 | 0.00 | 47.00 | 1117.38 | 25.62 | 0.00 |
| healthy/previous-legacy | 5/5 | 6207.29 | 0.00 | 1081.96 | 1117.84 | 20.03 | 0.02 |
| healthy/previous-adaptive | 5/5 | 6006.11 | 0.00 | 452.29 | 1118.98 | 20.18 | 0.02 |
| healthy/legacy | 5/5 | 6205.65 | 0.00 | 363.33 | 1117.56 | 21.86 | 0.00 |
| healthy/adaptive | 5/5 | 6007.71 | 0.00 | 462.02 | 1117.87 | 21.96 | 0.00 |
| mixed-peers/previous-legacy | 5/5 | 6004.59 | 0.00 | 525.75 | 1118.07 | 11.06 | 1.80 |
| mixed-peers/previous-adaptive | 5/5 | 5805.03 | 0.00 | 48.54 | 1125.37 | 11.04 | 4.70 |
| mixed-peers/legacy | 5/5 | 6004.07 | 0.00 | 381.16 | 1119.63 | 11.62 | 3.38 |
| mixed-peers/adaptive | 5/5 | 5804.19 | 0.00 | 286.26 | 1119.14 | 11.80 | 5.72 |
| outages/previous-legacy | 5/5 | 5606.05 | 15228.78 | 1082.17 | 1108.77 | 15.72 | 0.00 |
| outages/previous-adaptive | 5/5 | 5605.56 | 15754.12 | 1083.04 | 1116.07 | 15.75 | 0.02 |
| outages/legacy | 5/5 | 5804.64 | 15318.59 | 978.28 | 1116.91 | 17.06 | 0.02 |
| outages/adaptive | 5/5 | 5603.25 | 14047.84 | 779.66 | 1116.87 | 16.87 | 0.02 |


### Transport buffers

| Scenario/variant | Passed | Startup ms | Blocking over 250 ms, total ms | Seek P99 ms | Peak RSS MiB | CPU seconds | Native duplicate MiB |
|---|---:|---:|---:|---:|---:|---:|---:|
| bursts/adaptive-transport-0 | 5/5 | 5804.11 | 0.00 | 758.02 | 1116.12 | 12.40 | 0.02 |
| bursts/adaptive-transport-64 | 5/5 | 5805.10 | 0.00 | 46.97 | 1116.34 | 12.63 | 0.00 |
| bursts/adaptive-transport-256 | 5/5 | 5804.61 | 0.00 | 1094.89 | 1115.54 | 12.66 | 0.00 |
| bursts/adaptive-transport-1024 | 5/5 | 5804.16 | 0.00 | 873.75 | 1117.25 | 12.44 | 0.00 |
| healthy/adaptive-transport-0 | 5/5 | 5406.31 | 0.00 | 690.81 | 1116.53 | 14.91 | 0.00 |
| healthy/adaptive-transport-64 | 5/5 | 5405.25 | 0.00 | 915.23 | 1117.85 | 14.91 | 0.00 |
| healthy/adaptive-transport-256 | 5/5 | 5205.59 | 0.00 | 1081.10 | 1117.01 | 15.17 | 0.00 |
| healthy/adaptive-transport-1024 | 5/5 | 5405.59 | 0.00 | 1082.75 | 1120.62 | 14.99 | 0.02 |
| mixed-peers/adaptive-transport-0 | 5/5 | 5404.98 | 0.00 | 264.97 | 1118.04 | 16.70 | 2.42 |
| mixed-peers/adaptive-transport-64 | 5/5 | 5405.44 | 0.00 | 417.63 | 1117.90 | 16.61 | 6.02 |
| mixed-peers/adaptive-transport-256 | 5/5 | 5405.95 | 0.00 | 894.40 | 1117.57 | 16.99 | 16.00 |
| mixed-peers/adaptive-transport-1024 | 5/5 | 5603.55 | 0.00 | 1092.54 | 1116.87 | 16.94 | 4.97 |
| outages/adaptive-transport-0 | 5/5 | 5803.44 | 14170.26 | 879.76 | 1107.37 | 12.44 | 0.00 |
| outages/adaptive-transport-64 | 5/5 | 5803.99 | 13683.26 | 855.69 | 1117.13 | 12.14 | 0.00 |
| outages/adaptive-transport-256 | 5/5 | 5804.56 | 15203.11 | 1049.32 | 1116.98 | 12.08 | 0.00 |
| outages/adaptive-transport-1024 | 5/5 | 5803.64 | 13412.13 | 744.61 | 1116.95 | 12.04 | 0.00 |

### Interpretation

- Healthy, mixed-peer and burst cases again had no reads blocked over 250 ms. Outage cases still had substantial blocking; this is not evidence of uninterrupted phone playback.
- Candidate Adaptive outage blocking was 14.05 s versus 15.75 s for stable Adaptive, about 10.8% lower. Candidate Legacy was 15.32 s versus 15.23 s for stable Legacy. Candidate CPU medians were about 5-10% higher across these profile cases.
- The transport outage medians were 14.17 s (direct), 13.68 s (64 KiB), 15.20 s (256 KiB) and 13.41 s (1024 KiB). The earlier checkpoint favored 256 KiB. This reversal reinforces keeping the 1024 KiB default rather than promoting a small-sample winner.
- Several burst seek medians improved, while mixed-peer Adaptive seek latency and duplicate bytes increased versus stable Adaptive. The data supports scenario-specific investigation, not a universal speed claim.
- Profile and transport cohorts ran independently; compare CPU only within a cohort. Five runs are not a precise population tail estimate. Native CPU attribution and final-source endurance/comparisons remain separate required work.
