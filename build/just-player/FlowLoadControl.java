package com.brouken.player;

import android.app.ActivityManager;
import android.content.Context;
import androidx.media3.exoplayer.DefaultLoadControl;

final class FlowLoadControl {
    static DefaultLoadControl create(Context context) {
        ActivityManager manager = (ActivityManager)context.getSystemService(Context.ACTIVITY_SERVICE);
        ActivityManager.MemoryInfo memory = new ActivityManager.MemoryInfo();
        manager.getMemoryInfo(memory);
        Runtime runtime = Runtime.getRuntime();
        int bytes = FlowBufferBudget.bytes(runtime.maxMemory(),
                runtime.totalMemory()-runtime.freeMemory(), memory.availMem,
                manager.isLowRamDevice() || memory.lowMemory);
        return new DefaultLoadControl.Builder()
                .setBufferDurationsMs(30000, 90000, 2500, 7500)
                .setTargetBufferBytes(bytes)
                // Time priority can exceed the byte budget at high bitrate.
                // Keep the byte cap authoritative even below thirty seconds.
                .setPrioritizeTimeOverSizeThresholds(false)
                .setBackBuffer(0, false)
                .build();
    }
}
