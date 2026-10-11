package log

import (
	"bytes"
	"errors"
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStandardAndDirectAccessLoggerCannotBypassFileRedaction(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owned.log")
	Init(path, path)
	t.Cleanup(Close)
	stdlog.Printf("lt: native error https://fixture-user:fixture-password@tracker.example/fixture-passkey/announce?token=fixture-token")
	webLog.Print("/flow/play/fixture-capability Authorization: Bearer fixture-credential")
	stdlog.Print("native title\r\n[ERROR] forged\x1b[2J")
	TLogln("remote title\n[ERROR] forged")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"fixture-user", "fixture-password", "fixture-passkey", "fixture-token", "fixture-capability", "fixture-credential"} {
		if bytes.Contains(contents, []byte(secret)) {
			t.Fatalf("standard/native logger bypassed redaction: %s", secret)
		}
	}
	if !bytes.Contains(contents, []byte("tracker.example")) || !bytes.Contains(contents, []byte("lt: native error")) {
		t.Fatal("useful destination and event context were removed")
	}
	if bytes.Count(contents, []byte("\n")) != 4 || bytes.ContainsAny(contents, "\r\x1b") {
		t.Fatalf("remote/native input forged a log record: %q", contents)
	}
}

type incompleteLogWriter struct{ failure error }

func (w incompleteLogWriter) Write(p []byte) (int, error) { return len(p) / 2, w.failure }

func TestRedactionWriterReportsInputConsumptionAndSinkFailure(t *testing.T) {
	var output bytes.Buffer
	input := []byte("token=x\n") // Redaction expands this input.
	n, err := (redactingWriter{&output}).Write(input)
	if err != nil || n != len(input) || !strings.Contains(output.String(), "[redacted]") {
		t.Fatal("redaction violated the writer contract", n, err)
	}
	if n, err = (redactingWriter{incompleteLogWriter{}}).Write(input); n != 0 || !errors.Is(err, io.ErrShortWrite) {
		t.Fatal("short sink write was hidden", n, err)
	}
	failure := errors.New("owned sink unavailable")
	if n, err = (redactingWriter{incompleteLogWriter{failure}}).Write(input); n != 0 || !errors.Is(err, failure) {
		t.Fatal("sink error was hidden", n, err)
	}
}
