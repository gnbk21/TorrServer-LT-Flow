// libtorrent custom disk_interface implementation.
//
// Backs every piece read/write/hash with a Go callback. New torrents get
// a shim-minted storage_id; Go uses that id to associate operations with
// the right per-torrent Cache. The translation between libtorrent's
// `peer_request` (piece + offset + length) and the Go-side Piece layout
// is a 1:1 pass-through.
//
// For Etap 4.1 the storage is purely a delegation layer: libtorrent
// calls us, we ferry bytes between its disk thread pool and the Go
// Cache. The Go side may keep the bytes in RAM (MemPiece) or on disk
// (DiskPiece, Etap 4.2) — that decision belongs in Go, not here.

#include "lt_disk_io.h"
#include "lt_io_queue.hpp"
#include <libtorrent/disk_observer.hpp>
#include "lt_shim.h"

#include <libtorrent/disk_buffer_holder.hpp>
#include <libtorrent/disk_interface.hpp>
#include <libtorrent/file_storage.hpp>
#include <libtorrent/hasher.hpp>
#include <libtorrent/io_context.hpp>
#include <libtorrent/peer_request.hpp>
#include <libtorrent/session_params.hpp>
#include <libtorrent/storage_defs.hpp>
#include <libtorrent/units.hpp>
#include <libtorrent/version.hpp>

#include <atomic>
#include <condition_variable>
#include <cstdlib>
#include <cstring>
#include <deque>
#include <functional>
#include <memory>
#include <mutex>
#include <shared_mutex>
#include <thread>
#include <unordered_map>
#include <vector>

namespace lt = libtorrent;

// ============================================================================
// global Go-callback registry
// ============================================================================
namespace {

std::mutex                g_cb_mu;
tsl_storage_callbacks     g_cb{};
bool                      g_cb_set = false;
std::mutex g_io_mu;
std::vector<flow_io_queue*> g_io_queues;
std::array<std::atomic<uint64_t>, 32> g_wait_hist{}, g_callback_hist{};
std::atomic<uint64_t> g_completed{0}, g_wait_max{0}, g_callback_max{0};

void record_latency(uint64_t value, std::array<std::atomic<uint64_t>, 32>& hist,
                    std::atomic<uint64_t>& maximum) {
    auto old = maximum.load();
    while (value > old && !maximum.compare_exchange_weak(old, value)) {}
    unsigned bucket = 0;
    auto remaining = value;
    while (remaining > 1 && bucket < 31) { remaining = (remaining+1)/2; ++bucket; }
    ++hist[bucket];
}
void record_io_latency(uint64_t wait, uint64_t execution) {
    record_latency(wait, g_wait_hist, g_wait_max);
    record_latency(execution, g_callback_hist, g_callback_max);
    ++g_completed;
}
uint64_t percentile(std::array<std::atomic<uint64_t>, 32> const& hist) {
    uint64_t total = 0, seen = 0;
    for (auto const& bucket : hist) total += bucket.load();
    if (!total) return 0;
    auto const target = total-total/20;
    for (unsigned i = 0; i < hist.size(); ++i) {
        seen += hist[i].load();
        if (seen >= target) return uint64_t(1) << i;
    }
    return uint64_t(1) << 31;
}

bool callbacks_complete(tsl_storage_callbacks const& c) {
    return c.open && c.close && c.deleted && c.read && c.write && c.have;
}

} // namespace

extern "C" void lt_storage_io_stats(tsl_io_stats* out) {
    if (!out) return;
    *out = {};
    std::lock_guard<std::mutex> lock(g_io_mu);
    for (auto* queue : g_io_queues) {
        auto const s = queue->status();
        out->queued_bytes += s.bytes;
        out->peak_queue_bytes += s.peak;
        out->queued_jobs += s.jobs;
        out->rejected_jobs += s.rejected;
    }
    out->completed_jobs = g_completed.load();
    out->wait_p95_us = percentile(g_wait_hist);
    out->wait_max_us = g_wait_max.load();
    out->callback_p95_us = percentile(g_callback_hist);
    out->callback_max_us = g_callback_max.load();
}

