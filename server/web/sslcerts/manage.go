package sslcerts

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"server/diagnostics"
	"server/settings"
)

// Certificates uploaded from the web UI are stored as copies in uploadDir, next to the
// settings. Each validated pair is staged separately until settings commit.
const (
	uploadDir      = "ssl"
	uploadCertName = "uploaded.crt"
	uploadKeyName  = "uploaded.key"
)

// MaxPEMSize limits uploaded cert and key files.
const MaxPEMSize = 1 << 20

// Source of the configured certificate.
const (
	SourceNone       = "none"        // no paths configured yet
	SourceSelfSigned = "self-signed" // generated and renewed by TorrServer
	SourceUploaded   = "uploaded"    // uploaded from the web UI
	SourceUser       = "user"        // any other path (--sslcert/--sslkey or settings)
)

// Info describes a certificate pair for the web UI. It never contains key material.
type Info struct {
	Source    string    `json:"source"`
	CertFile  string    `json:"cert_file,omitempty"`
	KeyFile   string    `json:"key_file,omitempty"`
	Subject   string    `json:"subject,omitempty"`
	Issuer    string    `json:"issuer,omitempty"`
	DNSNames  []string  `json:"dns_names,omitempty"`
	IPs       []string  `json:"ips,omitempty"`
	NotBefore time.Time `json:"not_before,omitzero"`
	NotAfter  time.Time `json:"not_after,omitzero"`
	// Trusted reports whether the chain verifies against this system's root CAs.
	Trusted bool   `json:"trusted"`
	Error   string `json:"error,omitempty"`
}

// Inspect describes the pair at the given paths. Load errors are reported in Info.Error.
func Inspect(certFile, keyFile string) Info {
	info := Info{Source: source(certFile, keyFile), CertFile: certFile, KeyFile: keyFile}
	if info.Source == SourceNone {
		return info
	}
	pair, err := loadPair(certFile, keyFile)
	if err != nil {
		info.Error = err.Error()
		if leaf := readLeaf(certFile); leaf != nil {
			describe(&info, leaf, nil)
		}
		return info
	}
	describe(&info, pair.Leaf, pair.Certificate[1:])
	return info
}

func describe(info *Info, leaf *x509.Certificate, chain [][]byte) {
	info.Subject = leaf.Subject.String()
	info.Issuer = leaf.Issuer.String()
	info.DNSNames = leaf.DNSNames
	for _, ip := range leaf.IPAddresses {
		info.IPs = append(info.IPs, ip.String())
	}
	info.NotBefore, info.NotAfter = leaf.NotBefore, leaf.NotAfter
	inter := x509.NewCertPool()
	for _, der := range chain {
		if c, err := x509.ParseCertificate(der); err == nil {
			inter.AddCert(c)
		}
	}
	_, err := leaf.Verify(x509.VerifyOptions{Intermediates: inter})
	info.Trusted = err == nil
}

func source(certFile, keyFile string) string {
	switch {
	case certFile == "" || keyFile == "":
		return SourceNone
	case IsGenerated(certFile, keyFile):
		return SourceSelfSigned
	case IsUploaded(certFile, keyFile):
		return SourceUploaded
	default:
		return SourceUser
	}
}

// SelfSignedPaths returns where the self-signed pair is kept. Passing them to EnsureCert
// reuses an existing pair instead of generating a new one.
func SelfSignedPaths() (string, string) {
	return generatedPaths()
}

// IsUploaded reports whether the paths point to the pair stored by SaveUploaded.
func IsUploaded(certFile, keyFile string) bool {
	c, k := uploadedPaths()
	if samePath(certFile, c) && samePath(keyFile, k) {
		return true
	}
	root, _ := filepath.Abs(filepath.Join(settings.Path, uploadDir))
	dir := filepath.Dir(certFile)
	return filepath.IsAbs(certFile) && filepath.IsAbs(keyFile) && filepath.Dir(keyFile) == dir && filepath.Dir(dir) == root && strings.HasPrefix(filepath.Base(dir), "pair-") && filepath.Base(certFile) == uploadCertName && filepath.Base(keyFile) == uploadKeyName
}

