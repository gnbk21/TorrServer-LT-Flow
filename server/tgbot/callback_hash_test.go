package tgbot

import (
	"strconv"
	"strings"
	"testing"
)

func TestCallbackHashPreservesFullIdentityAndButtonLimit(t *testing.T) {
	for _, hash := range []string{strings.Repeat("ab", 20), strings.Repeat("ab", 32), strings.Repeat("ab", 20) + strings.Repeat("cd", 12)} {
		encoded := callbackHash(hash)
		if expandCallbackData(encoded) != hash || expandCallbackData(encoded+"|9") != hash+"|9" {
			t.Fatal("callback lost digest bytes")
		}
		for _, action := range []string{"upload", "fpreload", "fitems", "fifresh", "flink"} {
			data := "\f" + action + "|" + encoded + "|" + strconv.Itoa(2147483647)
			if len(data) > 64 {
				t.Fatalf("%s callback exceeds Telegram limit: %d", action, len(data))
			}
		}
	}
	if expandCallbackData(strings.Repeat("_", 43)) != strings.Repeat("_", 43) {
		t.Fatal("accepted noncanonical padding bits")
	}
}
