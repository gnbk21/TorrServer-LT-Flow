// TorrServer-LT C shim implementation.
// See lt_shim.h for the contract.
//
// libtorrent (arvidn) 2.x API only — v1 SHA-1 torrents.
// Custom disk_io (Etap 4.1) lives in lt_disk_io.cpp and is wired in when
// Go has installed storage callbacks via lt_install_storage_callbacks_full.

#include "lt_shim.h"
#include "lt_disk_io.h"

#include "third_party/nlohmann/json.hpp"

#include <libtorrent/add_torrent_params.hpp>
#include <libtorrent/bdecode.hpp>
#include <libtorrent/alert.hpp>
#include <libtorrent/alert_types.hpp>
#include <libtorrent/error_code.hpp>
#include <libtorrent/extensions/smart_ban.hpp>
#include <libtorrent/extensions/ut_metadata.hpp>
#include <libtorrent/ip_filter.hpp>
#include <libtorrent/magnet_uri.hpp>
#include <libtorrent/session.hpp>
#include <libtorrent/session_params.hpp>
#include <libtorrent/session_stats.hpp>
#include <libtorrent/settings_pack.hpp>
#include <libtorrent/sha1_hash.hpp>
#include <libtorrent/torrent_handle.hpp>
#include <libtorrent/torrent_info.hpp>
#include <libtorrent/torrent_status.hpp>
#include <libtorrent/version.hpp>

// Internal headers for lt_torrent_we_dont_have: poking the piece_picker to
// drop a "have" piece must happen on the session network thread. These symbols
// are only present when linking a static libtorrent built from source (the
// build/*.sh release path, which defines TSL_HAVE_LT_INTERNALS); a shared
// distro/Homebrew libtorrent doesn't export them, so the feature falls back to
// a public-API no-op there. See lt_torrent_we_dont_have below.
#ifdef TSL_HAVE_LT_INTERNALS
#include <libtorrent/download_priority.hpp>
#include <libtorrent/io_context.hpp>
#if LIBTORRENT_VERSION_NUM >= 20100
// 2.1 moved the internal headers under aux_ (classes now in lt::aux::; the
// methods we call are unchanged and native_handle() already returns the
// aux type, so only the include paths differ).
#include <libtorrent/aux_/piece_picker.hpp>
#include <libtorrent/aux_/peer_list.hpp>
#include <libtorrent/aux_/peer_connection.hpp>
#include <libtorrent/aux_/torrent_peer.hpp>
#include <libtorrent/aux_/torrent.hpp>
#else
#include <libtorrent/piece_picker.hpp>
#include <libtorrent/peer_list.hpp>
#include <libtorrent/peer_connection.hpp>
#include <libtorrent/torrent.hpp>
#endif
#include <libtorrent/aux_/session_interface.hpp>
#endif

// Implemented in lt_disk_io.cpp; declared here after all libtorrent
// headers have been seen so the session_params type is unambiguous.
extern void tsl_install_disk_io_on(libtorrent::session_params& params);

#include <atomic>
#include <chrono>
#include <cstdlib>
#include <cstring>
#include <fstream>
#include <future>
#include <memory>
#include <mutex>
#include <shared_mutex>
#include <sstream>
#include <string>
#include <stdexcept>
#include <unordered_map>
#include <utility>
#include <vector>

namespace lt = libtorrent;

#ifdef TSL_HAVE_LT_INTERNALS
// A cache miss is not a hash failure: reconcile availability without blaming
// peers or rechecking the entire torrent. The pinned build declares this
// non-virtual member in torrent.hpp; it changes no class layout.
#if LIBTORRENT_VERSION_NUM >= 20100
void libtorrent::aux::torrent::flow_forget_piece(piece_index_t const index,
    download_priority_t const priority)
#else
void libtorrent::torrent::flow_forget_piece(piece_index_t const index,
    download_priority_t const priority)
#endif
{
    if (m_abort || !valid_metadata() || index < piece_index_t{0}
        || index >= m_torrent_file->end_piece()) return;
    bool const was_finished = is_finished();
    bool const was_all = m_have_all;
    leave_seed_mode(seed_mode_t::skip_checking);
    m_have_all = false;
    if (!has_picker()) {
        need_picker();
        if (was_all) m_picker->we_have_all();
    }
    bool const had_piece = m_picker->have_piece(index);
    m_picker->set_piece_priority(index, dont_download);
    m_picker->we_dont_have(index);
    m_picker->set_piece_priority(index, priority);
    // File progress must forget the byte count too, or the next completion
    // counts the same bytes twice. Rebuild only for formerly complete pieces.
    if (had_piece || was_all) {
        m_file_progress.clear();
#if LIBTORRENT_VERSION_NUM >= 20100
        m_file_progress.init(*m_picker, m_torrent_file->layout());
#else
        m_file_progress.init(*m_picker, m_torrent_file->files());
#endif
    }
    if (had_piece) inc_stats_counter(counters::num_have_pieces, -1);
    // is_seed() also consults m_state. Leaving that at seeding makes priority
    // updates no-ops and disconnects the very seed needed for the cache miss.
    if (state() == torrent_status::seeding)
        set_state(m_picker->is_finished() ? torrent_status::finished : torrent_status::downloading);
    update_gauge();
    update_peer_interest(was_finished);
    flow_refresh_connect_candidates();
    set_need_save_resume(torrent_handle::if_download_progress);
    state_updated();
}

#if LIBTORRENT_VERSION_NUM >= 20100
void libtorrent::aux::torrent::flow_refresh_connect_candidates()
#else
void libtorrent::torrent::flow_refresh_connect_candidates()
#endif
{
    if (m_abort) return;
    // Ordinary priority/deadline changes can also transition finished to
    // downloading after an eviction left the missing piece at priority zero.
    // Refresh after every demand change, not only when clearing a have bit.
    if (m_peer_list) {
        auto peer_state = get_peer_list_state();
        m_peer_list->flow_refresh_connect_candidates(&peer_state);
    }
    update_want_peers();
}
#endif

static void refresh_connect_candidates(lt::torrent_handle const& h) {
#ifdef TSL_HAVE_LT_INTERNALS
    auto tor = h.native_handle();
    if (!tor) return;
    // torrent_handle enqueues its priority operation on this same context.
    // Posting afterwards observes the resulting state without racing it.
    lt::post(tor->session().get_context(), [tor]() {
        // Avoid scanning a large peer list on every window/deadline update.
        // The blocked transition is specifically a downloading torrent with
        // known peers but a stale zero candidate count.
        if (!tor->is_finished() && tor->num_connect_candidates() == 0
            && tor->num_known_peers() > 0)
            tor->flow_refresh_connect_candidates();
    });
#else
    (void)h;
#endif
}
using json = nlohmann::json;

// nlohmann::json::dump() defaults to error_handler_t::strict, which THROWS
// type_error.316 ("invalid UTF-8 byte ...") when any string in the object is
// not valid UTF-8. libtorrent error/alert strings come from the OS: on a
// non-UTF-8 system locale (e.g. Russian Windows, where error_code::message()
// returns CP1251 bytes) those strings are not valid UTF-8, so a single such
// alert would abort the whole batched dump and the Go side would lose every
// alert in the pop (stalling piece_finished delivery and breaking streaming).
// Replace bad bytes with U+FFFD instead of throwing — diagnostic text may be
// mojibake, but the pump never fails. Use this for every dump in the shim.
inline std::string json_dump(json const& j) {
    return j.dump(-1, ' ', false, json::error_handler_t::replace);
}

// ============================================================================
// global last error
//
// The shim previously stored this state thread-locally, which is incorrect
// when the caller is Go: a goroutine can migrate between OS threads
// between two cgo calls, so the lt_last_error() call would read state on
// a different thread than the one that produced it.
//
// We instead keep one process-wide pair under a mutex. lt_last_error()
// returns a pointer into a thread-local snapshot string, so the returned
// pointer remains valid until the next lt_last_error() call from the same
// thread (which is the contract documented in lt_shim.h).
// ============================================================================
namespace {

std::mutex     g_err_mu;
std::string    g_last_error;
int            g_last_error_code = LT_OK;

inline int set_err(int code, std::string msg) {
    std::lock_guard<std::mutex> lk(g_err_mu);
    g_last_error = std::move(msg);
    g_last_error_code = code;
    return code;
}

inline int set_err_ec(int code, lt::error_code const& ec) {
    std::lock_guard<std::mutex> lk(g_err_mu);
    g_last_error = ec.message();
    g_last_error_code = code;
    return code;
}

inline char* alloc_string(std::string const& s, size_t* out_len) {
    if (out_len) *out_len = s.size();
    char* p = static_cast<char*>(std::malloc(s.size() + 1));
    if (!p) {
        set_err(LT_ERR_INTERNAL, "out of memory");
        return nullptr;
    }
    std::memcpy(p, s.data(), s.size());
    p[s.size()] = '\0';
    return p;
}

inline size_t copy_string(std::string const& s, char* buf, size_t cap) {
    if (buf && cap > s.size()) {
        std::memcpy(buf, s.data(), s.size());
        buf[s.size()] = '\0';
    }
    return s.size();
}

// 20-byte sha1 → 40-char lowercase hex
inline std::string sha1_hex(lt::sha1_hash const& h) {
    static const char* hex = "0123456789abcdef";
    std::string out(40, '0');
    auto const* p = reinterpret_cast<unsigned char const*>(h.data());
    for (int i = 0; i < 20; ++i) {
        out[i * 2]     = hex[p[i] >> 4];
        out[i * 2 + 1] = hex[p[i] & 0x0f];
    }
    return out;
}

// 40-char hex → sha1_hash; returns false if input is not exactly 40 hex chars.
inline bool hex_to_sha1(std::string const& hex, lt::sha1_hash& out) {
    if (hex.size() != 40) return false;
    unsigned char buf[20] = {0};
    auto nyb = [](char c) -> int {
        if (c >= '0' && c <= '9') return c - '0';
        if (c >= 'a' && c <= 'f') return 10 + c - 'a';
        if (c >= 'A' && c <= 'F') return 10 + c - 'A';
        return -1;
    };
    for (int i = 0; i < 20; ++i) {
        int hi = nyb(hex[i * 2]);
        int lo = nyb(hex[i * 2 + 1]);
        if (hi < 0 || lo < 0) return false;
        buf[i] = static_cast<unsigned char>((hi << 4) | lo);
    }
    out = lt::sha1_hash(reinterpret_cast<char const*>(buf));
    return true;
}

inline std::vector<std::string> split_csv(std::string const& csv) {
    std::vector<std::string> out;
    std::string cur;
    for (char c : csv) {
        if (c == ',') {
            if (!cur.empty()) out.push_back(std::move(cur));
            cur.clear();
        } else if (c != '\r' && c != '\n') {
            cur.push_back(c);
        }
    }
    if (!cur.empty()) out.push_back(std::move(cur));
    return out;
}

} // namespace

