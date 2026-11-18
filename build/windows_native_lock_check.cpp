// Check first use of the registry lock from concurrent readers and a writer.
#include "lt_shared_mutex.hpp"
#include <array>
#include <atomic>
#include <mutex>
#include <thread>

int main() {
    constexpr int attempts = 512;
    constexpr int readers = 4;
    std::array<flow_shared_mutex, attempts> locks;
    std::array<int, attempts> values{};
    std::atomic<int> arrivals{0};
    std::atomic<bool> valid{true};
    auto worker = [&](bool write) {
        for (int i = 0; i < attempts; ++i) {
            ++arrivals;
            while (arrivals.load() < (i + 1) * (readers + 1)) std::this_thread::yield();
            if (write) {
                std::unique_lock<flow_shared_mutex> lock(locks[i]);
                ++values[i];
            } else {
                std::shared_lock<flow_shared_mutex> lock(locks[i]);
                if (values[i] != 0 && values[i] != 1) valid = false;
            }
        }
    };
    std::array<std::thread, readers> workers;
    for (auto& thread : workers) thread = std::thread(worker, false);
    worker(true);
    for (auto& thread : workers) thread.join();
    for (auto value : values) if (value != 1) return 1;
    return valid ? 0 : 1;
}
