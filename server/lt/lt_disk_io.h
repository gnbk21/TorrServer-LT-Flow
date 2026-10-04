// libtorrent custom disk_interface that delegates piece I/O to Go.
//
// Wired into the session at lt_session_new() time iff Go has registered
// non-NULL storage callbacks beforehand via lt_install_storage_callbacks_full().
// Otherwise libtorrent's default disk_io is used.

#ifndef TS_LT_DISK_IO_H
#define TS_LT_DISK_IO_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

// Go-side storage callbacks. Full set must be non-NULL to be considered
// active — partial sets are rejected by lt_install_storage_callbacks_full.
struct tsl_storage_callbacks {
    // Storage lifecycle. storage_id is a shim-minted opaque integer.
    void (*open)   (int64_t storage_id, const uint8_t* info_hash_20, int num_pieces, int64_t piece_length);
    void (*close)  (int64_t storage_id);
    void (*deleted)(int64_t storage_id);

    // Synchronous piece I/O. Return bytes transferred (< 0 = error).
    int  (*read)   (int64_t storage_id, int piece, int64_t offset, uint8_t* buf, int len);
    int  (*write)  (int64_t storage_id, int piece, int64_t offset, const uint8_t* buf, int len);

    // Etap 4.2: report locally-have pieces during resume.
    int  (*have)   (int64_t storage_id, int piece);
    // Optional: remove a stale partial after native write/hash work settles.
    int  (*prune)  (int64_t storage_id, int piece);
    // Optional: remove a complete LRU entry after native I/O settles.
    int  (*evict)  (int64_t storage_id, int piece);
    // Exact metadata size, including the short final piece.
    void (*size)   (int64_t storage_id, int64_t total_size);
    // Native hash-failure fence; keep the backend but clear readable state.
    void (*clear_piece)(int64_t storage_id, int piece);
};

// Install / clear the Go callbacks. Pass NULL to revert to default
// libtorrent disk_io on the next session_new. Returns LT_OK on success.
int lt_install_storage_callbacks_full(const struct tsl_storage_callbacks* cb);
int lt_storage_prune_partial(int64_t storage_id, int piece);
int lt_storage_evict_complete(int64_t storage_id, int piece);

#ifdef __cplusplus
}
#endif

#endif
