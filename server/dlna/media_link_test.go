package dlna

import (
	"server/settings"
	"testing"
)

func TestMediaLinkIPv6AndHTTPSOnly(t *testing.T) {
	oldSSL, oldPort, oldSSLPort, oldArgs := settings.Ssl, settings.Port, settings.SslPort, settings.Args
	t.Cleanup(func() {
		settings.Ssl, settings.Port, settings.SslPort, settings.Args = oldSSL, oldPort, oldSSLPort, oldArgs
	})
	settings.Ssl, settings.Port, settings.SslPort = true, "8090", "8091"
	settings.Args = &settings.ExecArgs{HTTPSOnly: true}
	if got := getLink("http://[fd00::1]:9080", "play/hash/1"); got != "https://[fd00::1]:8091/play/hash/1" {
		t.Fatal(got)
	}
	settings.Args = &settings.ExecArgs{ForceHTTPS: true, HTTPMedia: true}
	if got := getLink("http://192.168.1.2:9080", "play/hash/1"); got != "http://192.168.1.2:8090/play/hash/1" {
		t.Fatal(got)
	}
}
