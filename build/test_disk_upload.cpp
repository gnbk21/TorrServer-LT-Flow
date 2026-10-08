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
int read_fixture(int64_t, int, int64_t, uint8_t* buffer, int length) {
    if (supplied > 0) std::memset(buffer, 0x5a, std::min(supplied, length));
    return supplied;
}
int write_fixture(int64_t, int, int64_t, uint8_t const*, int) { return supplied; }
}

int main() {
    lt::io_context io;
    lt::aux::session_settings settings;
    lt::counters counters;
    tsl_storage_callbacks callbacks{};
    callbacks.read = read_fixture;
    callbacks.write = write_fixture;
    callbacks.open = [](int64_t, uint8_t const*, int, int64_t) {};
    callbacks.close = [](int64_t) {};
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
    std::cout << "Missing, partial and complete native reads/writes passed\n";
}
