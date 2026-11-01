# MatriX.148 review

Reviewed 1 November 2026 before further release qualification. The official
[release](https://github.com/YouROK/TorrServer/releases/tag/MatriX.148)
was published on 31 October. Its
[complete comparison](https://github.com/YouROK/TorrServer/compare/MatriX.147.2...MatriX.148)
contains two commits and four changed files, including two test files:
`8a69eadd28e6d243cea62ab6e37dbf55eb4eb77a` →
`3395046476632f68fb4cc85fd10d42cfede40ec0`.

| Change | Flow assessment and adaptation |
|---|---|
| [#946](https://github.com/YouROK/TorrServer/pull/946), `b41c19d113a50fa23e436de71c39a41523f4d112`: canonical IPv6 referer matching | Applicable security correctness fix. Different spellings of the same address previously evaded a configured IP referer deny rule. Flow now uses `net.IP.String()` in its shared rule/header normalizer, retaining stricter IDNA lookup, hostname validation and the independent management-origin checks. Equivalent IPv4-mapped addresses also deduplicate. Other addresses remain unaffected. This does not turn referer filtering into authentication. |
| [#947](https://github.com/YouROK/TorrServer/pull/947), `3395046476632f68fb4cc85fd10d42cfede40ec0`: binary size labels | Applicable presentation/API correction. Torznab sizes used a stray `C` (`1.5 GCiB`); now they use `KiB`, `MiB`, `GiB`, etc. Actual byte counts and media are unchanged. Flow retains its context cancellation, bounded response, source outcomes and selected-interface transport. |

No engine, piece scheduler, buffering, Go/native dependency, web application or
quality-control change appears in this comparison. Therefore this release gives
no evidence of increased torrent bandwidth or shorter buffering. Both small fixes
are included in Flow 1.2.0 without merging unrelated upstream architecture or
dropping compatibility with older `MatriX.145.Flow-*` identities. The owner
requested the new prefix `MatriX.148.Flow-v1.2.0`. It identifies the reviewed
maintenance baseline; Flow keeps its own feature set and semantic version.
Release classification, provenance selection and new installer/updater helpers
accept both reviewed prefixes. Older installed helpers only understand `145`,
so crossing to `148` requires installing the new package/helper once; settings
and torrent identities do not need migration.

## Verification scope

Focused local regressions reproduced both defects before the production fixes:
IPv6/IPv4-mapped rule equivalence failed, and real loopback Torznab results
contained `KCiB`, `MCiB`, `GCiB` and `TCiB`. Private before/after logs are retained
under `.tools/upstream148`. The corrected suites must pass together with Flow's
existing WAF and Torznab tests; the final source still requires full native,
security, platform and tagged distribution gates. These checks use generated
metadata and do not certify phone playback or public-swarm throughput.
