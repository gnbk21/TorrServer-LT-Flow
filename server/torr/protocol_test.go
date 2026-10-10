package torr

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestV2BareDigestAndHybridAliasesRemainStable(t *testing.T) {
	text := strings.Repeat("ab", 32)
	spec, err := ParseLink(strings.ToUpper(text))
	if err != nil {
		t.Fatal(err)
	}
	if spec.InfoHash.HexString() != text || magnetFromSpec(spec) != "magnet:?xt=urn:btmh:1220"+text {
		t.Fatal("v2 digest was dispatched as a compact token")
	}
	v1 := NewHashFromHex(strings.Repeat("cd", 20))
	v2 := spec.InfoHash
	tor := &Torrent{TorrentSpec: &TorrentSpec{InfoHash: v2, InfoHashV1: v1.HexString(), InfoHashV2: v2.HexString()}}
	tor.identities.Store(&torrentIdentities{primary: v1, v1: v1, v2: v2})
	bt := &BTServer{torrents: map[Hash]*Torrent{v2: tor}}
	if bt.GetTorrent(v1) != tor || bt.GetTorrent(v2) != tor || len(bt.ListTorrents()) != 1 || tor.Hash() != v2 || tor.StorageHash() != v1 {
		t.Fatal("aliases changed an existing link or duplicated a torrent")
	}
	if bt.GetTorrent(Hash{}) != nil {
		t.Fatal("unknown alias matched a zero digest")
	}
	wire, err := json.Marshal(settingsSpec(tor.TorrentSpec))
	if err != nil {
		t.Fatal(err)
	}
	var restored TorrentSpec
	if err = json.Unmarshal(wire, &restored); err != nil || restored.InfoHash != v2 || restored.InfoHashV1 != v1.HexString() {
		t.Fatal("lost full persisted identity", err)
	}
}
