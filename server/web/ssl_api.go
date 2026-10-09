package web

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"

	"encoding/pem"
	"os"
	"server/log"
	"server/settings"
	"server/torr"
	"server/web/auth"
	"server/web/sslcerts"
)

// sslMu serialises certificate changes made through the API.
var sslMu sync.Mutex

type sslStatus struct {
	// Enabled is true when TorrServer was started with --ssl. HTTP/HTTPS modes and ports
	// are startup flags; the certificate can only be managed here while HTTPS runs.
	Enabled     bool   `json:"enabled"`
	Port        string `json:"port,omitempty"`
	HTTPPort    string `json:"http_port,omitempty"`
	HTTPEnabled bool   `json:"http_enabled"`
	ForceHTTPS  bool   `json:"force_https"`
	HTTPMedia   bool   `json:"http_media"`
	ReadOnly    bool   `json:"read_only"`
	// CertFromFlags is true when --sslcert/--sslkey set the paths: they are applied again
	// on every start, so the certificate can't be changed here.
	Revision      string        `json:"revision"`
	CertFromFlags bool          `json:"cert_from_flags"`
	Cert          sslcerts.Info `json:"cert"`
}

func setupSSLRoutes(route gin.IRouter) {
	g := route.Group("/ssl", auth.CheckAuth())
	g.GET("/status", sslStatusHandler)
	g.GET("/cert", sslCertDownload)
	g.POST("/upload", sslUpload)
	g.POST("/paths", sslSetPaths)
	g.POST("/selfsigned", sslUseSelfSigned)
	g.POST("/regenerate", sslRegenerate)
}

func currentSSLStatus() sslStatus {
	st := sslStatus{
		Enabled:       settings.Ssl,
		Revision:      torr.ConfigurationSnapshot().Revision,
		HTTPPort:      settings.Port,
		HTTPEnabled:   settings.HTTPEnabled(),
		ReadOnly:      settings.ReadOnly,
		CertFromFlags: certFromFlags(),
		Cert:          sslcerts.Inspect(sslCertPaths()),
	}
	if settings.Ssl {
		st.Port = settings.SslPort
		if settings.Args != nil {
			st.ForceHTTPS = settings.Args.ForceHTTPS
			st.HTTPMedia = settings.Args.ForceHTTPS && settings.Args.HTTPMedia
		}
	}
	return st
}

// sslStatusHandler godoc
//
//	@Summary		HTTPS status
//	@Description	HTTPS mode, ports and the configured certificate (subject, SANs, issuer, validity, source). Never returns key material.
//
//	@Tags			API
//	@Produce		json
//	@Security		BasicAuth
//	@Success		200	{object}	sslStatus
//	@Router			/ssl/status [get]
func sslStatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, currentSSLStatus())
}

