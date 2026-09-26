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
	if !f.AdaptiveReadAhead || f.TargetBufferSeconds != 60 || f.MaxBufferSeconds != 180 {
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
