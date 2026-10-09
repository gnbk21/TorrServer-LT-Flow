package com.brouken.player;

// Pure policy, tested without Android. Native decoder/surface memory is outside
// the Java heap, so also respect system headroom and leave a heap reserve.
final class FlowBufferBudget {
    static int bytes(long maximumHeap, long usedHeap, long availableSystem, boolean lowRam) {
        final long mib = 1024L * 1024;
        long reserve = Math.max(64*mib, maximumHeap/4);
        long headroom = Math.max(0, maximumHeap-usedHeap-reserve);
        long budget = Math.min(384*mib, Math.min(maximumHeap/3, headroom/2));
        budget = Math.min(budget, Math.max(0, availableSystem)/16);
        if (lowRam) budget = Math.min(budget, 32*mib);
        // Small devices still need a usable allocator, but never a 1 GiB target.
        return (int)Math.max(1024*1024, budget);
    }
}
