package torrshash

import (
	"bytes"
	"compress/zlib"
	"errors"
	"strings"
	"testing"
)

func TestLegacyTokenRoundTripAndV2Rejection(t *testing.T) {
	th := New(strings.Repeat("ab", 20))
	th.AddField(TagTitle, "Episode")
	th.AddField(TagSize, "10001")
	token, err := Pack(th)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Unpack(token)
	if err != nil || decoded.Hash != th.Hash || decoded.Title() != "Episode" {
		t.Fatal(decoded, err)
	}
	for _, bad := range []*TorrsHash{nil, New(strings.Repeat("ab", 32)), New("invalid")} {
		if _, err := Pack(bad); err == nil {
			t.Fatal("unsupported identity encoded")
		}
	}
	th.AddField(TagTitle, strings.Repeat("x", 65536))
	if _, err := Pack(th); err == nil {
		t.Fatal("field length overflow")
	}
}

func TestTokenDecompressionAndFieldBounds(t *testing.T) {
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	_, _ = w.Write(make([]byte, 20))
	for i := 0; i < maxTokenFields+1; i++ {
		_ = NewField(TagTitle, "x").write(w)
	}
	_ = w.Close()
	if _, err := UnpackBytes(compressed.Bytes()); err == nil {
		t.Fatal("unbounded fields accepted")
	}
	if _, err := Unpack(strings.Repeat("a", maxTokenBytes+1)); err == nil {
		t.Fatal("oversized encoded input accepted")
	}
	if _, err := ReadField(brokenReader{}); !errors.Is(err, errBroken) {
		t.Fatal("reader failure discarded", err)
	}
}

var errBroken = errors.New("fixture read failure")

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errBroken }
