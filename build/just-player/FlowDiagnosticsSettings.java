package com.brouken.player;

import android.content.Context;
import android.widget.Toast;
import android.util.AtomicFile;
import androidx.activity.result.ActivityResultLauncher;
import androidx.activity.result.contract.ActivityResultContracts;
import androidx.preference.Preference;
import androidx.preference.PreferenceCategory;
import androidx.preference.PreferenceFragmentCompat;
import androidx.preference.PreferenceManager;
import androidx.preference.SwitchPreferenceCompat;
import java.io.File;
import java.io.InputStream;
import java.io.OutputStream;
import java.io.ByteArrayOutputStream;

final class FlowDiagnosticsSettings {
    static final Object STORE_LOCK = new Object();
    static long generation;
    static boolean enabled(Context context) {
        return PreferenceManager.getDefaultSharedPreferences(context).getBoolean("flow_diagnostics", false);
    }
    static void install(PreferenceFragmentCompat fragment) {
        Context context = fragment.requireContext();
        PreferenceCategory category = new PreferenceCategory(context);
        category.setTitle(R.string.flow_diagnostics_title);
        fragment.getPreferenceScreen().addPreference(category);
        SwitchPreferenceCompat toggle = new SwitchPreferenceCompat(context);
        toggle.setKey("flow_diagnostics");
        toggle.setDefaultValue(false);
        toggle.setTitle(R.string.flow_diagnostics_enable);
        toggle.setSummary(R.string.flow_diagnostics_hint);
        category.addPreference(toggle);
        Preference export = new Preference(context);
        export.setTitle(R.string.flow_diagnostics_export);
        export.setSummary(R.string.flow_diagnostics_export_hint);
        category.addPreference(export);
        ActivityResultLauncher<String> chooser = fragment.registerForActivityResult(
                new ActivityResultContracts.CreateDocument("application/json"), uri -> {
            if (uri == null) return;
            new Thread(() -> {
                boolean success = false;
                try {
                    ByteArrayOutputStream snapshot = new ByteArrayOutputStream();
                    synchronized (STORE_LOCK) {
                        try (InputStream input = new AtomicFile(new File(context.getFilesDir(), FlowPlaybackAnalytics.FILE)).openRead()) {
                            byte[] bytes = new byte[8192]; int count;
                            while ((count = input.read(bytes)) != -1) {
                                if (snapshot.size()+count > FlowPlaybackAnalytics.MAX_BYTES) throw new IllegalStateException();
                                snapshot.write(bytes,0,count);
                            }
                        }
                    }
                    try (OutputStream output = context.getContentResolver().openOutputStream(uri, "wt")) {
                        if (output == null) throw new IllegalStateException();
                        snapshot.writeTo(output);
                    }
                    success = true;
                } catch (Exception error) { /* export failure contains no private exception text */ }
                final boolean completed = success;
                new android.os.Handler(context.getMainLooper()).post(() -> Toast.makeText(context,
                        completed ? R.string.flow_diagnostics_exported : R.string.flow_diagnostics_unavailable,
                        Toast.LENGTH_LONG).show());
            }, "FlowDiagnosticExport").start();
        });
        export.setOnPreferenceClickListener(preference -> { chooser.launch("Flow-Player-diagnostics.json"); return true; });
        toggle.setOnPreferenceChangeListener((preference, value) -> {
            if (Boolean.FALSE.equals(value)) synchronized (STORE_LOCK) {
                generation++;
                PreferenceManager.getDefaultSharedPreferences(context).edit().putBoolean("flow_diagnostics",false).apply();
                new AtomicFile(new File(context.getFilesDir(), FlowPlaybackAnalytics.FILE)).delete();
            }
            return true;
        });
    }
}
