package com.brouken.player;

import android.os.Handler;
import android.os.SystemClock;
import android.util.Log;
import androidx.media3.common.PlaybackException;
import androidx.media3.common.Player;
import androidx.media3.exoplayer.ExoPlayer;
import androidx.media3.exoplayer.analytics.AnalyticsListener;
import androidx.media3.exoplayer.source.LoadEventInfo;
import androidx.media3.exoplayer.source.MediaLoadData;
import java.io.IOException;

// Opt-in local numeric diagnostics. No URLs, filenames, headers, exception
// messages, credentials, persistent files or upload destination are recorded.
final class FlowPlaybackAnalytics implements AnalyticsListener {
    private final ExoPlayer player;
    private final Handler handler;
    private final long started = SystemClock.elapsedRealtime();
    private long rebufferAt = -1;
    private long rebufferMs;
    private long bandwidth;
    private int stalls, dropped, errors;
    private boolean ready, released;
    private final Runnable sample = new Runnable() {
        @Override public void run() {
            if (released) return;
            long active = rebufferAt < 0 ? 0 : SystemClock.elapsedRealtime()-rebufferAt;
            Log.i("FlowPlayback", "elapsed_ms="+(SystemClock.elapsedRealtime()-started)
                    +" buffered_ms="+player.getTotalBufferedDuration()
                    +" state="+player.getPlaybackState()+" playing="+player.isPlaying()
                    +" rebuffer_events="+stalls+" rebuffer_ms="+(rebufferMs+active)
                    +" dropped_frames="+dropped+" errors="+errors+" bandwidth_bps="+bandwidth);
            handler.postDelayed(this, 5000);
        }
    };
    private FlowPlaybackAnalytics(ExoPlayer player) {
        this.player = player;
        handler = new Handler(player.getApplicationLooper());
        handler.post(sample);
    }
    static void attach(ExoPlayer player) { player.addAnalyticsListener(new FlowPlaybackAnalytics(player)); }
    @Override public void onPlaybackStateChanged(EventTime time, int state) {
        if (state == Player.STATE_READY) ready = true;
        updateBuffering();
    }
    @Override public void onPlayWhenReadyChanged(EventTime time, boolean play, int reason) { updateBuffering(); }
    private void updateBuffering() {
        boolean buffering = ready && player.getPlayWhenReady() && player.getPlaybackState() == Player.STATE_BUFFERING;
        long now = SystemClock.elapsedRealtime();
        if (buffering && rebufferAt < 0) { stalls++; rebufferAt = now; }
        if (!buffering && rebufferAt >= 0) { rebufferMs += now-rebufferAt; rebufferAt = -1; }
    }
    @Override public void onBandwidthEstimate(EventTime time, int elapsedMs, long bytes, long estimate) { bandwidth = estimate; }
    @Override public void onDroppedVideoFrames(EventTime time, int count, long elapsedMs) { dropped += count; }
    @Override public void onPlayerError(EventTime time, PlaybackException error) {
        errors++; Log.i("FlowPlayback", "player_error_code="+error.errorCode);
    }
    @Override public void onLoadError(EventTime time, LoadEventInfo info, MediaLoadData data, IOException error, boolean canceled) {
        if (!canceled) { errors++; Log.i("FlowPlayback", "load_error_type="+error.getClass().getSimpleName()); }
    }
    @Override public void onPlayerReleased(EventTime time) { released = true; handler.removeCallbacks(sample); }
}