extern "C" int lt_install_storage_callbacks_full(tsl_storage_callbacks const* cb) {
    std::lock_guard<std::mutex> lk(g_cb_mu);
    if (!cb) {
        g_cb = {};
        g_cb_set = false;
        return LT_OK;
    }
    if (!callbacks_complete(*cb)) return LT_ERR_INVALID;
    g_cb = *cb;
    g_cb_set = true;
    return LT_OK;
}

bool tsl_has_storage_callbacks() {
    std::lock_guard<std::mutex> lk(g_cb_mu);
    return g_cb_set;
}

namespace {
// Snapshot of the registered callbacks. Holds a local copy at session
// creation time so a later lt_install_storage_callbacks_full(nullptr) on
// another thread can't pull the rug out.
tsl_storage_callbacks current_callbacks() {
    std::lock_guard<std::mutex> lk(g_cb_mu);
    return g_cb;
}
} // namespace

// ============================================================================
// helpers
// ============================================================================
namespace {

inline lt::storage_error make_io_error(char const* op) {
    lt::storage_error se;
    se.ec = lt::error_code(boost::system::errc::io_error, lt::system_category());
    se.operation = lt::operation_t::partfile_read;
    (void)op;
    return se;
}

inline std::string sha1_raw_from(lt::sha1_hash const& h) {
    return std::string(reinterpret_cast<char const*>(h.data()), 20);
}

// libtorrent's storage_index_t is aux::strong_typedef wrapping either
// `int` (Ubuntu's 2.0.10 packaged headers) or `unsigned int` (upstream
// RC_2_0 git). Use the public underlying_index_t meta-function so the
// shim compiles against both.
inline int64_t storage_id_of(lt::storage_index_t s) {
    using U = typename lt::aux::underlying_index_t<lt::storage_index_t>::type;
    return static_cast<int64_t>(static_cast<U>(s));
}

// libtorrent 2.1 turned status_t into a flags type: "no error" is the empty
// flag set and the bits live in lt::disk_status::*. 2.0 spelled the same
// thing status_t::no_error. One name for both so the handlers read clean.
#if LIBTORRENT_VERSION_NUM >= 20100
constexpr lt::status_t tsl_status_ok{};
#else
constexpr lt::status_t tsl_status_ok = lt::status_t::no_error;
#endif

} // namespace

// ============================================================================
// the custom disk_interface
// ============================================================================
extern "C" int lt_storage_prune_partial(int64_t storage_id, int piece) {
    auto cb = current_callbacks();
    return cb.prune ? cb.prune(storage_id, piece) : 0;
}

extern "C" int lt_storage_evict_complete(int64_t storage_id, int piece) {
    auto cb = current_callbacks();
    return cb.evict ? cb.evict(storage_id, piece) : 0;
}

