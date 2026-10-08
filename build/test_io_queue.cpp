// No libtorrent dependency: exercise ordering, bounded admission and lifecycle.
#include "../server/lt/lt_io_queue.hpp"
#include <cassert>
#include <iostream>

int main() {
    flow_io_queue queue({});
    std::promise<void> entered, release;
    auto ready = release.get_future().share();
    bool throttle = false;
    assert(queue.submit(1, 1, flow_io_queue::high_bytes, [&] {
        entered.set_value(); ready.wait();
    }, throttle));
    entered.get_future().wait();
    assert(throttle);
    assert(queue.status().bytes == flow_io_queue::high_bytes);
    std::atomic<int> sequence{0};
    bool woke = false;
    assert(queue.submit(1, 1, flow_io_queue::limit_bytes-flow_io_queue::high_bytes,
        [&] { assert(sequence.fetch_add(1) == 0); }, throttle, [&] { woke = true; }));
    assert(throttle);
    assert(!queue.submit(1, 1, 1, [] {}, throttle));
    assert(queue.status().rejected == 1);
    assert(queue.submit(1, 1, 0, [&] { assert(sequence.fetch_add(1) == 1); }, throttle));
    // Lifecycle completion cannot overtake writes/hashes/clear in any lane.
    std::promise<void> fenced;
    queue.fence([&] { assert(sequence == 2); fenced.set_value(); });
    release.set_value();
    fenced.get_future().wait();
    queue.stop();
    assert(woke && queue.status().bytes == 0 && queue.status().jobs == 0);
    assert(!queue.submit(1, 1, 0, [] {}, throttle));
    bool stopped_fence = false;
    queue.fence([&] { stopped_fence = queue.status().jobs == 0; });
    assert(stopped_fence);
    std::cout << "Ordered work, admission, backpressure and lifecycle passed\n";
}
