// Exercise the production disk upload completion with controlled cache reads.
// Including the implementation keeps this fixture outside the public shim API.
#include <libtorrent/aux_/session_settings.hpp>
#include <libtorrent/performance_counters.hpp>
#include "../server/lt/lt_disk_io.cpp"
#include <iostream>
#include <stdexcept>

namespace {
int supplied;
int read_fixture(int64_t, int, int64_t, uint8_t* buffer, int length) {
    if (supplied > 0) std::memset(buffer, 0x5a, std::min(supplied, length));
    return supplied;
}
}

int main() {
    lt::io_context io;
    lt::aux::session_settings settings;
    lt::counters counters;
    tsl_storage_callbacks callbacks{};
    callbacks.read = read_fixture;
    tsl_disk_io disk(io, settings, counters, callbacks);
    for (int count : {-1, 0, 99, 100}) {
        supplied = count;
        bool completed = false;
        disk.async_read(lt::storage_index_t{0}, {lt::piece_index_t{0}, 0, 100},
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
        io.run();
        if (!completed) throw std::runtime_error("disk completion missing");
        io.restart();
    }
    std::cout << "Missing, partial and complete native upload reads passed\n";
}
