package com.brouken.player;

import android.os.Handler;
import android.os.SystemClock;
import android.util.Log;
import androidx.media3.common.PlaybackException;
import androidx.media3.common.MediaItem;
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
    private int stalls, dropped, errors, seeks;
    private boolean ready, released, seeking;
    private final Runnable sample = new Runnable() {
        @Override public void run() {
            if (released) return;
            long active = rebufferAt < 0 ? 0 : SystemClock.elapsedRealtime()-rebufferAt;
            Log.i("FlowPlayback", "elapsed_ms="+(SystemClock.elapsedRealtime()-started)
                    +" buffered_ms="+player.getTotalBufferedDuration()
                    +" state="+player.getPlaybackState()+" playing="+player.isPlaying()
                    +" rebuffer_events="+stalls+" rebuffer_ms="+(rebufferMs+active)
                    +" seeks="+seeks+" dropped_frames="+dropped+" errors="+errors+" bandwidth_bps="+bandwidth);
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
        if (state == Player.STATE_READY) { ready = true; seeking = false; }
    }
    @Override public void onPositionDiscontinuity(EventTime time, Player.PositionInfo oldPosition,
            Player.PositionInfo newPosition, int reason) {
        if (reason == Player.DISCONTINUITY_REASON_SEEK) { seeks++; seeking = true; }
    }
    @Override public void onMediaItemTransition(EventTime time, MediaItem item, int reason) {
        ready = false; seeking = false;
    }
    // Evaluate after the event batch so seek-induced BUFFERING cannot be
    // counted as a supply stall regardless of callback ordering.
    @Override public void onEvents(Player player, AnalyticsListener.Events events) {
        if (player.getPlaybackState() == Player.STATE_READY) { ready = true; seeking = false; }
        updateBuffering();
    }
    private void updateBuffering() {
        boolean buffering = ready && !seeking && player.getPlayWhenReady() && player.getPlaybackState() == Player.STATE_BUFFERING;
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
        if (!canceled) { errors++; Log.i("FlowPlayback", "load_error=1"); }
    }
    @Override public void onPlayerReleased(EventTime time) { released = true; handler.removeCallbacks(sample); }
}
