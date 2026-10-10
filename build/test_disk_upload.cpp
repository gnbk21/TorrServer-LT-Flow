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
bool hybrid_fixture=false;
std::vector<char> hybrid_bytes;
std::vector<int64_t> geometry_pairs;
int read_fixture(int64_t, int piece, int64_t offset, uint8_t* buffer, int length) {
    if (hybrid_fixture) {
        auto const start=int64_t(piece)*32768+offset;
        if (start<0 || start+length>int64_t(hybrid_bytes.size())) return -1;
        std::memcpy(buffer,hybrid_bytes.data()+start,length);return length;
    }
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
    callbacks.close = [](int64_t storage) {
        if (lifecycle && storage==1) { std::lock_guard<std::mutex> lock(fixture_mu); events.push_back("close"); }
    };
    callbacks.clear_piece = [](int64_t, int) {
        std::lock_guard<std::mutex> lock(fixture_mu); events.push_back("clear");
    };
    callbacks.deleted = [](int64_t) {
        std::lock_guard<std::mutex> lock(fixture_mu); events.push_back("delete");
    };
    callbacks.geometry=[](int64_t, int64_t const* pairs, int count) {
        geometry_pairs.assign(pairs,pairs+2*count);
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
                if (error.ec && error.ec != boost::system::errc::make_error_code(boost::system::errc::io_error))
                    throw std::runtime_error("read failure used a platform system error number");
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
                if (error.ec && error.ec != boost::system::errc::make_error_code(boost::system::errc::io_error))
                    throw std::runtime_error("write failure used a platform system error number");
            });
        if (completed) throw std::runtime_error("reentrant write completion");
        run(completed);
        if (!completed) throw std::runtime_error("write completion missing");
        io.restart();
    }
    bool block_hashed = false;
    std::array<char,100> block; block.fill(0x5a);
    auto const block_digest=lt::hasher256(lt::span<char const>(block.data(),block.size())).final();
    disk.async_hash2(storage, lt::piece_index_t{0}, 0, {},
        [&](lt::piece_index_t, lt::sha256_hash const& hash, lt::storage_error const& error) {
            if (error.ec || hash!=block_digest) throw std::runtime_error("short BEP 52 block hash changed bytes");
            block_hashed = true;
        });
    if (block_hashed) throw std::runtime_error("reentrant block-hash completion");
    run(block_hashed);
    io.restart();
    std::array<lt::sha256_hash,1> block_hashes;
    bool both_hashed=false;
    disk.async_hash(storage,lt::piece_index_t{0},block_hashes,lt::disk_interface::v1_hash,
        [&](lt::piece_index_t,lt::sha1_hash const& hash,lt::storage_error const& error) {
            auto const expected=lt::hasher(lt::span<char const>(block.data(),block.size())).final();
            if (error.ec || hash!=expected || block_hashes[0]!=block_digest) throw std::runtime_error("hybrid hashes disagree");
            both_hashed=true;
        });
    run(both_hashed);
    io.restart();
    for (int offset : {-1,1,16384}) {
        bool rejected=false;
        disk.async_hash2(storage,lt::piece_index_t{0},offset,{},
            [&](lt::piece_index_t,lt::sha256_hash const&,lt::storage_error const& error) {
                if (!error.ec) throw std::runtime_error("invalid v2 block offset accepted");
                rejected=true;
            });
        run(rejected);io.restart();
    }
    supplied=99;
    bool short_hash_rejected=false;
    disk.async_hash2(storage,lt::piece_index_t{0},0,{},
        [&](lt::piece_index_t,lt::sha256_hash const&,lt::storage_error const& error) {
            if (!error.ec) throw std::runtime_error("short v2 read was trusted");
            short_hash_rejected=true;
        });
    run(short_hash_rejected);
    io.restart();
    {
        hybrid_fixture=true;
        lt::sha256_hash root;
        lt::file_storage hybrid_files;
        hybrid_files.set_piece_length(32768);
        hybrid_files.add_file("hybrid/a.mkv",17001,{},0,{},root.data());
        hybrid_files.add_file("hybrid/b.mkv",10001,{},0,{},root.data());
        hybrid_files.set_num_pieces(2);
        lt::storage_params hybrid_params(hybrid_files,renamed,"","",lt::storage_mode_sparse,priorities,lt::sha1_hash{},true,true);
        auto hybrid=disk.new_torrent(hybrid_params,{});
        if (geometry_pairs!=std::vector<int64_t>{0,17001,1,10001})
            throw std::runtime_error("short per-file BEP 52 geometry was lost");
        hybrid_bytes.assign(65536,0);
        for (int i=0;i<17001;++i) hybrid_bytes[i]=char(i%251);
        for (int i=0;i<10001;++i) hybrid_bytes[32768+i]=char((i+7)%251);
        std::array<lt::sha256_hash,2> hashes;
        bool finished=false;
        disk.async_hash(hybrid,lt::piece_index_t{0},hashes,lt::disk_interface::v1_hash,
            [&](lt::piece_index_t,lt::sha1_hash const& digest,lt::storage_error const& error) {
                if (error.ec || digest!=lt::hasher(lt::span<char const>(hybrid_bytes.data(),32768)).final()
                    || hashes[0]!=lt::hasher256(lt::span<char const>(hybrid_bytes.data(),16384)).final()
                    || hashes[1]!=lt::hasher256(lt::span<char const>(hybrid_bytes.data()+16384,617)).final())
                    throw std::runtime_error("hybrid SHA-1 padding or SHA-256 short leaves were incorrect");
                finished=true;
            });
        run(finished);io.restart();finished=false;
        disk.async_hash2(hybrid,lt::piece_index_t{1},0,{},
            [&](lt::piece_index_t,lt::sha256_hash const& digest,lt::storage_error const& error) {
                if (error.ec || digest!=lt::hasher256(lt::span<char const>(hybrid_bytes.data()+32768,10001)).final())
                    throw std::runtime_error("second-file short leaf included padding");
                finished=true;
            });
        run(finished);io.restart();
        hybrid.reset();
        hybrid_fixture=false;
    }
    lifecycle = true;
    char payload[100]; std::memset(payload, 0x3c, sizeof(payload));
    bool wrote = false, hashed = false, cleared = false, rewrote = false, deleted = false;
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
    disk.async_hash(storage, lt::piece_index_t{0}, {}, lt::disk_interface::v1_hash,
        [&](lt::piece_index_t, lt::sha1_hash const& hash, lt::storage_error const& error) {
            if (error.ec || hash != digest) throw std::runtime_error("hash overtook write or caller bytes leaked");
            hashed = true;
        });
    disk.async_clear_piece(storage, lt::piece_index_t{0}, [&](lt::piece_index_t) { cleared = true; });
    disk.async_write(storage, {lt::piece_index_t{0}, 0, 100}, expected.data(), {},
        [&](lt::storage_error const& error) {
            if (error.ec) throw std::runtime_error("write after clear failed");
            rewrote = true;
        });
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
    if (!wrote || !hashed || !cleared || !rewrote
        || events != std::vector<std::string>{"write", "read", "clear", "write", "delete", "close"})
        throw std::runtime_error("piece/lifecycle ordering or deferred storage close failed");
    bool late_read = false;
    {
        tsl_disk_io replacement(io, settings, counters, callbacks);
        auto next = replacement.new_torrent(params, {});
        if (storage_id_of(next) <= 1) throw std::runtime_error("replacement session reused a retiring storage ID");
        replacement.async_read(next, {lt::piece_index_t{0},0,100},
            [&](lt::disk_buffer_holder buffer, lt::storage_error const& error) {
                if (error.ec || uint8_t(buffer.data()[0]) != 0x3c) throw std::runtime_error("late read lost its storage");
                late_read = true;
            });
        next.reset();
        replacement.abort(true);
    }
    // The queued completion frees its read buffer after replacement destruction.
    run(late_read);
    std::cout << "Short I/O, owned buffers, write/hash/clear/delete/remove/abort ordering passed\n";
}
