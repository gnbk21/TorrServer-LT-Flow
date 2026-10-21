# Verified subpieces: feasibility and decision

22 October 2026. Research outcome for the October upgrade; no new block-serving
mode is enabled by this work.

## Existing behavior

Flow's pinned libtorrent 2.1.2 supports v1, v2 and hybrid torrents. Its custom
storage supplies actual short-tail geometry, native hashing callbacks and
clear-piece invalidation. `Piece.complete` is set only after native verification
and the cache checks that the actual bytes are present. Existing protocol tests
exercise original-byte transfers separately over TCP and µTP.

Responsive streaming can serve arrived blocks before whole-piece verification;
the separate verified/mirror gate requires completion. These are different
guarantees. The upgraded API/UI reports readable and verified contiguous bytes
separately at the same demand offset. A hash-failure alert is evidence of an
integrity failure, not proof of a malicious peer or bad hardware.

## What a finer verified mode would require

[BEP 52](https://www.bittorrent.org/beps/bep_0052.html) authenticates file content
through SHA-256 Merkle roots over 16 KiB leaves. A future independently verified
subpiece mode would need all of the following:

1. An authenticated proof path from each leaf to the known file root, not merely
   a peer's claimed hash or an arrival/finished-block flag.
2. A native verification result exposed together with storage generation,
   actual file/tail geometry and piece/block identity.
3. A cache bitmap distinct from written-block availability and whole-piece
   completion, invalidated atomically on clear, eviction, reuse or refetch.
4. Reader and contiguous-prefix checks under the same copy/availability locks;
   no gap crossing, stale verification or padding treated as media bytes.
5. Proof/CPU/memory/request bounds, corruption and cancellation tests, and
   compatibility checks across native platforms and the existing patches.
6. Matched high-bitrate comparisons demonstrating a useful benefit before
   enabling the new mode. A native public availability bit currently denotes
   a piece, not a ready-made independently verified per-block contract for Go.

V1's piece SHA-1 does not supply such per-block proofs. Hybrid content must
actually establish its v2 verification path. Neither a similar title nor a
matching size authorizes combining different releases.

## Decision

Retain the existing modes and native verification contract. Do not relabel
written blocks as independently verified, weaken the mirror verification gate,
or add a second media cache. The October work improves integrity evidence,
clear/refetch regression coverage and consistent prefix measurements. A new
verified-subpiece bridge remains a separate experiment requiring the contract
and evidence above; it is not a promised improvement for existing v1 swarms.