func uploadedPaths() (string, string) {
	dir := filepath.Join(settings.Path, uploadDir)
	return filepath.Join(dir, uploadCertName), filepath.Join(dir, uploadKeyName)
}

// SaveUploaded validates a PEM certificate (chain) and private key and stores them in
// <config>/ssl/, replacing a previous upload. It returns the absolute paths to configure.
// The pair must match and the leaf must be currently valid; it need not be trusted.
func SaveUploaded(certPEM, keyPEM []byte) (string, string, error) {
	if err := validatePEMPair(certPEM, keyPEM); err != nil {
		return "", "", err
	}
	root := filepath.Join(settings.Path, uploadDir)
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", "", err
	}
	dir, err := os.MkdirTemp(root, "pair-")
	if err != nil {
		return "", "", err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(filepath.Join(dir, uploadCertName))
			_ = os.Remove(filepath.Join(dir, uploadKeyName))
			_ = os.Remove(dir)
		}
	}()
	certPath, keyPath := filepath.Join(dir, uploadCertName), filepath.Join(dir, uploadKeyName)
	certPath, err = filepath.Abs(certPath)
	if err != nil {
		return "", "", err
	}
	if keyPath, err = filepath.Abs(keyPath); err != nil {
		return "", "", err
	}
	if err = os.MkdirAll(filepath.Dir(certPath), 0o700); err != nil {
		return "", "", err
	}
	// key first: until the cert is replaced too the pair mismatches and the Loader keeps
	// serving the previous certificate instead of a half-updated one
	if err = diagnostics.AtomicPrivateFile(keyPath, keyPEM); err != nil {
		return "", "", fmt.Errorf("write private key: %w", err)
	}
	if err = writeFileAtomic(certPath, certPEM, 0o644); err != nil {
		return "", "", fmt.Errorf("write certificate: %w", err)
	}
	complete = true
	return certPath, keyPath, nil
}

func validatePEMPair(certPEM, keyPEM []byte) error {
	if len(certPEM) > MaxPEMSize || len(keyPEM) > MaxPEMSize {
		return errors.New("PEM file exceeds 1 MiB")
	}
	if len(certPEM) > MaxPEMSize || len(keyPEM) > MaxPEMSize {
		return errors.New("PEM file exceeds 1 MiB")
	}
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		return errors.New("both a certificate and a private key are required")
	}
	for rest := certPEM; len(strings.TrimSpace(string(rest))) > 0; {
		block, tail := pem.Decode(rest)
		if block == nil || block.Type != "CERTIFICATE" {
			return errors.New("certificate file must contain only certificate PEM blocks")
		}
		rest = tail
	}
	if block, _ := pem.Decode(keyPEM); block == nil {
		return errors.New("private key is not PEM encoded")
	} else if x509.IsEncryptedPEMBlock(block) { //nolint:staticcheck // only detecting, not decrypting
		return errors.New("private key is password protected; upload it unencrypted")
	}
	pair, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return fmt.Errorf("certificate and key don't form a valid pair: %w", err)
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return err
	}
	now := time.Now()
	if now.Before(leaf.NotBefore) {
		return fmt.Errorf("certificate is not valid until %s", leaf.NotBefore.Format(time.RFC3339))
	}
	if now.After(leaf.NotAfter) {
		return fmt.Errorf("certificate has expired on %s", leaf.NotAfter.Format(time.RFC3339))
	}
	return nil
}

// RemoveUploaded deletes the uploaded pair, if any.
func RemoveUploaded() error {
	c, k := uploadedPaths()
	return RemoveUploadedPair(c, k)
}

// Only an exact managed pair can be retired; never recursively remove a path
// supplied by the user. Configuration holds its mutation lock while calling it.
func RemoveUploadedPair(c, k string) error {
	if !IsUploaded(c, k) {
		return nil
	}
	for _, f := range []string{c, k} {
		if err := os.Remove(f); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	legacyC, _ := uploadedPaths()
	if !samePath(c, legacyC) {
		_ = os.Remove(filepath.Dir(c))
	}
	return nil
}
