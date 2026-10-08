// Exercise the production disk upload completion with controlled cache reads.
// Including the implementation keeps this fixture outside the public shim API.
#include <libtorrent/aux_/session_settings.hpp>
#include <libtorrent/performance_counters.hpp>
#include "../server/lt/lt_disk_io.cpp"
#include <iostream>
#include <stdexcept>
#include <boost/asio/executor_work_guard.hpp>

namespace {
int supplied;
bool lifecycle = false, entered = false, released = false;
std::mutex fixture_mu;
std::condition_variable fixture_cv;
std::array<uint8_t, 100> stored{};
std::vector<std::string> events;
int read_fixture(int64_t, int, int64_t, uint8_t* buffer, int length) {
    if (lifecycle) {
        std::lock_guard<std::mutex> lock(fixture_mu);
        events.push_back("read");
        std::memcpy(buffer, stored.data(), length);
        return length;
    }
    if (supplied > 0) std::memset(buffer, 0x5a, std::min(supplied, length));
    return supplied;
}
int write_fixture(int64_t, int, int64_t, uint8_t const* buffer, int length) {
    if (!lifecycle) return supplied;
    std::unique_lock<std::mutex> lock(fixture_mu);
    entered = true; fixture_cv.notify_all();
    fixture_cv.wait(lock, [] { return released; });
    std::memcpy(stored.data(), buffer, length);
    events.push_back("write");
    return length;
}
}

int main() {
    lt::io_context io;
    lt::aux::session_settings settings;
    lt::counters counters;
    tsl_storage_callbacks callbacks{};
    callbacks.read = read_fixture;
    callbacks.write = write_fixture;
    callbacks.open = [](int64_t, uint8_t const*, int, int64_t) {};
    callbacks.close = [](int64_t) {
        if (lifecycle) { std::lock_guard<std::mutex> lock(fixture_mu); events.push_back("close"); }
    };
    callbacks.clear_piece = [](int64_t, int) {
        std::lock_guard<std::mutex> lock(fixture_mu); events.push_back("clear");
    };
    callbacks.deleted = [](int64_t) {
        std::lock_guard<std::mutex> lock(fixture_mu); events.push_back("delete");
    };
    tsl_disk_io disk(io, settings, counters, callbacks);
    lt::file_storage files;
    files.add_file("fixture", 100);
    files.set_piece_length(16*1024);
    files.set_num_pieces(1);
    lt::aux::vector<lt::download_priority_t, lt::file_index_t> priorities;
#if LIBTORRENT_VERSION_NUM >= 20100
    lt::renamed_files renamed;
    lt::storage_params params(files, renamed, "", "", lt::storage_mode_sparse, priorities, lt::sha1_hash{}, true, false);
#else
    lt::storage_params params(files, nullptr, "", lt::storage_mode_sparse, priorities, lt::sha1_hash{});
#endif
    auto storage = disk.new_torrent(params, {});
    auto guard = boost::asio::make_work_guard(io);
    auto run = [&](bool& completed) {
        auto const until = std::chrono::steady_clock::now()+std::chrono::seconds(5);
        while (!completed && std::chrono::steady_clock::now() < until) io.run_for(std::chrono::milliseconds(20));
        if (!completed) throw std::runtime_error("disk completion timeout");
    };
    for (int count : {-1, 0, 99, 100}) {
        supplied = count;
        bool completed = false;
        disk.async_read(lt::storage_index_t{1}, {lt::piece_index_t{0}, 0, 100},
            [&](lt::disk_buffer_holder buffer, lt::storage_error const& error) {
                completed = true;
                if (bool(error.ec) != (count != 100))
                    throw std::runtime_error("missing or partial upload block was accepted");
                if (!error.ec) {
                    for (int i = 0; i < 100; ++i)
                        if (uint8_t(buffer.data()[i]) != 0x5a)
                            throw std::runtime_error("upload payload changed");
                }
            });
        if (completed) throw std::runtime_error("reentrant disk completion");
        run(completed);
        if (!completed) throw std::runtime_error("disk completion missing");
        io.restart();
        completed = false;
        char payload[100]{};
        disk.async_write(lt::storage_index_t{1}, {lt::piece_index_t{0}, 0, 100}, payload, {},
            [&](lt::storage_error const& error) {
                completed = true;
                if (bool(error.ec) != (count != 100))
                    throw std::runtime_error("missing or partial cache write was acknowledged");
            });
        if (completed) throw std::runtime_error("reentrant write completion");
        run(completed);
        if (!completed) throw std::runtime_error("write completion missing");
        io.restart();
    }
    lifecycle = true;
    char payload[100]; std::memset(payload, 0x3c, sizeof(payload));
    bool wrote = false, hashed = false, cleared = false, deleted = false;
    disk.async_write(storage, {lt::piece_index_t{0}, 0, 100}, payload, {},
        [&](lt::storage_error const& error) {
            if (error.ec) throw std::runtime_error("owned write failed");
            wrote = true;
        });
    {
        std::unique_lock<std::mutex> lock(fixture_mu);
        if (!fixture_cv.wait_for(lock, std::chrono::seconds(5), [] { return entered; }))
            throw std::runtime_error("write did not enter worker");
    }
    // The worker has not consumed the caller's bytes yet. Mutation after return
    // must not affect the write, or the SHA-1 queued behind it for this piece.
    std::memset(payload, 0xee, sizeof(payload));
    std::array<char, 100> expected; expected.fill(0x3c);
    lt::hasher expected_hash(lt::span<char const>(expected.data(), expected.size()));
    auto const digest = expected_hash.final();
    disk.async_hash(storage, lt::piece_index_t{0}, {}, {},
        [&](lt::piece_index_t, lt::sha1_hash const& hash, lt::storage_error const& error) {
            if (error.ec || hash != digest) throw std::runtime_error("hash overtook write or caller bytes leaked");
            hashed = true;
        });
    disk.async_clear_piece(storage, lt::piece_index_t{0}, [&](lt::piece_index_t) { cleared = true; });
    disk.async_delete_files(storage, {}, [&](lt::storage_error const& error) {
        if (error.ec) throw std::runtime_error("delete failed");
        deleted = true;
    });
    storage.reset();
    {
        std::lock_guard<std::mutex> lock(fixture_mu);
        if (!events.empty()) throw std::runtime_error("remove/delete overtook pending write");
        released = true;
    }
    fixture_cv.notify_all();
    disk.abort(true); // joins disk work while network completions still retain storage
    run(deleted);
    io.restart(); io.poll();
    if (!wrote || !hashed || !cleared || events != std::vector<std::string>{"write", "read", "clear", "delete", "close"})
        throw std::runtime_error("piece/lifecycle ordering or deferred storage close failed");
    std::cout << "Short I/O, owned buffers, write/hash/clear/delete/remove/abort ordering passed\n";
}
