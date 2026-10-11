package api

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestErrorMessageRemovesCredentialsAndBoundsUntrustedDetails(t *testing.T) {
	message := errorMessage(errors.New(`request https://user:password@example.test/private-passkey/announce?token=credential failed; Authorization: Bearer topsecret`))
	for _, secret := range []string{"user:", "password", "private-passkey", "credential", "topsecret"} {
		if strings.Contains(message, secret) {
			t.Fatal("credential exposed in API error")
		}
	}
	message = errorMessage(errors.New(strings.Repeat("例", 5000)))
	if len(message) > 4100 || !utf8.ValidString(message) {
		t.Fatal("error response is unbounded or invalid UTF-8")
	}
	if errorMessage(errors.New("metadata timed out")) != "metadata timed out" {
		t.Fatal("useful explanation removed")
	}
}
