package settings

import (
	"encoding/json"
	"testing"
)

func TestFlowSettingsMigrateNewFieldsWithoutOverwritingFlags(t *testing.T) {
	var f FlowSettings
	if err := json.Unmarshal([]byte(`{"Enabled":true,"AdaptiveStartup":true,"RangeTraceEnabled":false,"TargetBufferSeconds":60}`), &f); err != nil {
		t.Fatal(err)
	}
	if !f.AdaptiveReadAhead || f.TargetBufferSeconds != 60 || f.MaxBufferSeconds != 180 ||
		f.NetworkRetryMinSec != 2 || f.NetworkRetryMaxSec != 60 || f.SwarmProfile != "legacy" {
		t.Fatalf("new fields not migrated: %+v", f)
	}
	if f.RangeTraceEnabled {
		t.Fatal("explicit false changed")
	}
	var disabled FlowSettings
	if err := json.Unmarshal([]byte(`{"AdaptiveReadAhead":false}`), &disabled); err != nil {
		t.Fatal(err)
	}
	if disabled.AdaptiveReadAhead {
		t.Fatal("explicit disable changed")
	}
}

func TestFlowSwarmSettingsNormalize(t *testing.T) {
	f := DefaultFlowSettings()
	f.SwarmProfile = " Balanced "
	f.SwarmCustom.ConnectionSpeed = 900
	f.SwarmCustom.PeerConnectTimeout = 20
	f.Normalize()
	if f.SwarmProfile != "balanced" || f.SwarmCustom.ConnectionSpeed != 0 || f.SwarmCustom.PeerConnectTimeout != 20 {
		t.Fatalf("normalized swarm settings: %+v", f)
	}
}

func TestFlowHistoryAndDHTMigration(t *testing.T) {
	var migrated FlowSettings
	if err := json.Unmarshal([]byte(`{"Enabled":true}`), &migrated); err != nil {
		t.Fatal(err)
	}
	if migrated.DiagnosticHistory || !migrated.DHTStatePersistence {
		t.Fatal("incorrect new-field defaults")
	}
	var explicit FlowSettings
	if err := json.Unmarshal([]byte(`{"DiagnosticHistory":true,"DHTStatePersistence":false}`), &explicit); err != nil {
		t.Fatal(err)
	}
	explicit.Normalize()
	if !explicit.DiagnosticHistory || explicit.DHTStatePersistence {
		t.Fatal("explicit choices changed")
	}
}

func TestRateDeadlineExperimentMigration(t *testing.T) {
	for _, input := range []string{`{}`, `{"RateAwareDeadlines":false}`} {
		var f FlowSettings
		if err := json.Unmarshal([]byte(input), &f); err != nil {
			t.Fatal(err)
		}
		f.Normalize()
		if f.RateAwareDeadlines {
			t.Fatal("migration enabled an unproven deadline experiment")
		}
	}
	var enabled FlowSettings
	if err := json.Unmarshal([]byte(`{"RateAwareDeadlines":true}`), &enabled); err != nil {
		t.Fatal(err)
	}
	enabled.Normalize()
	if !enabled.RateAwareDeadlines {
		t.Fatal("explicit experiment choice lost")
	}
}
