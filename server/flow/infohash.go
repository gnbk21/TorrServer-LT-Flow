package flow

import (
	"encoding/hex"
	"fmt"
)

// InfoHash retains the complete digest. The last byte distinguishes SHA-256
// from SHA-1; zero remains the historical all-zero SHA-1 value. Keeping it
// comparable allows registry keys without truncating v2 identities.
type InfoHash [33]byte

func V1Hash(digest [20]byte) InfoHash {
	var h InfoHash
	copy(h[:20], digest[:])
	return h
}

func (h InfoHash) IsV2() bool { return h[32] == 2 }
func (h InfoHash) HexString() string {
	if h.IsV2() {
		return hex.EncodeToString(h[:32])
	}
	return hex.EncodeToString(h[:20])
}
func (h InfoHash) String() string { return h.HexString() }
func (h InfoHash) Magnet() string {
	if h.IsV2() {
		return "magnet:?xt=urn:btmh:1220" + h.HexString()
	}
	return "magnet:?xt=urn:btih:" + h.HexString()
}
func (h InfoHash) IsZero() bool {
	n := 20
	if h.IsV2() {
		n = 32
	}
	for _, b := range h[:n] {
		if b != 0 {
			return false
		}
	}
	return true
}
func (h InfoHash) MarshalText() ([]byte, error) { return []byte(h.HexString()), nil }
func (h *InfoHash) UnmarshalText(text []byte) error {
	if len(text) != 40 && len(text) != 64 {
		return fmt.Errorf("info hash: bad hex length %d (want 40 or 64)", len(text))
	}
	var next InfoHash
	if _, err := hex.Decode(next[:len(text)/2], text); err != nil {
		return fmt.Errorf("info hash: %w", err)
	}
	if len(text) == 64 {
		next[32] = 2
	}
	*h = next // malformed input must not mutate an existing registry identity
	return nil
}
func ParseInfoHash(text string) InfoHash {
	var h InfoHash
	_ = h.UnmarshalText([]byte(text))
	return h
}

func ValidInfoHash(text string) bool {
	var h InfoHash
	return h.UnmarshalText([]byte(text)) == nil
}