// ============================================================================
// session and torrent registries
// ============================================================================
namespace {

struct session_slot {
    std::unique_ptr<lt::session> s;
    std::mutex pump_mu; // serializes wait_alert+pop_alerts on this session
    std::vector<lt::udp::endpoint> restored_dht_nodes; // <=32 per address family; pump_mu
};

std::shared_mutex g_sess_mu;
std::unordered_map<int64_t, std::shared_ptr<session_slot>> g_sessions;
std::atomic<int64_t> g_next_sess{1};

std::shared_mutex g_torr_mu;
// The hex key is captured at registration time so unregistering never
// depends on the handle still being valid: remove_torrent is async, and a
// handle can expire between remove and unregister. Erasing by the stored key
// keeps g_hash2id from retaining a stale hash -> dead-id mapping, which would
// make every future add of the same info-hash return the dead id ("torrent
// not found" forever).
struct sparse_snapshot {
    std::mutex mu;
    bool pending = false;
    std::chrono::steady_clock::time_point last_request{};
    std::string cached = "{\"known\":false}";
    std::string resume_peers = "[]"; // private opt-in state, never a status field
    std::chrono::steady_clock::time_point peers_at{};
    std::int64_t peers_at_ms = 0;
};
struct torrent_entry {
    lt::torrent_handle h;
    std::string hex; // 40-char lowercase v1 info-hash
    std::shared_ptr<sparse_snapshot> sparse = std::make_shared<sparse_snapshot>();
};
std::unordered_map<int64_t, torrent_entry> g_torrents;
std::unordered_map<std::string, int64_t> g_hash2id;
std::atomic<int64_t> g_next_torr{1};

std::shared_ptr<session_slot> get_session(lt_session id) {
    std::shared_lock<std::shared_mutex> lk(g_sess_mu);
    auto it = g_sessions.find(id);
    if (it == g_sessions.end()) return nullptr;
    return it->second;
}

lt::torrent_handle get_torrent(lt_torrent id) {
    std::shared_lock<std::shared_mutex> lk(g_torr_mu);
    auto it = g_torrents.find(id);
    if (it == g_torrents.end()) return lt::torrent_handle();
    return it->second.h;
}

int64_t register_torrent(lt::torrent_handle const& h) {
    if (!h.is_valid()) return 0;
    auto hashes = h.info_hashes();
    std::string hex = sha1_hex(hashes.v1);
    std::unique_lock<std::shared_mutex> lk(g_torr_mu);
    auto exist = g_hash2id.find(hex);
    if (exist != g_hash2id.end()) {
        // Same info-hash re-added. If the old entry's handle is dead (torrent
        // removed, then re-added), replace it with the fresh handle under the
        // same id; returning the dead id would break the new torrent.
        auto told = g_torrents.find(exist->second);
        if (told == g_torrents.end()) {
            g_torrents.emplace(exist->second, torrent_entry{h, hex});
        } else if (!told->second.h.is_valid()) {
            told->second.h = h;
            told->second.sparse = std::make_shared<sparse_snapshot>();
        }
        return exist->second;
    }
    int64_t id = g_next_torr++;
    g_torrents.emplace(id, torrent_entry{h, hex});
    g_hash2id.emplace(std::move(hex), id);
    return id;
}

int64_t lookup_torrent_id(lt::torrent_handle const& h) {
    if (!h.is_valid()) return 0;
    auto hashes = h.info_hashes();
    std::string hex = sha1_hex(hashes.v1);
    std::shared_lock<std::shared_mutex> lk(g_torr_mu);
    auto it = g_hash2id.find(hex);
    if (it == g_hash2id.end()) return 0;
    return it->second;
}

void unregister_torrent(lt_torrent id) {
    std::unique_lock<std::shared_mutex> lk(g_torr_mu);
    auto it = g_torrents.find(id);
    if (it == g_torrents.end()) return;
    g_hash2id.erase(it->second.hex);
    g_torrents.erase(it);
}

} // namespace

// ============================================================================
// settings JSON ↔ settings_pack
// ============================================================================
namespace {

void json_into_settings(json const& j, lt::settings_pack& sp, std::string* warnings) {
    if (!j.is_object()) return;
    for (auto it = j.begin(); it != j.end(); ++it) {
        std::string name = it.key();
        int id = lt::setting_by_name(name);
        if (id < 0) {
            if (warnings) {
                if (!warnings->empty()) warnings->push_back(';');
                *warnings += "unknown setting: " + name;
            }
            continue;
        }
        int const type_mask = id & lt::settings_pack::type_mask;
        try {
            if (type_mask == lt::settings_pack::string_type_base) {
                if (it->is_string()) sp.set_str(id, it->get<std::string>());
            } else if (type_mask == lt::settings_pack::int_type_base) {
                if (it->is_number_integer()) sp.set_int(id, it->get<int>());
                else if (it->is_number_float()) sp.set_int(id, static_cast<int>(it->get<double>()));
            } else if (type_mask == lt::settings_pack::bool_type_base) {
                if (it->is_boolean()) sp.set_bool(id, it->get<bool>());
                else if (it->is_number_integer()) sp.set_bool(id, it->get<int>() != 0);
            }
        } catch (std::exception const& e) {
            if (warnings) {
                if (!warnings->empty()) warnings->push_back(';');
                *warnings += std::string("bad value for ") + name + ": " + e.what();
            }
        }
    }
}

} // namespace

// ============================================================================
// ip_filter (P2P-format text → lt::ip_filter)
// ============================================================================
namespace {

lt::ip_filter parse_p2p_filter(std::string const& text) {
    lt::ip_filter f;
    std::istringstream in(text);
    std::string line;
    while (std::getline(in, line)) {
        if (!line.empty() && line.back() == '\r') line.pop_back();
        std::string s;
        for (char c : line) {
            if (c == '\t' || c == ' ') continue;
            s.push_back(c);
        }
        if (s.empty() || s[0] == '#') continue;
        auto colon = s.find_last_of(':');
        if (colon == std::string::npos) continue;
        std::string rng = s.substr(colon + 1);
        auto dash = rng.find('-');
        lt::error_code ec1, ec2;
        lt::address first, last;
        if (dash == std::string::npos) {
            first = lt::make_address(rng, ec1);
            last = first;
        } else {
            first = lt::make_address(rng.substr(0, dash), ec1);
            last  = lt::make_address(rng.substr(dash + 1), ec2);
        }
        if (ec1 || ec2) continue;
        if (first.is_v4() != last.is_v4()) continue;
        f.add_rule(first, last, lt::ip_filter::blocked);
    }
    return f;
}

} // namespace

// ============================================================================
// status → JSON
// ============================================================================
namespace {

std::string state_name(lt::torrent_status::state_t st) {
    switch (st) {
        case lt::torrent_status::checking_files:    return "checking_files";
        case lt::torrent_status::downloading_metadata: return "downloading_metadata";
        case lt::torrent_status::downloading:       return "downloading";
        case lt::torrent_status::finished:          return "finished";
        case lt::torrent_status::seeding:           return "seeding";
        case lt::torrent_status::checking_resume_data: return "checking_resume_data";
        default: return "unknown";
    }
}

json status_to_json(lt::torrent_handle const& h) {
    json j;
    if (!h.is_valid()) return j;
    auto st = h.status();
    j["name"]         = st.name;
    auto hashes = h.info_hashes();
    j["info_hash"]    = sha1_hex(hashes.v1);
    j["state"]        = state_name(st.state);
    j["is_finished"]  = st.is_finished;
    j["progress"]     = st.progress;
    j["total_done"]   = st.total_done;
    j["total_wanted"] = st.total_wanted;
    j["download_rate"]   = st.download_rate;
    j["upload_rate"]     = st.upload_rate;
    j["num_peers"]       = st.num_peers;
    j["num_seeds"]       = st.num_seeds;
    j["list_peers"]      = st.list_peers;
    j["list_seeds"]      = st.list_seeds;
    j["connect_candidates"]    = st.connect_candidates;
    j["total_payload_download"] = st.total_payload_download;
    j["total_payload_upload"]   = st.total_payload_upload;
    j["total_download"]         = st.total_download;
    j["total_upload"]           = st.total_upload;
    j["num_pieces"]             = st.num_pieces;
    auto ti = h.torrent_file();
    if (ti) {
        j["piece_length"] = ti->piece_length();
        j["total_size"]   = ti->total_size();
        j["has_metadata"] = true;
        j["private"] = ti->priv();
    } else {
        j["piece_length"] = 0;
        j["total_size"]   = 0;
        j["has_metadata"] = false;
    }
    return j;
}

} // namespace

