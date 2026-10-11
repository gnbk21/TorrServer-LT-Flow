package tgbot

import (
	"encoding/base64"
	"encoding/hex"
	"strings"

	tele "gopkg.in/telebot.v4"
)

// Telegram callback data is limited to 64 bytes including the action and
// file index. Encode all 32 v2 bytes losslessly; legacy buttons stay compatible.
func callbackHash(hash string) string {
	if len(hash) != 64 {
		return hash
	}
	data, err := hex.DecodeString(hash)
	if err != nil {
		return hash
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func expandCallbackData(data string) string {
	parts := strings.Split(data, "|")
	for i, part := range parts {
		if len(part) != 43 {
			continue
		}
		decoded, err := base64.RawURLEncoding.Strict().DecodeString(part)
		if err == nil && len(decoded) == 32 {
			parts[i] = hex.EncodeToString(decoded)
		}
	}
	return strings.Join(parts, "|")
}

func callbackArgs(c tele.Context) []string {
	args := append([]string(nil), c.Args()...)
	if c.Callback() == nil {
		return args
	}
	for i := 1; i < len(args); i++ {
		args[i] = expandCallbackData(args[i])
	}
	return args
}