// sslCertDownload godoc
//
//	@Summary		Download the HTTPS certificate
//	@Description	The configured certificate (chain) in PEM, e.g. to trust the self-signed one on a device. The private key is never served.
//
//	@Tags			API
//	@Produce		application/x-x509-ca-cert
//	@Security		BasicAuth
//	@Success		200	{file}		file
//	@Failure		404	{object}	map[string]string
//	@Router			/ssl/cert [get]
func sslCertDownload(c *gin.Context) {
	certFile, keyFile := sslCertPaths()
	if !settings.Ssl || certFile == "" || keyFile == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "no certificate configured"})
		return
	}
	f, err := os.Open(certFile)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, sslcerts.MaxPEMSize+1))
	if err != nil || len(data) > sslcerts.MaxPEMSize {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	var public []byte
	for len(data) > 0 {
		block, rest := pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			public = append(public, pem.EncodeToMemory(block)...)
		}
		data = rest
	}
	if len(public) == 0 {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="torrserver.crt"`)
	c.Data(http.StatusOK, "application/x-x509-ca-cert", public)
}

// sslUpload godoc
//
//	@Summary		Upload an HTTPS certificate
//	@Description	Stores a PEM certificate (chain) and its unencrypted private key in <config>/ssl/ and uses them. The pair must match and be currently valid. Served without a restart when HTTPS is running.
//
//	@Tags			API
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BasicAuth
//	@Param			cert	formData	file	true	"Certificate (chain), PEM"
//	@Param			key		formData	file	true	"Private key, PEM"
//	@Success		200		{object}	sslStatus
//	@Failure		400		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Param			If-Match	header	string	true	"Current settings revision from /ssl/status"
//	@Router			/ssl/upload [post]
func sslUpload(c *gin.Context) {
	if denyCertChange(c) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2*sslcerts.MaxPEMSize+64<<10)
	certPEM, err := formFile(c, "cert")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	keyPEM, err := formFile(c, "key")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sslMu.Lock()
	defer sslMu.Unlock()
	var cert, key string
	err = applySSLCertificate(c, func() (string, string, error) {
		var err error
		cert, key, err = sslcerts.SaveUploaded(certPEM, keyPEM)
		return cert, key, err
	})
	if err != nil {
		if cert != "" {
			_ = os.Remove(cert)
			_ = os.Remove(key)
			_ = os.Remove(filepath.Dir(cert))
		}
		certError(c, err)
		return
	}
	log.TLogln("Uploaded HTTPS certificate saved")
	c.JSON(http.StatusOK, currentSSLStatus())
}

func formFile(c *gin.Context, field string) ([]byte, error) {
	fh, err := c.FormFile(field)
	if err != nil {
		return nil, errors.New(field + ": file is required")
	}
	if fh.Size > sslcerts.MaxPEMSize {
		return nil, errors.New(field + ": file is too large")
	}
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, sslcerts.MaxPEMSize+1))
	if len(data) > sslcerts.MaxPEMSize {
		return nil, errors.New(field + ": file is too large")
	}
	return data, err
}

type sslPathsReq struct {
	Cert string `json:"cert" binding:"required"`
	Key  string `json:"key" binding:"required"`
}

// sslSetPaths godoc
//
//	@Summary		Use HTTPS certificate files by path
//	@Description	Uses a certificate (chain) and key already on the server, e.g. kept up to date by acme.sh or certbot. The pair must load, match and be currently valid. Renewals of these files are picked up without a restart.
//
//	@Tags			API
//	@Accept			json
//	@Produce		json
//	@Security		BasicAuth
//	@Param			request	body		sslPathsReq	true	"Absolute paths of the certificate and key files"
//	@Success		200		{object}	sslStatus
//	@Failure		400		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Param			If-Match	header	string	true	"Current settings revision from /ssl/status"
//	@Router			/ssl/paths [post]
func sslSetPaths(c *gin.Context) {
	if denyCertChange(c) {
		return
	}
	var req sslPathsReq
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cert and key paths are required"})
		return
	}
	cert, err := filepath.Abs(req.Cert)
	if err == nil {
		req.Key, err = filepath.Abs(req.Key)
	}
	if err == nil {
		err = sslcerts.VerifyCertKeyFiles(cert, req.Key)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sslMu.Lock()
	defer sslMu.Unlock()
	if err := setSSLCertPaths(c, cert, req.Key); err != nil {
		certError(c, err)
		return
	}
	c.JSON(http.StatusOK, currentSSLStatus())
}

// sslUseSelfSigned godoc
//
//	@Summary		Use the self-signed HTTPS certificate
//	@Description	Switches to TorrServer's self-signed certificate (reused, or generated if missing) and deletes an uploaded one. Certificate files given by path are left on disk.
//
//	@Tags			API
//	@Produce		json
//	@Security		BasicAuth
//	@Success		200	{object}	sslStatus
//	@Failure		403	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Param			If-Match	header	string	true	"Current settings revision from /ssl/status"
//	@Router			/ssl/selfsigned [post]
func sslUseSelfSigned(c *gin.Context) {
	if denyCertChange(c) {
		return
	}
	sslMu.Lock()
	defer sslMu.Unlock()
	// reuse the existing self-signed pair: devices may already trust it
	err := applySSLCertificate(c, func() (string, string, error) {
		c0, k0 := sslcerts.SelfSignedPaths()
		cert, key, _, err := sslcerts.EnsureCert(c0, k0, certIPs())
		return cert, key, err
	})
	if err != nil {
		certError(c, err)
		return
	}
	c.JSON(http.StatusOK, currentSSLStatus())
}

// sslRegenerate godoc
//
//	@Summary		Regenerate the self-signed HTTPS certificate
//	@Description	Creates a new self-signed certificate and key for the current local IPs and hostname. Only when the self-signed certificate is in use.
//
//	@Tags			API
//	@Produce		json
//	@Security		BasicAuth
//	@Success		200	{object}	sslStatus
//	@Failure		403	{object}	map[string]string
//	@Failure		409	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Param			If-Match	header	string	true	"Current settings revision from /ssl/status"
//	@Router			/ssl/regenerate [post]
func sslRegenerate(c *gin.Context) {
	if denyCertChange(c) {
		return
	}
	sslMu.Lock()
	defer sslMu.Unlock()
	if !sslcerts.IsGenerated(sslCertPaths()) {
		c.JSON(http.StatusConflict, gin.H{"error": "not using the self-signed certificate"})
		return
	}
	err := applySSLCertificate(c, func() (string, string, error) {
		if !sslcerts.IsGenerated(sslCertPaths()) {
			return "", "", errors.New("not using the self-signed certificate")
		}
		return sslcerts.MakeCertKeyFiles(certIPs())
	})
	if err != nil {
		certError(c, err)
		return
	}
	c.JSON(http.StatusOK, currentSSLStatus())
}

func certFromFlags() bool {
	return settings.Args != nil && (settings.Args.SslCert != "" || settings.Args.SslKey != "")
}

// denyCertChange rejects certificate changes that can't be saved or would be undone on
// the next start.
func denyCertChange(c *gin.Context) bool {
	switch {
	case !settings.Ssl:
		c.JSON(http.StatusConflict, gin.H{"error": "HTTPS is not enabled (start TorrServer with --ssl)"})
	case settings.ReadOnly:
		c.JSON(http.StatusForbidden, gin.H{"error": "Read-only mode"})
	case certFromFlags():
		c.JSON(http.StatusConflict, gin.H{"error": "the certificate is set by --sslcert/--sslkey"})
	default:
		return false
	}
	return true
}

// setSSLCertPaths saves new cert paths; the Loader serves them within a few seconds.
// Uploaded pairs are immutable until saved; the previous managed upload is then retired.
func setSSLCertPaths(c *gin.Context, cert, key string) error {
	return applySSLCertificate(c, func() (string, string, error) {
		return cert, key, sslcerts.VerifyCertKeyFiles(cert, key)
	})
}

func certError(c *gin.Context, err error) {
	code := http.StatusBadRequest
	if errors.Is(err, torr.ErrSettingsConflict) {
		code = http.StatusConflict
	}
	c.JSON(code, gin.H{"error": err.Error()})
}

func applySSLCertificate(c *gin.Context, prepare func() (string, string, error)) error {
	return torr.ApplyCertificateConfiguration(c.GetHeader("If-Match"), prepare, func(oldCert, oldKey string) {
		if err := sslcerts.RemoveUploadedPair(oldCert, oldKey); err != nil {
			log.TLogln("Could not retire previous uploaded certificate:", err)
		}
	})
}
