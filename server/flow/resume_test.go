package flow

import (
	"bytes"
	"crypto/sha1"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestVerifyPieceFilesRejectsCorruptionHolesAndWrongFinalSize(t *testing.T) {
	dir := t.TempDir()
	const plen = 32768
	payload := bytes.Repeat([]byte{0x67}, plen)
	short := []byte("verified final piece")
	hashes := make([][20]byte, 6)
	for i := range hashes {
		hashes[i] = sha1.Sum(payload)
	}
	hashes[5] = sha1.Sum(short)
	write := func(i int, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, strconv.Itoa(i)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(0, payload)
	write(1, payload[:plen-1])
	write(2, bytes.Repeat([]byte{0x66}, plen)) // exact size, bad hash
	write(3, make([]byte, plen))               // sparse/unclean write, exact size
	write(4, append(bytes.Clone(payload), 0))
	write(5, short[:1])
	total := int64(5*plen + len(short))
	if got := VerifyPieceFiles(dir, plen, total, hashes); !bytes.Equal(got, []byte{1}) {
		t.Fatalf("unsafe resume bitmap: %x", got)
	}
	write(5, short)
	if got := VerifyPieceFiles(dir, plen, total, hashes); !bytes.Equal(got, []byte{0x21}) {
		t.Fatalf("verified short final piece omitted: %x", got)
	}
	if got := VerifyPieceFiles(dir, plen, total+plen, hashes); got != nil {
		t.Fatalf("inconsistent geometry accepted: %x", got)
	}
}
