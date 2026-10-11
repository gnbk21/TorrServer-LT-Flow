package flow

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInfoHashPreservesV1AndFullV2(t *testing.T) {
	for _, text := range []string{strings.Repeat("ab", 20), strings.Repeat("cd", 32)} {
		var hash InfoHash
		if err := hash.UnmarshalText([]byte(strings.ToUpper(text))); err != nil {
			t.Fatal(err)
		}
		if hash.HexString() != text || hash.IsV2() != (len(text) == 64) {
			t.Fatal("changed identity", hash)
		}
		wire, err := json.Marshal(hash)
		if err != nil || string(wire) != `"`+text+`"` {
			t.Fatalf("schema changed: %s %v", wire, err)
		}
		var restored InfoHash
		if err = json.Unmarshal(wire, &restored); err != nil || restored != hash {
			t.Fatal("database round trip", err)
		}
		prior := hash
		if err = hash.UnmarshalText([]byte(strings.Repeat("z", len(text)))); err == nil || hash != prior {
			t.Fatal("malformed digest mutated identity")
		}
	}
	a := ParseInfoHash(strings.Repeat("11", 32))
	b := ParseInfoHash(strings.Repeat("11", 20) + strings.Repeat("22", 12))
	v1 := ParseInfoHash(strings.Repeat("11", 20))
	if a == b || a == v1 || b == v1 {
		t.Fatal("truncated or cross-algorithm registry collision")
	}
	if (InfoHash{}).HexString() != strings.Repeat("0", 40) || !(InfoHash{}).IsZero() {
		t.Fatal("legacy zero changed")
	}
}

func FuzzInfoHash(f *testing.F) {
	f.Add(strings.Repeat("AB", 20))
	f.Add(strings.Repeat("CD", 32))
	f.Add("invalid")
	f.Fuzz(func(t *testing.T, text string) {
		var hash InfoHash
		if err := hash.UnmarshalText([]byte(text)); err == nil {
			wire, _ := hash.MarshalText()
			var restored InfoHash
			if restored.UnmarshalText(wire) != nil || restored != hash {
				t.Fatal("unstable identity")
			}
		}
	})
}
