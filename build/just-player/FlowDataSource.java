package com.brouken.player;

import android.net.Uri;
import androidx.media3.datasource.DataSource;
import androidx.media3.datasource.DataSpec;
import androidx.media3.datasource.TransferListener;
import java.io.IOException;
import java.util.List;
import java.util.Map;

// Observe response headers immediately after open, even for an episode-length
// progressive request. Delegate all transport, credentials and byte handling.
final class FlowDataSource implements DataSource {
    private final DataSource delegate;
    private final FlowPlaybackAnalytics.Binding binding;
    private FlowDataSource(DataSource delegate, FlowPlaybackAnalytics.Binding binding) {
        this.delegate = delegate; this.binding = binding;
    }
    static Factory factory(Factory delegate, FlowPlaybackAnalytics.Binding binding) {
        return () -> new FlowDataSource(delegate.createDataSource(), binding);
    }
    @Override public long open(DataSpec spec) throws IOException {
        long length = delegate.open(spec);
        try { binding.opened(spec.uri, delegate.getResponseHeaders()); }
        catch (RuntimeException ignored) { /* Diagnostic failure must not interrupt media I/O. */ }
        return length;
    }
    @Override public int read(byte[] buffer, int offset, int length) throws IOException { return delegate.read(buffer, offset, length); }
    @Override public Uri getUri() { return delegate.getUri(); }
    @Override public Map<String, List<String>> getResponseHeaders() { return delegate.getResponseHeaders(); }
    @Override public void addTransferListener(TransferListener listener) { delegate.addTransferListener(listener); }
    @Override public void close() throws IOException { delegate.close(); }
}
