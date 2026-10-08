// No libtorrent dependency: exercise ordering, bounded admission and lifecycle.
#include "../server/lt/lt_io_queue.hpp"

#include <iostream>
#include <stdexcept>

void require(bool value) { if (!value) throw std::runtime_error("queue invariant failed"); }

int main() {
    flow_io_queue queue({});
    std::promise<void> entered, release;
    auto ready = release.get_future().share();
    bool throttle = false;
    require(queue.submit(1, 1, flow_io_queue::high_bytes, [&] {
        entered.set_value(); ready.wait();
    }, throttle));
    entered.get_future().wait();
    require(throttle);
    require(queue.status().bytes == flow_io_queue::high_bytes);
    std::atomic<int> sequence{0};
    bool woke = false;
    require(queue.submit(1, 1, flow_io_queue::limit_bytes-flow_io_queue::high_bytes,
        [&] { require(sequence.fetch_add(1) == 0); }, throttle, [&] { woke = true; }));
    require(throttle);
    require(!queue.submit(1, 1, 1, [] {}, throttle));
    require(queue.status().rejected == 1);
    require(queue.submit(1, 1, 0, [&] { require(sequence.fetch_add(1) == 1); }, throttle));
    // Lifecycle completion cannot overtake writes/hashes/clear in any lane.
    std::promise<void> fenced;
    queue.fence([&] { require(sequence == 2); fenced.set_value(); });
    release.set_value();
    fenced.get_future().wait();
    queue.stop();
    require(woke && queue.status().bytes == 0 && queue.status().jobs == 0);
    require(!queue.submit(1, 1, 0, [] {}, throttle));
    bool stopped_fence = false;
    queue.fence([&] { stopped_fence = queue.status().jobs == 0; });
    require(stopped_fence);
    std::cout << "Ordered work, admission, backpressure and lifecycle passed\n";
    flow_io_queue stopping({});
    std::promise<void> busy, finish;
    auto finish_ready = finish.get_future().share();
    require(stopping.submit(2, 2, 1, [&] { busy.set_value(); finish_ready.wait(); }, throttle));
    busy.get_future().wait();
    auto stop = std::async(std::launch::async, [&] { stopping.stop(); });
    // Observe closed admission before testing the concurrent lifecycle fence.
    while (!stopping.status().closed) std::this_thread::yield();
    auto fence = std::async(std::launch::async, [&] { stopping.fence([] {}); });
    require(fence.wait_for(std::chrono::milliseconds(20)) == std::future_status::timeout);
    finish.set_value();
    stop.get(); fence.get();
    require(stopping.status().jobs == 0);

    flow_io_queue saturated({});
    std::promise<void> blocked, unblock, cleared;
    auto unblock_ready = unblock.get_future().share();
    require(saturated.submit(0, 1, 0, [&] { blocked.set_value(); unblock_ready.wait(); }, throttle));
    blocked.get_future().wait();
    for (int i = 1; i < 8192; ++i)
        require(saturated.submit(0, 1, 0, [] {}, throttle));
    require(!saturated.submit(0, 0, 0, [] {}, throttle));
    require(!saturated.submit(0, 0, 1, [] {}, throttle, {}, true));
    // A clear in another lane must not wait for this unrelated blocked piece.
    // Its reserved admission is only for zero-byte maintenance, not payloads.
    require(saturated.submit(0, 0, 0, [&] { cleared.set_value(); }, throttle, {}, true));
    auto clear_ready = cleared.get_future();
    bool const timely = clear_ready.wait_for(std::chrono::seconds(5)) == std::future_status::ready;
    unblock.set_value();
    saturated.stop();
    require(timely && saturated.status().jobs == 0 && saturated.status().bytes == 0);
}
