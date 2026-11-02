package rutor

import (
	"encoding/json"
	"server/rutor/models"
	"testing"
)

func TestDatabaseMetadataDoesNotInventSeedOrSizeObservations(t *testing.T) {
	for _, row := range []struct {
		input      string
		seed, size bool
	}{
		{`{"Title":"Unobserved"}`, false, false},
		{`{"Title":"Known zero peers","Seed":0,"Size":"1.2 GB"}`, true, true},
		{`{"Seed":null,"Size":null}`, false, false},
		{`{"Seed":4,"Size":"","SeedKnown":false}`, false, false},
	} {
		var value models.TorrentDetails
		if err := json.Unmarshal([]byte(row.input), &value); err != nil {
			t.Fatal(err)
		}
		if value.SeedKnown == nil || value.SizeKnown == nil || *value.SeedKnown != row.seed || *value.SizeKnown != row.size {
			t.Fatal(value, row)
		}
	}
}