// ============================================================================
// alert → JSON
// ============================================================================
namespace {

json alert_to_json(lt::alert const* a) {
    json j;
    j["type"]     = a->what();
    j["category"] = static_cast<uint64_t>(static_cast<std::uint32_t>(a->category()));
    j["message"]  = a->message();
    // A signed mirror URL can include secrets in its path or query. Native
    // alerts must never hand that URL to console/debug logging or history.
    if (auto const* seed = lt::alert_cast<lt::url_seed_alert>(a)) {
        j["message"] = "web seed unavailable";
        j["error_code"] = seed->error.value();
    }

    // torrent_alert is an abstract base — its `alert_type` constant is
    // deprecated and may be absent (libtorrent built with
    // -Ddeprecated-functions=OFF). Walk the hierarchy via RTTI instead.
    if (auto const* ta = dynamic_cast<lt::torrent_alert const*>(a)) {
        int64_t id = lookup_torrent_id(ta->handle);
        if (id != 0) j["torrent"] = id;
        if (ta->handle.is_valid()) {
            j["torrent_hash"] = sha1_hex(ta->handle.info_hashes().v1);
        }
    }

    if (auto const* fa = lt::alert_cast<lt::listen_succeeded_alert>(a)) {
        j["port"] = fa->port;
        j["transport"] = fa->socket_type == lt::socket_type_t::udp ? "udp" : "tcp";
        j["ipv6"] = fa->address.is_v6();
    } else if (auto const* fa = lt::alert_cast<lt::listen_failed_alert>(a)) {
        j["error_code"] = fa->error.value();
    } else if (auto const* fa = lt::alert_cast<lt::portmap_alert>(a)) {
        j["port"] = fa->external_port;
        j["transport"] = fa->map_protocol == lt::portmap_protocol::tcp ? "tcp" : "udp";
    } else if (auto const* fa = lt::alert_cast<lt::portmap_error_alert>(a)) {
        j["error_code"] = fa->error.value();
    } else if (auto const* fa = lt::alert_cast<lt::incoming_connection_alert>(a)) {
        j["ipv6"] = fa->endpoint.address().is_v6();
        j["transport"] = fa->socket_type == lt::socket_type_t::utp ? "utp" : "tcp";
    } else if (auto const* fa = lt::alert_cast<lt::piece_finished_alert>(a)) {
        j["piece"] = static_cast<int>(fa->piece_index);
    } else if (auto const* fa = lt::alert_cast<lt::block_finished_alert>(a)) {
        j["piece"] = static_cast<int>(fa->piece_index);
        j["block"] = fa->block_index;
    } else if (auto const* fa = lt::alert_cast<lt::file_completed_alert>(a)) {
        j["file"] = static_cast<int>(fa->index);
    } else if (auto const* fa = lt::alert_cast<lt::hash_failed_alert>(a)) {
        j["piece"] = static_cast<int>(fa->piece_index);
    } else if (auto const* fa = lt::alert_cast<lt::peer_disconnected_alert>(a)) {
        j["error_code"] = fa->error.value();
        j["operation"] = static_cast<int>(fa->op);
        j["disconnect_reason"] = fa->error == lt::errors::torrent_paused ? "PAUSED"
            : fa->error == lt::errors::upload_upload_connection ? "REDUNDANT" : "OTHER";
    } else if (auto const* fa = lt::alert_cast<lt::dht_reply_alert>(a)) {
        j["peers"] = fa->num_peers;
    } else if (auto const* fa = lt::alert_cast<lt::tracker_reply_alert>(a)) {
        j["url"]   = std::string(fa->tracker_url());
        j["peers"] = fa->num_peers;
    } else if (auto const* fa = lt::alert_cast<lt::tracker_error_alert>(a)) {
        j["url"]   = std::string(fa->tracker_url());
        j["error"] = fa->error.message();
    } else if (auto const* fa = lt::alert_cast<lt::torrent_error_alert>(a)) {
        j["error"] = fa->error.message();
    } else if (auto const* fa = lt::alert_cast<lt::file_error_alert>(a)) {
        j["error"] = fa->error.message();
        j["file"]  = std::string(fa->filename());
    } else if (auto const* fa = lt::alert_cast<lt::session_stats_alert>(a)) {
        // The default message() is a bare line of numbers; map them to their
        // metric names so the Go side gets a usable counters snapshot.
        static auto const metrics = lt::session_stats_metrics();
        auto const counters = fa->counters();
        json c = json::object();
        for (auto const& m : metrics) {
            if (m.value_index >= 0
                && static_cast<std::size_t>(m.value_index) < counters.size())
                c[m.name] = counters[m.value_index];
        }
        j["counters"] = std::move(c);
    }
    return j;
}

} // namespace

// ============================================================================
// public C ABI
// ============================================================================

#define WRAP_BEGIN try {
#define WRAP_END(retcode) } catch (std::exception const& e) { \
    return set_err(LT_ERR_INTERNAL, e.what()); \
} catch (...) { \
    return set_err(LT_ERR_INTERNAL, "unknown C++ exception"); \
}

