package flow

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type PlaybackClaim struct {
	Hash    string `json:"h"`
	Index   int    `json:"i"`
	Expires int64  `json:"e"`
}
type CapabilitySigner struct{ key [32]byte }

func NewCapabilitySigner() (*CapabilitySigner, error) {
	s := new(CapabilitySigner)
	_, err := rand.Read(s.key[:])
	return s, err
}
func validClaim(c PlaybackClaim) bool {
	b, err := hex.DecodeString(c.Hash)
	return err == nil && len(b) == 20 && c.Hash == strings.ToLower(c.Hash) && c.Index > 0
}
func (s *CapabilitySigner) Mint(c PlaybackClaim, now time.Time) (string, error) {
	if s == nil || !validClaim(c) || c.Expires <= now.Unix() || c.Expires > now.Add(24*time.Hour).Unix() {
		return "", errors.New("invalid playback capability")
	}
	b, _ := json.Marshal(c)
	payload := base64.RawURLEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, s.key[:])
	mac.Write([]byte(payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func (s *CapabilitySigner) Verify(token string, now time.Time) (PlaybackClaim, error) {
	var c PlaybackClaim
	if s == nil || len(token) > 512 {
		return c, errors.New("invalid playback capability")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return c, errors.New("invalid playback capability")
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return c, err
	}
	mac := hmac.New(sha256.New, s.key[:])
	mac.Write([]byte(parts[0]))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return c, errors.New("invalid playback capability")
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return c, err
	}
	if json.Unmarshal(b, &c) != nil || !validClaim(c) || c.Expires <= now.Unix() {
		return PlaybackClaim{}, errors.New("expired or invalid playback capability")
	}
	return c, nil
}
