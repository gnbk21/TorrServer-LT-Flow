package settings

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestMetadataDefaultsValidationAndNormalization(t *testing.T) {
	metadata := SettingsMetadata()
	base := NewDefaultConfig()
	if metadata.SchemaVersion != 1 || len(metadata.Fields) < 100 {
		t.Fatal("incomplete settings inventory")
	}
	for path, m := range metadata.Fields {
		v := settingValue(reflect.ValueOf(base), path)
		if !reflect.DeepEqual(m.Default, v.Interface()) {
			t.Fatalf("default drift: %s", path)
		}
		if m.Type != "integer" || m.Max == nil {
			continue
		}
		next := CloneSettings(base)
		target := settingValue(reflect.ValueOf(next), path)
		if target.OverflowInt(*m.Max + 1) {
			// This architecture's JSON decoder rejects values beyond the field
			// type before ValidateSettings can see them; do not wrap via SetInt.
			document := map[string]any{}
			node := document
			parts := strings.Split(path, ".")
			for _, part := range parts[:len(parts)-1] {
				child := map[string]any{}
				node[part], node = child, child
			}
			node[parts[len(parts)-1]] = *m.Max + 1
			input, _ := json.Marshal(document)
			if json.Unmarshal(input, next) == nil {
				t.Fatalf("wire type accepted an unrepresentable value: %s", path)
			}
			continue
		}
		target.SetInt(*m.Max + 1)
		if m.InvalidPolicy == "reject" && ValidateSettings(next) == nil {
			t.Fatalf("unvalidated bound: %s", path)
		}
		if strings.HasPrefix(path, "Flow.") {
			next.Flow.Normalize()
			if settingValue(reflect.ValueOf(next), path).Int() > *m.Max {
				t.Fatalf("normalization drift: %s", path)
			}
		}
	}
	for _, value := range []int{0, 64, 256, 1024} {
		f := DefaultFlowSettings()
		f.StreamTransportBufferKiB = value
		f.Normalize()
		if f.StreamTransportBufferKiB != value {
			t.Fatal(value)
		}
	}
	f := DefaultFlowSettings()
	f.StreamTransportBufferKiB = 65
	f.Normalize()
	if f.StreamTransportBufferKiB != 1024 {
		t.Fatal("invalid discrete value survived")
	}
}

func TestMetadataDistinguishesEditorRangesFromLegacyAPIPolicy(t *testing.T) {
	metadata := SettingsMetadata()
	for _, path := range []string{"DHTConnectionsLimit", "SslPort", "TorrentDisconnectTimeout", "RetrackersMode"} {
		if metadata.Fields[path].InvalidPolicy != "legacy" {
			t.Fatal(path)
		}
	}
	old := NewDefaultConfig()
	old.DHTConnectionsLimit = 0
	old.TorrentDisconnectTimeout = 0
	if err := ValidateSettings(old); err != nil {
		t.Fatal("metadata tightened existing API", err)
	}
	if metadata.Fields["CacheSize"].InvalidPolicy != "reject" || metadata.Fields["Flow.ProbeGraceMs"].InvalidPolicy != "normalize" {
		t.Fatal("enforced policies misreported")
	}
}
func TestOldConfigurationsRemainCompatibleWithMetadata(t *testing.T) {
	for _, input := range []string{`{"CacheSize":0}`, `{"Flow":{"SecurityProfile":"restricted","PreparationConcurrency":0,"PlaybackTokenTTL":0}}`} {
		var old BTSets
		if err := json.Unmarshal([]byte(input), &old); err != nil {
			t.Fatal(err)
		}
		if err := ValidateSettings(&old); err != nil {
			t.Fatal(err)
		}
		n := NormalizeConfiguration(&old)
		if n.Flow == nil || n.Flow.SwarmProfile != "legacy" || n.Flow.IncidentFlightRecorder {
			t.Fatal("migration changed defaults")
		}
	}
	if !SettingsMetadata().Fields["Flow.SecurityProfile"].Deprecated {
		t.Fatal("deprecated field hidden from API")
	}
}
