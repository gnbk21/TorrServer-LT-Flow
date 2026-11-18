#pragma once

#include <shared_mutex>

#ifdef _WIN32
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <windows.h>

// The MinGW pthread rwlock can fail when several threads first acquire it.
// A statically initialized Windows reader/writer lock has no first-use race.
class flow_shared_mutex {
public:
    flow_shared_mutex() = default;
    flow_shared_mutex(flow_shared_mutex const&) = delete;
    flow_shared_mutex& operator=(flow_shared_mutex const&) = delete;

    void lock() noexcept { AcquireSRWLockExclusive(&lock_); }
    void unlock() noexcept { ReleaseSRWLockExclusive(&lock_); }
    void lock_shared() noexcept { AcquireSRWLockShared(&lock_); }
    void unlock_shared() noexcept { ReleaseSRWLockShared(&lock_); }

private:
    SRWLOCK lock_ = SRWLOCK_INIT;
};
#else
using flow_shared_mutex = std::shared_mutex;
#endif
