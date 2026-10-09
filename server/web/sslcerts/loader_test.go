package sslcerts

import (
	"bytes"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoaderReloadAndInvalidReplacement(t *testing.T) {
	withTempPath(t)
	cert, key, err := MakeCertKeyFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	loader, err := NewLoader(func() (string, string) { return cert, key })
	if err != nil {
		t.Fatal(err)
	}
	first, _ := loader.GetCertificate(nil)
	if _, _, err := MakeCertKeyFiles(nil); err != nil {
		t.Fatal(err)
	}
	loader.mu.Lock()
	loader.lastCheck = time.Time{}
	loader.mu.Unlock()
	second, err := loader.GetCertificate(nil)
	if err != nil || bytes.Equal(first.Certificate[0], second.Certificate[0]) {
		t.Fatal("valid replacement was not loaded", err)
	}
	if err := os.WriteFile(cert, []byte("broken replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	loader.mu.Lock()
	loader.lastCheck = time.Time{}
	loader.mu.Unlock()
	third, err := loader.GetCertificate(nil)
	if err != nil || !bytes.Equal(second.Certificate[0], third.Certificate[0]) {
		t.Fatal("failed replacement discarded the working certificate", err)
	}
}

func TestRenewalReasonAndStableAddressRetention(t *testing.T) {
	leaf := &x509.Certificate{NotAfter: time.Now().Add(365 * 24 * time.Hour), DNSNames: localDNSNames(), IPAddresses: []net.IP{net.ParseIP("192.168.1.10")}}
	if renewalReason(leaf, []string{"192.168.1.10", "2001:db8::1"}) != "" {
		t.Fatal("volatile IPv6 triggered renewal")
	}
	if renewalReason(leaf, []string{"192.168.1.11"}) == "" {
		t.Fatal("new LAN address was ignored")
	}
	leaf.NotAfter = time.Now().Add(time.Hour)
	if renewalReason(leaf, nil) == "" {
		t.Fatal("near expiry was ignored")
	}
	merged := mergeIPs([]net.IP{net.ParseIP("192.168.1.10"), net.ParseIP("2001:db8::1")}, []string{"192.168.1.11"})
	if len(merged) != 2 {
		t.Fatal("stable address retention", merged)
	}
}

func TestUploadedPairIsImmutableBoundedAndPrivate(t *testing.T) {
	withTempPath(t)
	certPEM, keyPEM, err := generateSelfSignedCert(nil)
	if err != nil {
		t.Fatal(err)
	}
	c1, k1, err := SaveUploaded(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	c2, k2, err := SaveUploaded(certPEM, keyPEM)
	if err != nil || c1 == c2 || k1 == k2 {
		t.Fatal("upload overwrote an existing identity", err)
	}
	if !IsUploaded(c1, k1) {
		t.Fatal("managed upload not recognized")
	}
	if _, _, err := SaveUploaded(append(certPEM, keyPEM...), keyPEM); err == nil {
		t.Fatal("private material accepted in the public certificate file")
	}
	if _, _, err := SaveUploaded(make([]byte, MaxPEMSize+1), keyPEM); err == nil {
		t.Fatal("oversized PEM accepted")
	}
	if err := RemoveUploadedPair(c1, k1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(c2); err != nil {
		t.Fatal("retirement removed the active pair", err)
	}
	outside := filepath.Join(t.TempDir(), "user.pem")
	if err := os.WriteFile(outside, certPEM, 0600); err != nil {
		t.Fatal(err)
	}
	if err := RemoveUploadedPair(outside, k2); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("user file was removed", err)
	}
}
