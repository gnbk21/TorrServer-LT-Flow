// Bounded piece-ordered work for the custom disk backend. No libtorrent types
// live here: completions and disk-observer notifications are posted by callers.
#pragma once
#include <algorithm>
#include <array>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <deque>
#include <functional>
#include <future>
#include <memory>
#include <mutex>
#include <thread>
#include <vector>

class flow_io_queue {
public:
    static constexpr std::size_t high_bytes = 16 * 1024 * 1024;
    static constexpr std::size_t low_bytes = high_bytes / 2;
    static constexpr std::size_t limit_bytes = 64 * 1024 * 1024;
    using clock = std::chrono::steady_clock;
    using metric = std::function<void(std::uint64_t, std::uint64_t)>;

    explicit flow_io_queue(metric record) : record_(std::move(record)) {
        try {
            for (std::size_t i = 0; i < lanes_.size(); ++i)
                threads_.emplace_back([this, i] { worker(i); });
        } catch (...) { stop(); throw; }
    }
    ~flow_io_queue() { stop(); }

    // Every operation for a (storage,piece) has FIFO ordering, including hashes
    // and clears. Different pieces may run concurrently. A rejected operation
    // must complete with a storage error; callers must never acknowledge it.
    bool submit(std::int64_t storage, int piece, std::size_t bytes,
        std::function<void()> work, bool& throttle, std::function<void()> wake = {},
        bool maintenance = false) {
        std::lock_guard<std::mutex> lock(mu_);
        // Zero-byte piece maintenance must keep its FIFO position even under
        // load. A global fence does not order it against later piece writes.
        if (stopped_ || bytes > limit_bytes - bytes_
            || (jobs_ >= 8192 && !(maintenance && bytes == 0))) {
            ++rejected_;
            return false;
        }
        bytes_ += bytes;
        ++jobs_;
        peak_ = std::max(peak_, bytes_);
        throttle = bytes_ >= high_bytes || jobs_ >= 4096;
        if (throttle && wake) observers_.push_back(std::move(wake));
        auto const lane = (std::uint64_t(storage) * 31 + std::uint32_t(piece)) % lanes_.size();
        lanes_[lane].push_back({std::move(work), bytes, clock::now()});
        cv_.notify_all();
        return true;
    }

    // A storage-wide fence runs after all previously submitted operations.
    // Lifecycle fences reserve no byte buffers and are never rejected by load.
    void fence(std::function<void()> done) {
        auto left = std::make_shared<std::atomic<unsigned>>(lanes_.size());
        std::unique_lock<std::mutex> lock(mu_);
        if (stopped_) {
            // Admission may be closed while a concurrent stop is still joining
            // workers. A lifecycle completion must not overtake their last job.
            drained_.wait(lock, [this] { return jobs_ == 0; });
            lock.unlock(); done(); return;
        }
        for (auto& lane : lanes_) {
            ++jobs_;
            lane.push_back({[left, done] { if (left->fetch_sub(1) == 1) done(); }, 0, clock::now()});
        }
        cv_.notify_all();
    }

    void drain() {
        std::promise<void> promise;
        auto ready = promise.get_future();
        fence([&promise] { promise.set_value(); });
        ready.wait();
    }

    void stop() {
        std::lock_guard<std::mutex> joining(stop_mu_);
        {
            std::lock_guard<std::mutex> lock(mu_);
            if (stopped_) return;
            stopped_ = true;
        }
        cv_.notify_all();
        for (auto& thread : threads_) if (thread.joinable()) thread.join();
    }

    struct snapshot { std::size_t bytes, peak, jobs, rejected; bool closed; };
    snapshot status() const {
        std::lock_guard<std::mutex> lock(mu_);
        return {bytes_, peak_, jobs_, rejected_, stopped_};
    }

private:
    struct job { std::function<void()> work; std::size_t bytes; clock::time_point queued; };
    void worker(std::size_t i) {
        for (;;) {
            job next;
            {
                std::unique_lock<std::mutex> lock(mu_);
                cv_.wait(lock, [this, i] { return stopped_ || !lanes_[i].empty(); });
                if (lanes_[i].empty()) return;
                next = std::move(lanes_[i].front());
                lanes_[i].pop_front();
            }
            auto const start = clock::now();
            next.work();
            auto const end = clock::now();
            if (record_) record_(std::chrono::duration_cast<std::chrono::microseconds>(start-next.queued).count(),
                std::chrono::duration_cast<std::chrono::microseconds>(end-start).count());
            // Release owned buffers before capacity is returned to producers.
            next.work = {};
            std::vector<std::function<void()>> observers;
            {
                std::lock_guard<std::mutex> lock(mu_);
                bytes_ -= next.bytes;
                --jobs_;
                if (jobs_ == 0) drained_.notify_all();
                if (bytes_ < low_bytes && jobs_ < 2048) observers.swap(observers_);
            }
            for (auto& wake : observers) wake();
        }
    }
    metric record_;
    mutable std::mutex mu_;
    std::mutex stop_mu_;
    std::condition_variable cv_, drained_;
    std::array<std::deque<job>, 4> lanes_;
    std::vector<std::thread> threads_;
    std::vector<std::function<void()>> observers_;
    std::size_t bytes_ = 0, peak_ = 0, jobs_ = 0, rejected_ = 0;
    bool stopped_ = false;
};