namespace {

struct storage_state {
    std::shared_ptr<lt::file_storage const> files;
    int          num_pieces   = 0;
    int          piece_length = 0;
    lt::sha1_hash info_hash;
    int64_t id = 0;
    tsl_storage_callbacks callbacks{};
    ~storage_state() { if (callbacks.close) callbacks.close(id); }
};

class tsl_disk_io final
    : public lt::disk_interface
    , public lt::buffer_allocator_interface
{
public:
    tsl_disk_io(lt::io_context& io,
                lt::settings_interface const& sets,
                lt::counters& cnt,
                tsl_storage_callbacks const& cb)
        : io_(io), sets_(sets), cnt_(cnt), cb_(cb)
    {
        std::lock_guard<std::mutex> lock(g_io_mu);
        g_io_queues.push_back(&work_);
    }

    ~tsl_disk_io() override {
        work_.stop();
        std::lock_guard<std::mutex> lock(g_io_mu);
        g_io_queues.erase(std::remove(g_io_queues.begin(), g_io_queues.end(), &work_), g_io_queues.end());
    }

    // ----- storage lifecycle -----

    lt::storage_holder new_torrent(lt::storage_params const& p,
                                   std::shared_ptr<void> const&) override
    {
        int64_t idx = next_id_++;

        auto ss = std::make_shared<storage_state>();
        ss->id = idx;
        ss->callbacks = cb_;
        ss->files = std::make_shared<lt::file_storage>(p.files);
        ss->num_pieces = ss->files->num_pieces();
        ss->piece_length = ss->files->piece_length();
        ss->info_hash = p.info_hash;

        {
            std::unique_lock<std::shared_mutex> lk(map_mu_);
            storages_.emplace(idx, ss);
        }

        auto raw = sha1_raw_from(p.info_hash);
        cb_.open(idx, reinterpret_cast<uint8_t const*>(raw.data()),
                 ss->num_pieces, ss->piece_length);
        if (cb_.size) cb_.size(idx, p.files.total_size());

        auto storage_idx = lt::storage_index_t(static_cast<int>(idx));
        return lt::storage_holder(storage_idx, *this);
    }

    void remove_torrent(lt::storage_index_t s) override {
        int64_t idx = storage_id_of(s);
        {
            std::unique_lock<std::shared_mutex> lk(map_mu_);
            storages_.erase(idx);
        }
        // Queued jobs retain the storage until their callbacks have settled.
    }

    void abort(bool /*wait*/) override { work_.stop(); }

    // ----- I/O -----

    void async_read(lt::storage_index_t s,
                    lt::peer_request const& r,
                    std::function<void(lt::disk_buffer_holder, lt::storage_error const&)> handler,
                    lt::disk_job_flags_t /*flags*/ = {}) override
    {
        auto state = storage(s);
        auto h = std::make_shared<decltype(handler)>(std::move(handler));
        auto fail = [this, h] {
            lt::post(io_, [h] { (*h)(lt::disk_buffer_holder{}, make_io_error("read")); });
        };
        bool throttle = false;
        if (!state || r.length <= 0 || !work_.submit(storage_id_of(s), int(r.piece),
                std::size_t(r.length), [this, state, r, h] {
            char* buf = static_cast<char*>(std::malloc(std::size_t(r.length)));
            lt::storage_error err;
            if (!buf || cb_.read(state->id, int(r.piece), r.start,
                    reinterpret_cast<uint8_t*>(buf), r.length) != r.length)
                err = make_io_error("read");
#if LIBTORRENT_VERSION_NUM >= 20100
            lt::disk_buffer_holder holder(*this, buf);
#else
            lt::disk_buffer_holder holder(*this, buf, r.length);
#endif
            lt::post(io_, [h, state, holder = std::move(holder), err]() mutable {
                (*h)(std::move(holder), err);
            });
        }, throttle)) fail();
    }

    bool async_write(lt::storage_index_t s,
                     lt::peer_request const& r, char const* buf,
                     std::shared_ptr<lt::disk_observer> observer,
                     std::function<void(lt::storage_error const&)> handler,
                     lt::disk_job_flags_t /*flags*/ = {}) override
    {
        auto state = storage(s);
        auto h = std::make_shared<decltype(handler)>(std::move(handler));
        bool throttle = false;
        // libtorrent owns buf only until this call returns. The queue budget
        // includes this copy; there is no second piece cache or write staging.
        if (!state || r.length <= 0 || std::size_t(r.length) > flow_io_queue::limit_bytes) {
            lt::post(io_, [h] { (*h)(make_io_error("write")); });
            return false;
        }
        std::shared_ptr<std::vector<char>> data;
        try { data = std::make_shared<std::vector<char>>(buf, buf+r.length); }
        catch (...) {
            lt::post(io_, [h] { (*h)(make_io_error("write:allocate")); });
            return false;
        }
        auto wake = [this, observer] {
            if (observer) lt::post(io_, [observer] { observer->on_disk(); });
        };
        if (!work_.submit(state->id, int(r.piece), data->size(), [this, state, r, data, h] {
            lt::storage_error err;
            if (cb_.write(state->id, int(r.piece), r.start,
                    reinterpret_cast<uint8_t const*>(data->data()), r.length) != r.length)
                err = make_io_error("write");
            lt::post(io_, [h, state, err] { (*h)(err); });
        }, throttle, std::move(wake))) {
            lt::post(io_, [h] { (*h)(make_io_error("write:queue-full")); });
            return false;
        }
        return throttle;
    }

    void async_hash(lt::storage_index_t s, lt::piece_index_t piece,
                    lt::span<lt::sha256_hash> /*v2*/, lt::disk_job_flags_t /*flags*/,
                    std::function<void(lt::piece_index_t, lt::sha1_hash const&, lt::storage_error const&)> handler) override
    {
        auto state = storage(s);
        auto h = std::make_shared<decltype(handler)>(std::move(handler));
        bool throttle = false;
        // Stream SHA-1 through a fixed buffer instead of allocating an entire
        // potentially very large piece on each of four workers.
        if (!state || !work_.submit(storage_id_of(s), int(piece), 64*1024,
                [this, state, piece, h] {
            lt::storage_error err;
            lt::sha1_hash digest;
            try {
                int const length = int(state->files->piece_size(piece));
                std::array<char, 64*1024> data;
                lt::hasher hash;
                for (int offset = 0; offset < length; ) {
                    int const count = std::min(int(data.size()), length-offset);
                    if (cb_.read(state->id, int(piece), offset,
                            reinterpret_cast<uint8_t*>(data.data()), count) != count) {
                        err = make_io_error("hash:short-read"); break;
                    }
                    hash.update(lt::span<char const>(data.data(), count));
                    offset += count;
                }
                if (!err.ec) digest = hash.final();
            } catch (...) { err = make_io_error("hash"); }
            lt::post(io_, [h, state, piece, digest, err] { (*h)(piece, digest, err); });
        }, throttle)) lt::post(io_, [h, piece] { (*h)(piece, lt::sha1_hash{}, make_io_error("hash:queue-full")); });
    }

    void async_hash2(lt::storage_index_t,
                     lt::piece_index_t piece,
                     int /*offset*/,
                     lt::disk_job_flags_t,
                     std::function<void(lt::piece_index_t, lt::sha256_hash const&, lt::storage_error const&)> handler) override
    {
        lt::storage_error se;
        se.ec = lt::error_code(boost::system::errc::function_not_supported, lt::system_category());
        lt::post(io_, [h = std::move(handler), piece, se]() mutable {
            h(piece, lt::sha256_hash{}, se);
        });
    }

    // ----- async maintenance ----- (handlers posted to io_, see async_read)

    void async_move_storage(lt::storage_index_t,
                            std::string p,
                            lt::move_flags_t,
                            std::function<void(lt::status_t, std::string const&, lt::storage_error const&)> handler) override
    {
        lt::post(io_, [h = std::move(handler), p = std::move(p)]() mutable {
            h(tsl_status_ok, p, lt::storage_error{});
        });
    }

    void async_release_files(lt::storage_index_t, std::function<void()> handler) override {
        work_.fence([this, h = std::move(handler)]() mutable { lt::post(io_, std::move(h)); });
    }

    void async_delete_files(lt::storage_index_t s,
                            lt::remove_flags_t,
                            std::function<void(lt::storage_error const&)> handler) override
    {
        auto state = storage(s);
        work_.fence([this, state, h = std::move(handler)]() mutable {
            if (state && cb_.deleted) cb_.deleted(state->id);
            lt::post(io_, [state, h = std::move(h)]() mutable { h(lt::storage_error{}); });
        });
    }

    void async_check_files(lt::storage_index_t /*s*/,
                           lt::add_torrent_params const* /*resume*/,
                           lt::aux::vector<std::string, lt::file_index_t> /*links*/,
                           std::function<void(lt::status_t, lt::storage_error const&)> handler) override
    {
        // 4.1: nothing on disk yet — let libtorrent treat all pieces as missing.
        // 4.2 hooks `have()` callback to populate the bitmap before this step.
        lt::post(io_, [h = std::move(handler)]() mutable {
            h(tsl_status_ok, lt::storage_error{});
        });
    }

    void async_rename_file(lt::storage_index_t,
                           lt::file_index_t idx,
                           std::string name,
                           std::function<void(std::string const&, lt::file_index_t, lt::storage_error const&)> handler) override
    {
        lt::post(io_, [h = std::move(handler), idx, name = std::move(name)]() mutable {
            h(name, idx, lt::storage_error{});
        });
    }

    void async_stop_torrent(lt::storage_index_t, std::function<void()> handler) override {
        work_.fence([this, h = std::move(handler)]() mutable { lt::post(io_, std::move(h)); });
    }

    void async_set_file_priority(lt::storage_index_t,
                                 lt::aux::vector<lt::download_priority_t, lt::file_index_t> prio,
                                 std::function<void(lt::storage_error const&, lt::aux::vector<lt::download_priority_t, lt::file_index_t>)> handler) override
    {
        lt::post(io_, [h = std::move(handler), prio = std::move(prio)]() mutable {
            h(lt::storage_error{}, std::move(prio));
        });
    }

    void async_clear_piece(lt::storage_index_t s,
                           lt::piece_index_t idx,
                           std::function<void(lt::piece_index_t)> handler) override
    {
        auto state = storage(s);
        auto h = std::make_shared<decltype(handler)>(std::move(handler));
        bool throttle = false;
        auto clear = [this, state, idx, h] {
            if (state && cb_.clear_piece) cb_.clear_piece(state->id, int(idx));
            lt::post(io_, [state, idx, h] { (*h)(idx); });
        };
        // A full byte queue still accepts bounded maintenance. If its job cap
        // is reached, a global fence provides the same ordering guarantee.
        if (!work_.submit(storage_id_of(s), int(idx), 0, clear, throttle))
            work_.fence(std::move(clear));
    }

    // ----- accounting / control -----

    void update_stats_counters(lt::counters&) const override {}
    std::vector<lt::open_file_state> get_status(lt::storage_index_t) const override { return {}; }
    void submit_jobs() override {}
    void settings_updated() override {}

    // ----- buffer_allocator_interface -----

    void free_disk_buffer(char* buf) override { std::free(buf); }

#if LIBTORRENT_VERSION_NUM >= 20100
    // New pure virtual in 2.1: batched frees from the peer connection loop.
    void free_multiple_buffers(lt::span<char*> bufs) override {
        for (char* b : bufs) std::free(b);
    }
#endif

private:
    std::shared_ptr<storage_state> storage(lt::storage_index_t s) {
        std::shared_lock<std::shared_mutex> lock(map_mu_);
        auto it = storages_.find(storage_id_of(s));
        return it == storages_.end() ? nullptr : it->second;
    }

    lt::io_context&               io_;
    lt::settings_interface const& sets_;
    lt::counters&                 cnt_;
    tsl_storage_callbacks         cb_;

    std::shared_mutex                              map_mu_;
    std::unordered_map<int64_t, std::shared_ptr<storage_state>>     storages_;
    std::atomic<int64_t>                           next_id_{1};

    flow_io_queue work_{record_io_latency};
};

// Factory used by session_params.disk_io_constructor. Captures the
// callback snapshot taken at session_new time via a function-local
// static — see tsl_install_disk_io_on().
tsl_storage_callbacks g_session_cb{};

std::unique_ptr<lt::disk_interface> tsl_make_disk_io(
    lt::io_context& io, lt::settings_interface const& s, lt::counters& cnt)
{
    return std::make_unique<tsl_disk_io>(io, s, cnt, g_session_cb);
}

} // namespace

void tsl_install_disk_io_on(lt::session_params& params) {
    if (!tsl_has_storage_callbacks()) return;
    g_session_cb = current_callbacks();
    params.disk_io_constructor = &tsl_make_disk_io;
}
