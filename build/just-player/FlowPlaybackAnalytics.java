package com.brouken.player;

import android.content.Context;
import android.content.SharedPreferences;
import android.os.Handler;
import android.os.SystemClock;
import android.util.AtomicFile;
import android.util.Log;
import androidx.media3.common.C;
import androidx.media3.common.MediaItem;
import androidx.media3.common.Timeline;
import androidx.preference.PreferenceManager;
import androidx.media3.exoplayer.ExoPlayer;
import androidx.media3.exoplayer.analytics.AnalyticsListener;
import androidx.media3.exoplayer.analytics.PlaybackStats;
import androidx.media3.exoplayer.analytics.PlaybackStatsListener;
import androidx.media3.exoplayer.source.LoadEventInfo;
import androidx.media3.exoplayer.source.MediaLoadData;
import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import org.json.JSONArray;
import org.json.JSONObject;

// Explicitly enabled, app-private, numeric diagnostics. Media3 keeps no event
// history here. The separately bounded export contains no media URL/name,
// credentials, peer identity, exception text or upload destination.
final class FlowPlaybackAnalytics implements AnalyticsListener {
    static final String FILE = "flow-player-diagnostics.json";
    static final int MAX_BYTES = 262144;
    private final ExoPlayer player;
    private final Handler handler;
    private final Context context;
    private final PlaybackStatsListener stats = new PlaybackStatsListener(false, null);
    private JSONArray samples = new JSONArray();
    private long started = SystemClock.elapsedRealtime();
    private final long generation;
    private final SharedPreferences preferences;
    private final SharedPreferences.OnSharedPreferenceChangeListener preferenceListener;
    private final ThreadPoolExecutor writer = new ThreadPoolExecutor(1, 1, 0, TimeUnit.SECONDS,
            new ArrayBlockingQueue<>(1), new ThreadPoolExecutor.DiscardOldestPolicy());
    private String correlation = "";
    private long bandwidth;
    private boolean released;
    private final Runnable sample = new Runnable() {
        @Override public void run() {
            if (released) return;
            if (!authorized()) { stop(); return; }
            snapshot();
            handler.postDelayed(this, 5000);
        }
    };
    private FlowPlaybackAnalytics(Context context, ExoPlayer player) {
        this.context = context.getApplicationContext();
        this.player = player;
        handler = new Handler(player.getApplicationLooper());
        synchronized (FlowDiagnosticsSettings.STORE_LOCK) {
            generation = ++FlowDiagnosticsSettings.generation;
        }
        preferences = PreferenceManager.getDefaultSharedPreferences(this.context);
        preferenceListener = (store, key) -> {
            if ("flow_diagnostics".equals(key) && !FlowDiagnosticsSettings.enabled(this.context))
                handler.post(this::stop);
        };
        preferences.registerOnSharedPreferenceChangeListener(preferenceListener);
        player.addAnalyticsListener(stats); // before loading media, while IDLE
        handler.post(sample);
    }
    static void attach(Context context, ExoPlayer player) {
        player.addAnalyticsListener(new FlowPlaybackAnalytics(context, player));
    }
    private void snapshot() {
        if (!authorized()) return;
        try {
            PlaybackStats current = stats.getPlaybackStats();
            if (current == null) return; // unknown until Media3 creates the current session
            JSONObject row = new JSONObject();
            row.put("elapsed_ms", SystemClock.elapsedRealtime()-started);
            row.put("recorded_at_ms", System.currentTimeMillis());
            long join = current.getMeanJoinTimeMs();
            row.put("startup_ms", join == C.TIME_UNSET ? JSONObject.NULL : join);
            row.put("rebuffer_events", current.totalRebufferCount);
            row.put("rebuffer_ms", current.getTotalRebufferTimeMs());
            row.put("seek_events", current.totalSeekCount);
            row.put("seek_buffer_ms", current.getTotalSeekTimeMs());
            row.put("paused_ms", current.getTotalPausedTimeMs());
            row.put("play_ms", current.getTotalPlayTimeMs());
            row.put("buffered_ms", Math.max(0, player.getTotalBufferedDuration()));
            row.put("dropped_frames", current.totalDroppedFrames);
            row.put("bandwidth_bps", Math.max(0, bandwidth));
            row.put("state", player.getPlaybackState());
            row.put("playing", player.isPlaying());
            row.put("session_id", correlation.isEmpty() ? JSONObject.NULL : correlation);
            if (samples.length() >= 128) samples.remove(0);
            samples.put(row);
            JSONObject report = new JSONObject();
            report.put("schema_version", 1);
            report.put("source", "flow-player-media3");
            report.put("clock", "player-elapsed-and-wall; server-clock-alignment-unknown");
            report.put("samples", samples);
            byte[] bytes = report.toString().getBytes(StandardCharsets.UTF_8);
            if (bytes.length > MAX_BYTES) return;
            writer.execute(() -> save(bytes));
            Log.i("FlowPlayback", "elapsed_ms="+row.getLong("elapsed_ms")
                    +" rebuffer_events="+current.totalRebufferCount
                    +" rebuffer_ms="+current.getTotalRebufferTimeMs()
                    +" seek_buffer_ms="+current.getTotalSeekTimeMs());
        } catch (Exception error) {
            Log.w("FlowPlayback", "diagnostic_sample_unavailable=1");
        }
    }
    private void save(byte[] bytes) {
        synchronized (FlowDiagnosticsSettings.STORE_LOCK) {
        if (!authorized()) return;
        AtomicFile file = new AtomicFile(new File(context.getFilesDir(), FILE));
        FileOutputStream output = null;
        try {
            output = file.startWrite(); output.write(bytes); file.finishWrite(output);
        } catch (Exception error) {
            if (output != null) file.failWrite(output);
            Log.w("FlowPlayback", "diagnostic_write_unavailable=1");
        }
        }
    }
    private boolean authorized() {
        synchronized (FlowDiagnosticsSettings.STORE_LOCK) {
            return generation == FlowDiagnosticsSettings.generation && FlowDiagnosticsSettings.enabled(context);
        }
    }
    private void stop() {
        stop(true);
    }
    private void stop(boolean discardPending) {
        if (released) return;
        released = true;
        handler.removeCallbacks(sample);
        preferences.unregisterOnSharedPreferenceChangeListener(preferenceListener);
        player.removeAnalyticsListener(stats);
        player.removeAnalyticsListener(this);
        if (discardPending) writer.shutdownNow(); else writer.shutdown();
    }
    @Override public void onMediaItemTransition(EventTime time, MediaItem item, int reason) {
        samples = new JSONArray(); correlation = ""; bandwidth = 0;
        started = SystemClock.elapsedRealtime();
    }
    private void correlate(EventTime time, LoadEventInfo info) {
        if (!authorized() || time.timeline.isEmpty() || player.getCurrentTimeline().isEmpty()) return;
        // A cancelled load from an earlier episode must never identify the new one.
        Object loaded = time.timeline.getWindow(time.windowIndex, new Timeline.Window()).uid;
        Object active = player.getCurrentTimeline().getWindow(player.getCurrentMediaItemIndex(), new Timeline.Window()).uid;
        if (!loaded.equals(active)) return;
        for (Map.Entry<String, List<String>> header : info.responseHeaders.entrySet()) {
            if (!"X-Flow-Session".equalsIgnoreCase(header.getKey()) || header.getValue().isEmpty()) continue;
            String value = header.getValue().get(0);
            if (value.matches("[A-Za-z0-9_-]{16,64}")) correlation = value;
        }
    }
    @Override public void onLoadCompleted(EventTime time, LoadEventInfo info, MediaLoadData data) { correlate(time, info); }
    @Override public void onLoadCanceled(EventTime time, LoadEventInfo info, MediaLoadData data) { correlate(time, info); }
    @Override public void onLoadError(EventTime time, LoadEventInfo info, MediaLoadData data, IOException error, boolean cancelled) { correlate(time, info); }
    @Override public void onBandwidthEstimate(EventTime time, int elapsedMs, long bytes, long estimate) { bandwidth = estimate; }
    @Override public void onPlayerReleased(EventTime time) {
        snapshot(); stop(false);
    }
}