extern "C" {

// ----- error reporting / memory / version -----

const char* lt_last_error(void) {
    // Copy under lock into a thread-local string so the returned pointer
    // is valid for as long as the contract promises ("until the next
    // lt_* call from this thread") regardless of concurrent set_err()s.
    static thread_local std::string snapshot;
    std::lock_guard<std::mutex> lk(g_err_mu);
    snapshot = g_last_error;
    return snapshot.c_str();
}

int lt_last_error_code(void) {
    std::lock_guard<std::mutex> lk(g_err_mu);
    return g_last_error_code;
}

void lt_free(void* p) {
    std::free(p);
}

size_t lt_shim_version(char* buf, size_t cap) {
    static const std::string ver = "MatriX.LT-001";
    return copy_string(ver, buf, cap);
}

int lt_cache_reconciliation_supported(void) {
#ifdef TSL_HAVE_LT_INTERNALS
    return 1;
#else
    return 0;
#endif
}

size_t lt_engine_version(char* buf, size_t cap) {
    static const std::string ver = LIBTORRENT_VERSION;
    return copy_string(ver, buf, cap);
}

// ----- session lifecycle -----

static lt::session_params decode_dht_state(const char* state, size_t len) {
    if (!state || len == 0 || len > 1024 * 1024)
        throw std::invalid_argument("invalid DHT state size");
    lt::error_code ec;
    auto root = lt::bdecode(lt::span<char const>(state, len), ec, nullptr, 16, 16384);
    if (ec || root.type() != lt::bdecode_node::dict_t || root.data_section().size() != len)
        throw std::invalid_argument("invalid DHT state encoding");
    auto dht = root.dict_find_dict("dht state");
    if (!dht) throw std::invalid_argument("missing DHT state");
    for (auto key : {"nodes", "nodes6", "node-id"}) {
        auto list = dht.dict_find(key);
        if (!list) continue;
        if (list.type() != lt::bdecode_node::list_t || list.list_size() > 4096)
            throw std::invalid_argument("invalid DHT state list");
        for (int i = 0; i < list.list_size(); ++i) {
            auto item = list.list_at(i);
            if (item.type() != lt::bdecode_node::string_t)
                throw std::invalid_argument("invalid DHT state entry");
            int n = item.string_length();
            bool ok = std::strcmp(key, "nodes") == 0 ? n == 6
                : std::strcmp(key, "nodes6") == 0 ? n == 18 : n == 24 || n == 36;
            if (!ok) throw std::invalid_argument("invalid DHT endpoint length");
        }
    }
    return lt::read_session_params(root, lt::session_handle::save_dht_state);
}

static char* encode_dht_state(lt::session_params const& params, size_t* len) {
    auto buf = lt::write_session_params_buf(params, lt::session_handle::save_dht_state);
    if (buf.size() > 1024 * 1024) { set_err(LT_ERR_INVALID, "DHT state too large"); return nullptr; }
    auto* out = static_cast<char*>(std::malloc(buf.size()));
    if (!out) { set_err(LT_ERR_INTERNAL, "DHT allocation failed"); return nullptr; }
    std::memcpy(out, buf.data(), buf.size());
    if (len) *len = buf.size();
    return out;
}

int lt_dht_state_nodes(const char* state, size_t len) {
    try {
        auto params = decode_dht_state(state, len);
        return static_cast<int>(params.dht_state.nodes.size() + params.dht_state.nodes6.size());
    } catch (std::exception const&) { return set_err(LT_ERR_PARSE, "invalid DHT state"); }
}

char* lt_dht_state_normalize(const char* state, size_t len, size_t* out_len) {
    set_err(LT_OK, "");
    try { return encode_dht_state(decode_dht_state(state, len), out_len); }
    catch (std::exception const&) { set_err(LT_ERR_PARSE, "invalid DHT state"); return nullptr; }
}

char* lt_session_dht_state(lt_session id, size_t* len) {
    set_err(LT_OK, "");
    try {
        auto slot = get_session(id);
        if (!slot) { set_err(LT_ERR_NOT_FOUND, "session not found"); return nullptr; }
        auto params = slot->s->session_state(lt::session_handle::save_dht_state);
        // Keep useful native hints for later address/socket changes too. An
        // offline or empty snapshot must not erase the last known nodes.
        if (!params.dht_state.nodes.empty() || !params.dht_state.nodes6.empty()) {
            std::lock_guard<std::mutex> lk(slot->pump_mu);
            slot->restored_dht_nodes.clear();
            for (auto const* nodes : {&params.dht_state.nodes, &params.dht_state.nodes6}) {
                for (size_t i = 0; i < std::min<size_t>(32, nodes->size()); ++i)
                    slot->restored_dht_nodes.push_back((*nodes)[i]);
            }
        }
        return encode_dht_state(params, len);
    } catch (std::exception const&) { set_err(LT_ERR_INTERNAL, "DHT snapshot failed"); return nullptr; }
}

lt_session lt_session_new(const char* settings_json) {
    return lt_session_new_with_dht(settings_json, nullptr, 0);
}

lt_session lt_session_new_with_dht(const char* settings_json, const char* state, size_t len) {
    set_err(LT_OK, "");
    try {
        lt::session_params params;
        if (len != 0) params.dht_state = decode_dht_state(state, len).dht_state;
        params.settings.set_int(lt::settings_pack::alert_mask, LT_ALERT_DEFAULT);

        // PEX (peer exchange) is libtorrent's ut_pex PLUGIN, not a settings_pack
        // toggle, so it can't ride in the settings JSON. We carry the user's choice
        // in a shim-private key "tsl_disable_pex" and strip it before the settings
        // pass (otherwise it would be reported as an unknown setting).
        bool disable_pex = false;
        bool network_paused = false;
        if (settings_json && *settings_json) {
            try {
                auto j = json::parse(settings_json);
                if (auto it = j.find("tsl_network_paused"); it != j.end()) {
                    if (!it->is_boolean()) {
                        set_err(LT_ERR_INVALID, "invalid network pause policy");
                        return 0;
                    }
                    network_paused = it->get<bool>();
                    j.erase(it);
                }
                if (auto it = j.find("tsl_disable_pex"); it != j.end()) {
                    if (it->is_boolean()) disable_pex = it->get<bool>();
                    j.erase(it);
                }
                std::string warn;
                json_into_settings(j, params.settings, &warn);
                if (!warn.empty()) g_last_error = "settings warnings: " + warn;
            } catch (std::exception const& e) {
                set_err(LT_ERR_PARSE, std::string("settings json parse: ") + e.what());
                return 0;
            }
        }

        // If Go has registered storage callbacks, swap in our custom
        // disk_io constructor before the session boots.
        tsl_install_disk_io_on(params);

        // session_params::extensions defaults to {ut_pex, ut_metadata, smart_ban}.
        // To honour DisablePEX, drop the default plugin set and re-add only metadata
        // (needed for magnet links) and smart_ban — leaving ut_pex out so the session
        // never exchanges peers. The add_default_plugins ctor flag has no effect on the
        // session_params overload, so the plugin set is governed solely by .extensions.
        auto slot = std::make_shared<session_slot>();
        // In 2.1, DHT may start before listen sockets exist. A late socket's
        // bootstrap then receives no saved nodes. Retain a bounded set of native
        // hints and reintroduce them through the public API after UDP readiness.
        auto retain_hints = [&](auto const& nodes) {
            for (size_t i = 0; i < std::min<size_t>(32, nodes.size()); ++i)
                slot->restored_dht_nodes.push_back(nodes[i]);
        };
        retain_hints(params.dht_state.nodes);
        retain_hints(params.dht_state.nodes6);
        if (disable_pex) {
            params.extensions.clear();
            slot->s = std::make_unique<lt::session>(std::move(params));
            slot->s->add_extension(&lt::create_ut_metadata_plugin);
            slot->s->add_extension(&lt::create_smart_ban_plugin);
        } else {
            slot->s = std::make_unique<lt::session>(std::move(params));
        }

        if (network_paused) slot->s->pause();
        int64_t id = g_next_sess++;
        {
            std::unique_lock<std::shared_mutex> lk(g_sess_mu);
            g_sessions.emplace(id, slot);
        }
        return id;
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return 0;
    }
}

int lt_session_apply_settings(lt_session id, const char* settings_json) {
    WRAP_BEGIN
    auto slot = get_session(id);
    if (!slot) return set_err(LT_ERR_NOT_FOUND, "session not found");
    if (!settings_json || !*settings_json) return LT_OK;

    json j;
    try { j = json::parse(settings_json); }
    catch (std::exception const& e) { return set_err(LT_ERR_PARSE, e.what()); }

    lt::settings_pack sp;
    std::string warn;
    bool change_pause = false;
    bool network_paused = false;
    if (auto it = j.find("tsl_network_paused"); it != j.end()) {
        if (!it->is_boolean()) return set_err(LT_ERR_INVALID, "invalid network pause policy");
        change_pause = true;
        network_paused = it->get<bool>();
        j.erase(it);
        if (network_paused) slot->s->pause();
    }
    json_into_settings(j, sp, &warn);
    slot->s->apply_settings(std::move(sp));
    if (change_pause && !network_paused) slot->s->resume();
    if (!warn.empty()) g_last_error = "settings warnings: " + warn;
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_session_get_setting_int(lt_session id, const char* name, int64_t* out) {
    WRAP_BEGIN
    auto slot = get_session(id);
    if (!slot) return set_err(LT_ERR_NOT_FOUND, "session not found");
    if (!name || !*name) return set_err(LT_ERR_INVALID, "empty setting name");
    int sid = lt::setting_by_name(name);
    if (sid < 0) return set_err(LT_ERR_NOT_FOUND, "unknown setting");
    if ((sid & lt::settings_pack::type_mask) != lt::settings_pack::int_type_base)
        return set_err(LT_ERR_INVALID, "not an int setting");
    lt::settings_pack sp = slot->s->get_settings();
    if (out) *out = sp.get_int(sid);
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_session_set_ip_filter(lt_session id, const char* p2p_text) {
    WRAP_BEGIN
    auto slot = get_session(id);
    if (!slot) return set_err(LT_ERR_NOT_FOUND, "session not found");
    std::string txt = p2p_text ? p2p_text : "";
    slot->s->set_ip_filter(parse_p2p_filter(txt));
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_session_set_alert_mask(lt_session id, uint32_t mask) {
    WRAP_BEGIN
    auto slot = get_session(id);
    if (!slot) return set_err(LT_ERR_NOT_FOUND, "session not found");
    if (mask == 0) mask = LT_ALERT_DEFAULT;
    lt::settings_pack sp;
    sp.set_int(lt::settings_pack::alert_mask, static_cast<int>(mask));
    slot->s->apply_settings(std::move(sp));
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_session_destroy(lt_session id) {
    WRAP_BEGIN
    std::shared_ptr<session_slot> slot;
    {
        std::unique_lock<std::shared_mutex> lk(g_sess_mu);
        auto it = g_sessions.find(id);
        if (it == g_sessions.end()) return set_err(LT_ERR_NOT_FOUND, "session not found");
        slot = std::move(it->second);
        g_sessions.erase(it);
    }
    // Forget all torrents — we only ever have one session at a time.
    {
        std::unique_lock<std::shared_mutex> lk(g_torr_mu);
        g_torrents.clear();
        g_hash2id.clear();
    }
    slot.reset();
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

// ----- torrent lifecycle -----

// Parse torrent bytes that may be either a full .torrent metainfo or a bare
// info-dict. The bare form is what lt_torrent_metadata_alloc returns
// (ti->info_section()) and what legacy anacrolix-era TorrServer DB records
// store as InfoBytes, so round-tripping metadata through the DB must accept
// it: wrap it into a minimal metainfo dict and re-parse. On failure returns
// nullptr with ec holding the original (full-metainfo) parse error.
static std::shared_ptr<lt::torrent_info> parse_torrent_or_info(
    char const* buf, int len, lt::error_code& ec)
{
    ec.clear();
    auto ti = std::make_shared<lt::torrent_info>(buf, len, ec);
    if (!ec) return ti;
    std::string wrapped;
    wrapped.reserve(static_cast<size_t>(len) + 8);
    wrapped += "d4:info";
    wrapped.append(buf, static_cast<size_t>(len));
    wrapped += 'e';
    lt::error_code ec2;
    auto ti2 = std::make_shared<lt::torrent_info>(
        wrapped.data(), static_cast<int>(wrapped.size()), ec2);
    if (!ec2) { ec.clear(); return ti2; }
    return nullptr;
}

// Keep tracker tiers bounded and reject malformed JSON before calling native code.
static bool valid_tracker_tiers(json const& tiers) {
    if (!tiers.is_array() || tiers.size() > 256) return false;
    std::size_t count = 0;
    for (auto const& urls : tiers) {
        if (!urls.is_array()) return false;
        for (auto const& url : urls) {
            if (!url.is_string() || url.get_ref<std::string const&>().size() > 8192
                || ++count > 1024) return false;
        }
    }
    return true;
}

lt_torrent lt_session_add_torrent(
    lt_session sid,
    const char* link,
    const uint8_t* info_bytes, size_t info_len,
    const char* trackers_json,
    const char* save_path,
    int paused,
    const uint8_t* have_pieces_bitmap, int have_pieces_count)
{
    set_err(LT_OK, "");
    try {
        auto slot = get_session(sid);
        if (!slot) { set_err(LT_ERR_NOT_FOUND, "session not found"); return 0; }

        lt::add_torrent_params atp;
        atp.save_path = save_path ? save_path : ".";

        if (info_bytes && info_len > 0) {
            lt::error_code ec;
            atp.ti = parse_torrent_or_info(
                reinterpret_cast<char const*>(info_bytes), static_cast<int>(info_len), ec);
            if (ec || !atp.ti) { set_err_ec(LT_ERR_PARSE, ec); return 0; }
        } else if (link && *link) {
            std::string l(link);
            if (l.size() == 40) {
                lt::sha1_hash h;
                if (hex_to_sha1(l, h)) {
                    atp.info_hashes.v1 = h;
                } else {
                    lt::error_code ec;
                    lt::parse_magnet_uri(l, atp, ec);
                    if (ec) { set_err_ec(LT_ERR_PARSE, ec); return 0; }
                }
            } else if (l.rfind("magnet:", 0) == 0) {
                lt::error_code ec;
                lt::parse_magnet_uri(l, atp, ec);
                if (ec) { set_err_ec(LT_ERR_PARSE, ec); return 0; }
            } else {
                set_err(LT_ERR_NOT_IMPL, "http/file links must be pre-fetched");
                return 0;
            }
        } else {
            set_err(LT_ERR_INVALID, "link and info_bytes are both empty");
            return 0;
        }

        if (trackers_json && *trackers_json
            && !(atp.ti && atp.ti->priv() && !atp.ti->trackers().empty())) {
            auto tiers = json::parse(trackers_json);
            if (!valid_tracker_tiers(tiers)) { set_err(LT_ERR_INVALID, "invalid tracker tiers"); return 0; }
            atp.trackers.clear();
            atp.tracker_tiers.clear();
            int tier = 0;
            for (auto const& urls : tiers) {
                for (auto const& url : urls) {
                    atp.trackers.push_back(url.get<std::string>());
                    atp.tracker_tiers.push_back(tier);
                }
                ++tier;
            }
            atp.flags |= lt::torrent_flags::override_trackers;
        }

        if (paused) {
            atp.flags |= lt::torrent_flags::paused;
            atp.flags &= ~lt::torrent_flags::auto_managed;
        } else {
            atp.flags &= ~lt::torrent_flags::paused;
            atp.flags |= lt::torrent_flags::auto_managed;
        }

        if (have_pieces_bitmap && have_pieces_count > 0) {
            atp.have_pieces.resize(have_pieces_count, false);
            for (int i = 0; i < have_pieces_count; ++i) {
                uint8_t byte = have_pieces_bitmap[i / 8];
                if ((byte >> (i % 8)) & 1u) {
                    atp.have_pieces.set_bit(lt::piece_index_t{i});
                }
            }
            // Go verified exact lengths and metadata SHA-1 hashes before addition.
            // Share that verified bitmap without doing a second full disk scan.
            atp.flags |= lt::torrent_flags::no_verify_files;
        }

        lt::error_code ec;
        lt::torrent_handle h = slot->s->add_torrent(std::move(atp), ec);
        if (ec) { set_err_ec(LT_ERR_INTERNAL, ec); return 0; }
        if (!h.is_valid()) { set_err(LT_ERR_INTERNAL, "invalid handle"); return 0; }

        return register_torrent(h);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return 0;
    }
}

int lt_torrent_remove(lt_session sid, lt_torrent tid, int delete_files) {
    WRAP_BEGIN
    auto slot = get_session(sid);
    if (!slot) return set_err(LT_ERR_NOT_FOUND, "session not found");
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    auto flags = delete_files ? lt::session::delete_files : lt::remove_flags_t{};
    slot->s->remove_torrent(h, flags);
    unregister_torrent(tid);
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_pause(lt_torrent tid) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    h.pause();
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_url_seed(lt_torrent tid, const char* url, int remove, int allow_local) {
    WRAP_BEGIN
    if (!url || std::strlen(url) > 8192) return set_err(LT_ERR_INVALID, "invalid mirror URL");
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    auto ti = h.torrent_file();
    if (!ti || (!remove && ti->priv())) return set_err(LT_ERR_INVALID, "known public metadata required");
#ifdef TSL_HAVE_LT_INTERNALS
    auto tor = h.native_handle();
    if (!tor) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    std::string value(url);
    lt::post(tor->session().get_context(), [tor, value=std::move(value), remove, allow_local]() {
        tor->remove_web_seed(value);
        if (!remove && tor->web_seeds().size() < 16)
            tor->add_web_seed(value, {}, {}, allow_local ? lt::aux::web_seed_flag_t{} : lt::aux::torrent::no_local_ips);
    });
    return LT_OK;
#else
    return set_err(LT_ERR_NOT_IMPL, "source destination guard unavailable");
#endif
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_replace_trackers(lt_torrent tid, const char* tiers_json) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    auto ti = h.torrent_file();
    if (!ti || ti->priv()) return set_err(LT_ERR_INVALID, "public metadata required");
    auto tiers = json::parse(tiers_json ? tiers_json : "[]");
    if (!valid_tracker_tiers(tiers)) return set_err(LT_ERR_INVALID, "invalid tracker tiers");
    std::vector<lt::announce_entry> entries;
    int tier = 0;
    for (auto const& urls : tiers) {
        for (auto const& url : urls) {
            lt::announce_entry e(url.get<std::string>());
            e.tier = static_cast<std::uint8_t>(std::min(tier, 255));
            entries.push_back(std::move(e));
        }
        ++tier;
    }
    h.replace_trackers(entries);
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

char* lt_torrent_sparse_json_alloc(lt_torrent tid, const char* ranges_json, size_t* out_len) {
    set_err(LT_OK, "");
    try {
        lt::torrent_handle h;
        std::shared_ptr<sparse_snapshot> snapshot;
        {
            std::shared_lock<std::shared_mutex> lk(g_torr_mu);
            auto it = g_torrents.find(tid);
            if (it == g_torrents.end()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return nullptr; }
            h = it->second.h;
            snapshot = it->second.sparse;
        }
        auto ranges = json::parse(ranges_json ? ranges_json : "[]");
        int pieces = 0;
        if (!ranges.is_array() || ranges.size() > 8) { set_err(LT_ERR_INVALID, "invalid sparse ranges"); return nullptr; }
        for (auto const& range : ranges) {
            if (!range.is_array() || range.size() != 2 || !range[0].is_number_integer()
                || !range[1].is_number_integer()) { set_err(LT_ERR_INVALID, "invalid sparse range"); return nullptr; }
            auto const first = range[0].get<std::int64_t>();
            auto const count = range[1].get<std::int64_t>();
            if (first < 0 || first > INT32_MAX || count < 1 || count > 128 || first+count > INT32_MAX
                || (pieces += static_cast<int>(count)) > 256) { set_err(LT_ERR_INVALID, "sparse range exceeds bounds"); return nullptr; }
        }
        std::lock_guard<std::mutex> lk(snapshot->mu);
        std::string cached = snapshot->cached;
#ifdef TSL_HAVE_LT_INTERNALS
        auto const now = std::chrono::steady_clock::now();
        if (!snapshot->pending && now-snapshot->last_request >= std::chrono::seconds(2)) {
            auto tor = h.native_handle();
            if (tor) {
                snapshot->pending = true;
                snapshot->last_request = now;
                lt::post(tor->session().get_context(), [tor, snapshot, ranges=std::move(ranges)]() {
                    json result = {{"known", false}};
                    json hints = json::array();
                    bool private_torrent = true;
                    try {
                        if (tor->valid_metadata()) {
                            private_torrent = tor->torrent_file().priv();
                            int const total = tor->torrent_file().num_pieces();
                            json windows = json::array();
                            for (auto const& range : ranges) {
                                int const first = range[0].get<int>();
                                int const count = range[1].get<int>();
                                if (first >= total || count > total-first) continue;
                                windows.push_back({{"first_piece", first}, {"availability", std::vector<int>(count, 0)},
                                    {"unchoked_suppliers", 0}});
                            }
                            int sampled = 0, useful = 0, downloading = 0, choked = 0, snubbed = 0, pending = 0;
                            int tracker = 0, dht = 0, pex = 0, incoming = 0;
                            std::int64_t outstanding = 0, queue_ms = 0, queued = 0;
                            for (auto* peer : *tor) {
                                if (sampled == 512) break;
                                ++sampled;
                                pending += peer->is_connecting();
                                choked += peer->has_peer_choked();
                                snubbed += peer->flow_snubbed();
                                incoming += !peer->is_outgoing();
                                if (auto* info = peer->peer_info_struct()) {
                                    auto const source = info->peer_source();
                                    tracker += bool(source & lt::peer_info::tracker);
                                    dht += bool(source & lt::peer_info::dht);
                                    pex += bool(source & lt::peer_info::pex);
                                }
                                bool supplies = false;
                                if (!peer->is_connecting() && !peer->is_disconnecting()) {
                                    for (auto& window : windows) {
                                        int const first = window["first_piece"].get<int>();
                                        auto& available = window["availability"];
                                        for (std::size_t i = 0; i < available.size(); ++i) {
                                            if (!peer->has_piece(lt::piece_index_t{first+static_cast<int>(i)})) continue;
                                            available[i] = available[i].get<int>()+1;
                                            supplies = true;
                                            if (i == 0 && !peer->has_peer_choked())
                                                window["unchoked_suppliers"] = window["unchoked_suppliers"].get<int>()+1;
                                        }
                                    }
                                    outstanding += peer->outstanding_bytes();
                                    queued += peer->request_queue().size()+peer->download_queue().size();
                                    queue_ms = std::max(queue_ms, lt::total_milliseconds(peer->download_queue_time()));
                                }
                                useful += supplies;
                                downloading += supplies && peer->statistics().download_payload_rate() > 0;
                                if (!private_torrent && hints.size() < 32 && !peer->on_parole()
                                    && peer->type() == lt::connection_type::bittorrent
                                    && !peer->is_connecting() && !peer->is_disconnecting()
                                    && peer->statistics().download_payload_rate() > 0) {
                                    auto const ep = peer->remote();
                                    if (ep.port() != 0 && !ep.address().is_unspecified()
                                        && !ep.address().is_multicast())
                                        hints.push_back({{"ip", ep.address().to_string()}, {"port", ep.port()}});
                                }
                            }
                            lt::torrent_status status;
                            tor->status(&status, {});
                            auto const timestamp = std::chrono::duration_cast<std::chrono::milliseconds>(
                                std::chrono::system_clock::now().time_since_epoch()).count();
                            result = {{"known", true}, {"private", tor->torrent_file().priv()}, {"sampled_at_ms", timestamp}, {"sampled_peers", sampled},
                                {"truncated", tor->num_peers() > sampled}, {"useful_peers", useful},
                                {"useful_downloading_peers", downloading}, {"choked_peers", choked}, {"snubbed_peers", snubbed},
                                {"pending_connections", pending}, {"tracker_peers", tracker}, {"dht_peers", dht},
                                {"pex_peers", pex}, {"incoming_peers", incoming}, {"outstanding_bytes", outstanding},
                                {"queued_blocks", queued}, {"max_queue_ms", queue_ms}, {"failed_bytes", status.total_failed_bytes},
                                {"redundant_bytes", status.total_redundant_bytes}, {"windows", std::move(windows)}};
                        }
                    } catch (...) { result = {{"known", false}}; }
                    std::lock_guard<std::mutex> done(snapshot->mu);
                    snapshot->cached = result.dump();
                    if (private_torrent) snapshot->resume_peers = "[]";
                    else if (!hints.empty()) {
                        snapshot->resume_peers = hints.dump();
                        snapshot->peers_at = std::chrono::steady_clock::now();
                        snapshot->peers_at_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                            std::chrono::system_clock::now().time_since_epoch()).count();
                    }
                    snapshot->pending = false;
                });
            }
        }
#endif
        return alloc_string(cached, out_len);
    } catch (std::exception const& e) { set_err(LT_ERR_INVALID, e.what()); return nullptr; }
}

char* lt_torrent_resume_peers_json_alloc(lt_torrent tid, size_t* out_len) {
    set_err(LT_OK, "");
    try {
    std::shared_lock<std::shared_mutex> lk(g_torr_mu);
    auto it = g_torrents.find(tid);
    if (it == g_torrents.end()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return nullptr; }
    auto snapshot = it->second.sparse;
    std::lock_guard<std::mutex> guard(snapshot->mu);
    if (std::chrono::steady_clock::now()-snapshot->peers_at > std::chrono::minutes(10))
        return alloc_string("{\"observed_at_ms\":0,\"peers\":[]}", out_len);
    return alloc_string(json{{"observed_at_ms", snapshot->peers_at_ms},
        {"peers", json::parse(snapshot->resume_peers)}}.dump(), out_len);
    } catch (std::exception const& e) { set_err(LT_ERR_INTERNAL, e.what()); return nullptr; }
}

int lt_torrent_restore_peers(lt_torrent tid, const char* peers_json) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    auto ti = h.torrent_file();
    if (!ti || ti->priv()) return set_err(LT_ERR_INVALID, "known public metadata required");
    auto peers = json::parse(peers_json ? peers_json : "[]");
    if (!peers.is_array() || peers.size() > 32) return set_err(LT_ERR_INVALID, "peer hint limit");
    std::vector<lt::tcp::endpoint> endpoints;
    for (auto const& item : peers) {
        if (!item.is_object() || !item.contains("ip") || !item["ip"].is_string()
            || !item.contains("port") || !item["port"].is_number_integer())
            return set_err(LT_ERR_INVALID, "invalid peer hint");
        lt::error_code ec;
        auto const address = lt::make_address(item["ip"].get<std::string>(), ec);
        int const port = item["port"].get<int>();
        if (ec || address.is_unspecified() || address.is_multicast() || port < 1 || port > 65535)
            return set_err(LT_ERR_INVALID, "invalid peer endpoint");
        endpoints.emplace_back(address, static_cast<std::uint16_t>(port));
    }
    // Native peer-list failure counters and reconnect backoff own all retries.
    for (auto const& ep : endpoints) h.connect_peer(ep, lt::peer_info::resume_data);
    return LT_OK;
    WRAP_END(LT_ERR_INVALID)
}

int lt_torrent_resume(lt_torrent tid) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    h.resume();
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_force_recheck(lt_torrent tid) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    h.force_recheck();
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

// Re-announce to all trackers now (ignoring the min interval). Used at playback
// start to grab peers fast for a lazily-added torrent.
int lt_torrent_force_reannounce(lt_torrent tid) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    h.force_reannounce();
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

// Force an immediate DHT announce for this torrent.
int lt_torrent_force_dht_announce(lt_torrent tid) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    h.force_dht_announce();
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_set_max_connections(lt_torrent tid, int n) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    h.set_max_connections(n);
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

// ----- metadata accessors -----

int lt_torrent_have_metadata(lt_torrent tid) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    auto ti = h.torrent_file();
    return (ti && ti->is_valid()) ? 1 : 0;
    WRAP_END(LT_ERR_INTERNAL)
}

char* lt_torrent_metadata_alloc(lt_torrent tid, size_t* out_len) {
    set_err(LT_OK, "");
    try {
        auto h = get_torrent(tid);
        if (!h.is_valid()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return nullptr; }
        auto ti = h.torrent_file();
        if (!ti || !ti->is_valid()) { set_err(LT_ERR_NOT_FOUND, "no metadata yet"); return nullptr; }
        auto const& info = ti->info_section();
        std::string s(info.data(), info.size());
        return alloc_string(s, out_len);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return nullptr;
    }
}

int lt_torrent_num_files(lt_torrent tid) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    auto ti = h.torrent_file();
    if (!ti) return 0;
    return ti->num_files();
    WRAP_END(LT_ERR_INTERNAL)
}

size_t lt_torrent_file_path(lt_torrent tid, int idx, char* buf, size_t cap) {
    set_err(LT_OK, "");
    try {
        auto h = get_torrent(tid);
        if (!h.is_valid()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return 0; }
        auto ti = h.torrent_file();
        if (!ti) { set_err(LT_ERR_NOT_FOUND, "no metadata yet"); return 0; }
        auto const& fs = ti->files();
        if (idx < 0 || idx >= fs.num_files()) { set_err(LT_ERR_INVALID, "file index out of range"); return 0; }
        std::string p = fs.file_path(lt::file_index_t{idx});
        return copy_string(p, buf, cap);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return 0;
    }
}

int64_t lt_torrent_file_size(lt_torrent tid, int idx) {
    set_err(LT_OK, "");
    try {
        auto h = get_torrent(tid);
        if (!h.is_valid()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return -1; }
        auto ti = h.torrent_file();
        if (!ti) { set_err(LT_ERR_NOT_FOUND, "no metadata yet"); return -1; }
        auto const& fs = ti->files();
        if (idx < 0 || idx >= fs.num_files()) { set_err(LT_ERR_INVALID, "file index out of range"); return -1; }
        return fs.file_size(lt::file_index_t{idx});
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return -1;
    }
}

int64_t lt_torrent_file_offset(lt_torrent tid, int idx) {
    set_err(LT_OK, "");
    try {
        auto h = get_torrent(tid);
        if (!h.is_valid()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return -1; }
        auto ti = h.torrent_file();
        if (!ti) { set_err(LT_ERR_NOT_FOUND, "no metadata yet"); return -1; }
        auto const& fs = ti->files();
        if (idx < 0 || idx >= fs.num_files()) { set_err(LT_ERR_INVALID, "file index out of range"); return -1; }
        return fs.file_offset(lt::file_index_t{idx});
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return -1;
    }
}

int lt_torrent_num_pieces(lt_torrent tid) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    auto ti = h.torrent_file();
    if (!ti) return 0;
    return ti->num_pieces();
    WRAP_END(LT_ERR_INTERNAL)
}

// lt_torrent_have_piece reports whether libtorrent's picker considers the piece
// downloaded+verified. Used by the streaming reader to detect a have-bitfield /
// cache desync (libtorrent thinks it has a piece our cache has evicted) so it can
// un-have just that piece on demand. Returns 1 = have, 0 = not, -1 = error.
int lt_torrent_have_piece(lt_torrent tid, int piece_idx) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    return h.have_piece(lt::piece_index_t{piece_idx}) ? 1 : 0;
    WRAP_END(LT_ERR_INTERNAL)
}

int64_t lt_torrent_piece_length(lt_torrent tid) {
    set_err(LT_OK, "");
    try {
        auto h = get_torrent(tid);
        if (!h.is_valid()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return -1; }
        auto ti = h.torrent_file();
        if (!ti) return 0;
        return ti->piece_length();
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return -1;
    }
}

int64_t lt_torrent_total_size(lt_torrent tid) {
    set_err(LT_OK, "");
    try {
        auto h = get_torrent(tid);
        if (!h.is_valid()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return -1; }
        auto ti = h.torrent_file();
        if (!ti) return 0;
        return ti->total_size();
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return -1;
    }
}

size_t lt_torrent_display_name(lt_torrent tid, char* buf, size_t cap) {
    set_err(LT_OK, "");
    try {
        auto h = get_torrent(tid);
        if (!h.is_valid()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return 0; }
        auto st = h.status(lt::torrent_handle::query_name);
        return copy_string(st.name, buf, cap);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return 0;
    }
}

size_t lt_torrent_info_hash_hex(lt_torrent tid, char* buf, size_t cap) {
    set_err(LT_OK, "");
    try {
        auto h = get_torrent(tid);
        if (!h.is_valid()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return 0; }
        std::string hex = sha1_hex(h.info_hashes().v1);
        return copy_string(hex, buf, cap);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return 0;
    }
}

// ----- priorities & streaming -----

int lt_torrent_set_piece_priority(lt_torrent tid, int piece_idx, int prio) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    if (prio < 0 || prio > 7) return set_err(LT_ERR_INVALID, "prio out of range");
    h.piece_priority(lt::piece_index_t{piece_idx},
                     static_cast<lt::download_priority_t>(static_cast<std::uint8_t>(prio)));
    refresh_connect_candidates(h);
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

// Set every piece to the same priority in one call (prioritize_pieces). Used
// to flip a torrent to "download nothing" (prio 0) so streaming only pulls the
// reader's window + preload buffer instead of the whole torrent.
int lt_torrent_set_all_pieces_priority(lt_torrent tid, int prio) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    if (prio < 0 || prio > 7) return set_err(LT_ERR_INVALID, "prio out of range");
    auto ti = h.torrent_file();
    if (!ti) return set_err(LT_ERR_NOT_FOUND, "no metadata yet");
    std::vector<lt::download_priority_t> v(
        static_cast<std::size_t>(ti->num_pieces()),
        static_cast<lt::download_priority_t>(static_cast<std::uint8_t>(prio)));
    h.prioritize_pieces(v);
    refresh_connect_candidates(h);
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

// Set the FULL piece-priority vector in one prioritize_pieces call. `prios` is an
// array of length `count` (== num_pieces); entry i is piece i's priority (0..7,
// clamped). This drives streaming DECLARATIVELY: each tick the cache rebuilds the
// whole vector (window pieces graded, pins/preload raised, everything else 0) and
// applies it atomically, so no piece can be left at a stale priority>0 and keep
// downloading past the window. Mismatched count is rejected (stale metadata).
int lt_torrent_prioritize_pieces(lt_torrent tid, const int* prios, int count) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    if (!prios || count < 0) return set_err(LT_ERR_INVALID, "bad priorities");
    auto ti = h.torrent_file();
    if (!ti) return set_err(LT_ERR_NOT_FOUND, "no metadata yet");
    if (count != ti->num_pieces()) return set_err(LT_ERR_INVALID, "count != num_pieces");
    std::vector<lt::download_priority_t> v;
    v.reserve(static_cast<std::size_t>(count));
    for (int i = 0; i < count; ++i) {
        int p = prios[i];
        if (p < 0) p = 0;
        if (p > 7) p = 7;
        v.push_back(static_cast<lt::download_priority_t>(static_cast<std::uint8_t>(p)));
    }
    h.prioritize_pieces(v);
    refresh_connect_candidates(h);
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

// Per-piece "un-have". The streaming cache evicts pieces that libtorrent still
// records as "have"; a seek back into an evicted region (or a playhead piece
// that was downloaded ahead, cached, then evicted before the reader reached it)
// would then never be re-downloaded — the picker skips pieces it believes it
// owns, and priority on a complete piece is ignored. There is no public
// torrent_handle API for this, so we reach into the internal piece_picker —
// only safe on the session's network thread, hence the post() onto its
// io_context.
//
// `prio` is the download priority to leave the piece at AFTER clearing the have
// bit. It is applied inside the same posted lambda, *after* we_dont_have, so it
// can't be lost to a race: doing set_piece_priority from the caller's thread (as
// we used to) ran concurrently with this lambda, which transiently drops the
// piece to dont_download to satisfy the picker — the lambda's store would then
// clobber the caller's, leaving the piece at priority 0 and never re-requested.
// Callers that want the piece re-downloaded now (the streaming reader) pass a
// non-zero prio; pass 0 to un-have and leave it lazy.
int lt_torrent_we_dont_have(lt_torrent tid, int piece_idx, int prio) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    if (prio < 0 || prio > 7) return set_err(LT_ERR_INVALID, "prio out of range");
#ifdef TSL_HAVE_LT_INTERNALS
    auto tor = h.native_handle();
    if (!tor) return set_err(LT_ERR_NOT_FOUND, "no native handle");
    h.reset_piece_deadline(lt::piece_index_t{piece_idx});
    lt::io_context& ioc = tor->session().get_context();
    lt::post(ioc, [tor, piece_idx, prio]() {
        lt::piece_index_t const pi{piece_idx};
        // A stale eviction marker must not reset a newer in-flight download.
        // In particular, clearing its hash/write bookkeeping can make a valid
        // peer look corrupt when a queued hash reads a replaced cache entry.
        if (tor->valid_metadata() && tor->has_picker() && piece_idx >= 0
            && pi < tor->torrent_file().end_piece()
            && !tor->picker().is_piece_flushed(pi)) {
            tor->set_piece_priority(pi,
                static_cast<lt::download_priority_t>(static_cast<std::uint8_t>(prio)));
            return;
        }
        tor->flow_forget_piece(pi,
            static_cast<lt::download_priority_t>(static_cast<std::uint8_t>(prio)));
    });
    return LT_OK;
#else
    // Linked against a shared libtorrent that doesn't export piece_picker /
    // torrent internals (distro or Homebrew). Per-piece un-have isn't
    // available, so just clear any deadline. The consequence is the documented
    // pre-we_dont_have limitation: a seek back into an already-evicted region
    // won't re-download (forward seek, playback and short rewind still work).
    // reset_piece_deadline is public torrent_handle API, so this always links.
    h.reset_piece_deadline(lt::piece_index_t{piece_idx});
    return LT_OK;
#endif
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_prune_partial(lt_torrent tid, int piece_idx) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
#ifdef TSL_HAVE_LT_INTERNALS
    auto tor = h.native_handle();
    if (!tor) return set_err(LT_ERR_NOT_FOUND, "no native handle");
    lt::post(tor->session().get_context(), [tor, piece_idx]() {
        lt::piece_index_t const pi{piece_idx};
        if (!tor->valid_metadata() || !tor->has_storage() || !tor->has_picker()
            || piece_idx < 0 || pi >= tor->torrent_file().end_piece()
            || tor->picker().have_piece(pi)) return;
        // A hash worker or deferred write completion still owns this data.
        // Never clear it or synthesize a hash failure that would blame peers.
        for (auto const& dp : tor->picker().get_download_queue())
            if (dp.index == pi && (dp.hashing || dp.writing != 0 || dp.locked)) return;
        using U = typename lt::aux::underlying_index_t<lt::storage_index_t>::type;
        if (!lt_storage_prune_partial(static_cast<int64_t>(static_cast<U>(tor->storage())), piece_idx)) return;
        // No native network operation can interleave the cache removal and
        // picker reset. Late wire blocks are stored normally, never acknowledged
        // as successful writes while silently discarding their bytes.
        int const blocks = tor->picker().blocks_in_piece(pi);
        for (int block = 0; block < blocks; ++block)
            tor->cancel_block(lt::piece_block{pi, block});
        tor->set_piece_priority(pi, lt::dont_download);
        tor->flow_forget_piece(pi, lt::dont_download);
    });
#endif
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_evict_complete(lt_torrent tid, int piece_idx) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return 0;
#ifdef TSL_HAVE_LT_INTERNALS
    auto tor = h.native_handle();
    if (!tor) return 0;
    auto result = std::make_shared<std::promise<int>>();
    auto ready = result->get_future();
    lt::post(tor->session().get_context(), [tor, piece_idx, result]() {
        lt::piece_index_t const pi{piece_idx};
        int removed = 0;
        if (tor->valid_metadata() && tor->has_storage() && piece_idx >= 0
            && pi < tor->torrent_file().end_piece()
            && (!tor->has_picker() || tor->picker().is_piece_flushed(pi))) {
            // Hash/write completions and cache removal share this context.
            // Keep the native have bit; reconciliation remains lazy on demand.
            using U = typename lt::aux::underlying_index_t<lt::storage_index_t>::type;
            removed = lt_storage_evict_complete(
                static_cast<int64_t>(static_cast<U>(tor->storage())), piece_idx);
        }
        result->set_value(removed);
    });
    // A concurrently closing session may stop its context before this task.
    // Do not strand the Go eviction goroutine during shutdown.
    if (ready.wait_for(std::chrono::seconds(2)) != std::future_status::ready) return 0;
    return ready.get();
#else
    return 0;
#endif
    WRAP_END(0)
}

int lt_torrent_set_piece_deadline(lt_torrent tid, int piece_idx, int deadline_ms, int alert_when_ready) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    lt::deadline_flags_t flags = {};
    if (alert_when_ready) flags |= lt::torrent_handle::alert_when_available;
    h.set_piece_deadline(lt::piece_index_t{piece_idx}, deadline_ms, flags);
    refresh_connect_candidates(h);
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_set_sequential_download(lt_torrent tid, int enable) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    // The legacy set_sequential_download(bool) is deprecated in lt 2.0; drive the
    // torrent_flags::sequential_download flag directly instead.
    if (enable)
        h.set_flags(lt::torrent_flags::sequential_download);
    else
        h.unset_flags(lt::torrent_flags::sequential_download);
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_reset_piece_deadline(lt_torrent tid, int piece_idx) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    // reset_piece_deadline takes the piece off the time-critical list. Public
    // torrent_handle API, so this always links (no internals needed).
    h.reset_piece_deadline(lt::piece_index_t{piece_idx});
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_clear_piece_deadlines(lt_torrent tid) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    h.clear_piece_deadlines();
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

int lt_torrent_set_file_priority(lt_torrent tid, int file_idx, int prio) {
    WRAP_BEGIN
    auto h = get_torrent(tid);
    if (!h.is_valid()) return set_err(LT_ERR_NOT_FOUND, "torrent not found");
    if (prio < 0 || prio > 7) return set_err(LT_ERR_INVALID, "prio out of range");
    h.file_priority(lt::file_index_t{file_idx},
                    static_cast<lt::download_priority_t>(static_cast<std::uint8_t>(prio)));
    return LT_OK;
    WRAP_END(LT_ERR_INTERNAL)
}

// ----- status & stats -----

size_t lt_torrent_status_json(lt_torrent tid, char* buf, size_t cap) {
    set_err(LT_OK, "");
    try {
        auto h = get_torrent(tid);
        if (!h.is_valid()) { set_err(LT_ERR_NOT_FOUND, "torrent not found"); return 0; }
        std::string s = json_dump(status_to_json(h));
        return copy_string(s, buf, cap);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return 0;
    }
}

char* lt_session_stats_json_alloc(lt_session sid, size_t* out_len) {
    set_err(LT_OK, "");
    try {
        auto slot = get_session(sid);
        if (!slot) { set_err(LT_ERR_NOT_FOUND, "session not found"); return nullptr; }
        slot->s->post_session_stats();
        json j;
        j["requested"] = true;
        std::string s = json_dump(j);
        return alloc_string(s, out_len);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return nullptr;
    }
}

// ----- alert pump -----

int lt_session_wait_alert(lt_session sid, int timeout_ms) {
    WRAP_BEGIN
    auto slot = get_session(sid);
    if (!slot) return set_err(LT_ERR_NOT_FOUND, "session not found");
    std::lock_guard<std::mutex> lk(slot->pump_mu);
    if (timeout_ms < 0) timeout_ms = 1000 * 60 * 60; // cap at 1h
    auto* a = slot->s->wait_for_alert(std::chrono::milliseconds(timeout_ms));
    return a ? 1 : 0;
    WRAP_END(LT_ERR_INTERNAL)
}

char* lt_session_pop_alerts_json_alloc(lt_session sid, size_t* out_len) {
    set_err(LT_OK, "");
    try {
        auto slot = get_session(sid);
        if (!slot) { set_err(LT_ERR_NOT_FOUND, "session not found"); return nullptr; }

        std::vector<lt::alert*> alerts;
        // Borrowed alert pointers and the hint queue stay protected until JSON
        // conversion finishes; a concurrent pop must not invalidate the batch.
        std::lock_guard<std::mutex> lk(slot->pump_mu);
        slot->s->pop_alerts(&alerts);
        json arr = json::array();
        for (auto* a : alerts) {
            if (auto* ready = lt::alert_cast<lt::listen_succeeded_alert>(a)) {
                if (ready->socket_type == lt::socket_type_t::utp
                    && !ready->address.is_loopback()
                    && !slot->restored_dht_nodes.empty()) {
                    bool enabled = slot->s->get_settings().get_bool(lt::settings_pack::enable_dht);
                    if (enabled) {
                        for (auto const& ep : slot->restored_dht_nodes) {
                            if (ep.address().is_v4() == ready->address.is_v4())
                                slot->s->add_dht_node({ep.address().to_string(), ep.port()});
                        }
                    }
                }
            }
            try { arr.push_back(alert_to_json(a)); }
            catch (std::exception const&) { /* skip malformed */ }
        }
        std::string s = json_dump(arr);
        return alloc_string(s, out_len);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return nullptr;
    }
}

// Storage callbacks are now installed via lt_install_storage_callbacks_full
// (declared in lt_disk_io.h, implemented in lt_disk_io.cpp). The old
// 3-pointer API was removed in Etap 4.

// ----- parsers (utility) -----

static char* parse_atp_to_json(lt::add_torrent_params const& atp, size_t* out_len) {
    json j;
    j["info_hash"] = sha1_hex(atp.info_hashes.v1);
    j["display_name"] = atp.name;
    json tr = json::array();
    for (auto const& t : atp.trackers) tr.push_back(t);
    j["trackers"] = std::move(tr);
    json tiers = json::array();
    for (std::size_t i = 0; i < atp.trackers.size(); ++i) {
        int const tier = i < atp.tracker_tiers.size() ? atp.tracker_tiers[i] : 0;
        while (tiers.size() <= static_cast<std::size_t>(tier)) tiers.push_back(json::array());
        tiers[tier].push_back(atp.trackers[i]);
    }
    j["tracker_tiers"] = std::move(tiers);
    if (atp.ti) {
        auto const& info = atp.ti->info_section();
        j["has_metadata"]  = true;
        j["private"] = atp.ti->priv();
        j["metadata_size"] = static_cast<int>(info.size());
        j["num_pieces"]    = atp.ti->num_pieces();
        j["piece_length"]  = atp.ti->piece_length();
        j["total_size"]    = atp.ti->total_size();
        std::string piece_hashes;
        if (atp.ti->v1()) {
            piece_hashes.reserve(static_cast<std::size_t>(atp.ti->num_pieces()) * 40);
            for (int i = 0; i < atp.ti->num_pieces(); ++i)
                piece_hashes += sha1_hex(atp.ti->hash_for_piece(lt::piece_index_t{i}));
        }
        j["piece_hashes"] = std::move(piece_hashes);
        json seeds = json::array();
        for (auto const& seed : atp.ti->web_seeds()) { if (seeds.size() == 16) break; seeds.push_back(seed.url); }
        j["web_seeds"] = std::move(seeds);
    } else {
        j["has_metadata"]  = false;
        j["metadata_size"] = 0;
        j["num_pieces"]    = 0;
        j["piece_length"]  = 0;
        j["total_size"]    = 0;
    }
    std::string s = json_dump(j);
    return alloc_string(s, out_len);
}

char* lt_parse_magnet_alloc(const char* uri, size_t* out_len) {
    set_err(LT_OK, "");
    try {
        if (!uri || !*uri) { set_err(LT_ERR_INVALID, "empty uri"); return nullptr; }
        lt::add_torrent_params atp;
        lt::error_code ec;
        lt::parse_magnet_uri(uri, atp, ec);
        if (ec) { set_err_ec(LT_ERR_PARSE, ec); return nullptr; }
        return parse_atp_to_json(atp, out_len);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return nullptr;
    }
}

char* lt_parse_torrent_bytes_alloc(const uint8_t* buf, size_t len, size_t* out_len) {
    set_err(LT_OK, "");
    try {
        if (!buf || len == 0) { set_err(LT_ERR_INVALID, "empty bytes"); return nullptr; }
        lt::error_code ec;
        auto ti = parse_torrent_or_info(
            reinterpret_cast<char const*>(buf), static_cast<int>(len), ec);
        if (ec || !ti) { set_err_ec(LT_ERR_PARSE, ec); return nullptr; }
        lt::add_torrent_params atp;
        atp.ti = ti;
        atp.info_hashes = ti->info_hashes();
        atp.name = ti->name();
        for (auto const& tracker : ti->trackers()) {
            atp.trackers.push_back(tracker.url);
            atp.tracker_tiers.push_back(tracker.tier);
        }
        return parse_atp_to_json(atp, out_len);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return nullptr;
    }
}

char* lt_parse_torrent_file_alloc(const char* path, size_t* out_len) {
    set_err(LT_OK, "");
    try {
        if (!path || !*path) { set_err(LT_ERR_INVALID, "empty path"); return nullptr; }
        std::ifstream f(path, std::ios::binary);
        if (!f) { set_err(LT_ERR_IO, std::string("cannot open ") + path); return nullptr; }
        std::vector<char> buf((std::istreambuf_iterator<char>(f)), {});
        if (buf.empty()) { set_err(LT_ERR_IO, "file is empty"); return nullptr; }
        return lt_parse_torrent_bytes_alloc(
            reinterpret_cast<uint8_t const*>(buf.data()), buf.size(), out_len);
    } catch (std::exception const& e) {
        set_err(LT_ERR_INTERNAL, e.what());
        return nullptr;
    }
}

} // extern "C"
