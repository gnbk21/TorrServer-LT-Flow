package web

import (
	"context"
	cryptoTLS "crypto/tls"
	"errors"
	stdlog "log"
	"net"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gstreamer "server/gstreamer/bridge"
	"server/netbind"

	"server/torrfs/fuse"
	"server/torrfs/webdav"

	"server/rutor"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/location/v2"
	"github.com/gin-gonic/gin"
	"github.com/wlynxg/anet"

	"server/bonjour"
	"server/dlna"
	"server/settings"
	"server/web/msx"

	"server/diagnostics"
	"server/flow"
	"server/log"
	"server/lt"
	"server/mcp"
	"server/torr"
	"server/version"
	"server/web/api"
	"server/web/auth"
	"server/web/pages"
	"server/web/sslcerts"
	"server/web/waf"
)

var (
	BTS            = torr.NewBTS()
	waitChan       = make(chan error, 1)
	serversMu      sync.Mutex
	servers        []*http.Server
	engineReady    atomic.Bool
	listenersReady atomic.Bool
	lifecycleMu    sync.Mutex
	stopRenew      chan struct{}
	renewDone      chan struct{}
)

//	@title			Swagger Torrserver API
//	@version		{version.Version}
//	@description	Torrent streaming server.

//	@license.name	GPL 3.0

//	@BasePath	/

//	@securityDefinitions.basic	BasicAuth

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func Start() {
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	listenersReady.Store(false)
	log.TLogln("Start TorrServer-LT " + version.Version + " libtorrent " + lt.Version())
	ips := GetLocalIps()
	if len(ips) > 0 {
		log.TLogln("Local IPs:", ips)
	}
	gin.SetMode(gin.ReleaseMode)

	// corsCfg := cors.DefaultConfig()
	// corsCfg.AllowAllOrigins = true
	// corsCfg.AllowHeaders = []string{"*"}
	// corsCfg.AllowMethods = []string{"*"}
	corsCfg := cors.DefaultConfig()
	corsCfg.AllowAllOrigins = true
	corsCfg.AllowPrivateNetwork = true
	corsCfg.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "HEAD", "OPTIONS", "DELETE"}
	corsCfg.AllowHeaders = []string{
		"Origin", "Content-Length", "Content-Type", "X-Requested-With", "Accept", "Authorization", "If-Match",
		// MCP Streamable HTTP (browser-based agents)
		"Mcp-Protocol-Version", "Mcp-Session-Id", "Last-Event-ID", "Mcp-Method", "Mcp-Name",
	}

	route := gin.New()
	route.Use(log.WebLogger(), auth.PreserveRejectedBody(), waf.WAF(), gin.Recovery(), api.ManagementPolicy(), cors.New(corsCfg), location.Default())
	engineReady.Store(false)
	diagnostics.MarkEngineReady(false)
	route.Use(func(c *gin.Context) {
		if !engineReady.Load() && c.Request.URL.Path != "/echo" && c.Request.URL.Path != "/flow/network" {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		c.Next()
	})
	auth.SetupAuth(route)
	route.Use(api.PlaybackPolicy())
	route.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/flow/maintenance" || strings.HasPrefix(path, "/flow/backup") || path == "/echo" || path == "/flow/tray" || path == "/flow/network" || path == "/runtime/status" || strings.HasPrefix(path, "/shutdown") || path == "/" || strings.HasPrefix(path, "/assets/") {
			c.Next()
			return
		}
		if !flow.Maintenance.Enter() {
			c.Header("Retry-After", "30")
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		defer flow.Maintenance.Leave()
		c.Next()
	})

	route.GET("/echo", echo)

	api.SetupRoute(route)
	setupSSLRoutes(route)
	mcp.Mount(route.Group("/", auth.CheckAuth()))
	gstreamer.SetupRoute(route)
	msx.SetupRoute(route)
	pages.SetupRoute(route)
	if settings.Args.WebDAV {
		webdav.MountWebDAV(route)
	}

	route.GET("/swagger/*any", swaggerHandler())

	// Explicit HTTPS configuration fails before any listener is opened. Never
	// replace a user's identity or silently downgrade a forced-HTTPS deployment.
	if settings.Ssl {
		cert, key, changed, err := sslcerts.EnsureCert(settings.BTsets().SslCert, settings.BTsets().SslKey, certIPs())
		if err != nil {
			startupError(err)
			return
		}
		if changed {
			next := settings.CloneSettings(settings.BTsets())
			next.SslCert, next.SslKey = cert, key
			if settings.ReadOnly {
				settings.StoreBTsets(next)
			} else if err := settings.SetBTSetsChecked(next); err != nil {
				startupError(err)
				return
			}
		}
	}
	// Bind and serve the local API before constructing the libtorrent session.
	// Only /echo and /flow/network answer until the engine is ready.
	for _, ip := range netbind.Normalize(settings.IPs) {
		if settings.Ssl {
			if err := startListener(route, netbind.Addr(ip, settings.SslPort), true); err != nil {
				startupError(err)
				return
			}
		}
		handler := http.Handler(route)
		if settings.Args != nil && settings.Args.ForceHTTPS && settings.Ssl {
			handler = forceHTTPSHandler(route, settings.Args.HTTPMedia)
		}
		if !settings.HTTPEnabled() {
			continue
		}
		if err := startListener(handler, netbind.Addr(ip, settings.Port), false); err != nil {
			startupError(err)
			return
		}
	}
	if err := startInternalListener(route); err != nil {
		startupError(err)
		return
	}
	if settings.Ssl {
		stopRenew, renewDone = make(chan struct{}), make(chan struct{})
		stop, done := stopRenew, renewDone
		go func() { defer close(done); sslcerts.RenewLoop(stop, time.Hour, sslCertPaths, certIPs) }()
		if !settings.PlainHTTPServesMedia() && sslcerts.IsGenerated(sslCertPaths()) {
			log.TLogln("HTTPS media uses a self-signed certificate; players must trust it, or configure a trusted certificate")
		}
	}
	listenersReady.Store(true)
	diagnostics.MarkListenersReady(true)
	if err := BTS.Connect(); err != nil {
		startupError(err)
		return
	}
	engineReady.Store(true)
	diagnostics.MarkEngineReady(true)
	rutor.Start()
	if settings.BTsets().EnableDLNA {
		dlna.Start()
	}
	if settings.BTsets().EnableBonjour {
		bonjour.Start()
	}
	// Auto-mount FUSE filesystem if enabled.
	fuse.FuseAutoMount()
}

func Wait() error {
	return <-waitChan
}

// ListenersReady reports this process's successful binds independently of
// torrent engine initialization or external network availability.
func ListenersReady() bool { return listenersReady.Load() }

func Stop() {
	// Finish initialization before disconnecting; otherwise Connect can finish
	// after Stop and leave an engine running against a closed database.
	lifecycleMu.Lock()
	defer lifecycleMu.Unlock()
	listenersReady.Store(false)
	engineReady.Store(false)
	diagnostics.MarkEngineReady(false)
	shutdownListeners()
	gstreamer.Stop()
	dlna.Stop()
	bonjour.Stop()
	// Unmount FUSE filesystem if mounted
	fuse.FuseCleanup()
	BTS.Disconnect()
	select {
	case waitChan <- nil:
	default:
	}
}

func registerServer(handler http.Handler) *http.Server {
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 120 * time.Second, ErrorLog: stdlog.New(serverErrors, "", 0)}
	serversMu.Lock()
	servers = append(servers, srv)
	serversMu.Unlock()
	return srv
}

func reportServe(fn func() error) {
	go func() {
		if err := fn(); err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
			select {
			case waitChan <- err:
			default:
			}
		}
	}()
}

func startListener(handler http.Handler, addr string, secure bool) error {
	var loader *sslcerts.Loader
	if secure {
		var err error
		loader, err = sslcerts.NewLoader(sslCertPaths)
		if err != nil {
			return err
		}
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	srv := registerServer(handler)
	if secure {
		tlsLn, plainLn := splitTLS(listener)
		srv.TLSConfig = &cryptoTLS.Config{GetCertificate: loader.GetCertificate}
		redirect := registerServer(httpsRedirectHandler())
		reportServe(func() error { return srv.ServeTLS(tlsLn, "", "") })
		reportServe(func() error { return redirect.Serve(plainLn) })
		log.TLogln("Start https server at", addr)
	} else {
		reportServe(func() error { return srv.Serve(listener) })
		log.TLogln("Start http server at", addr)
	}
	return nil
}

func startInternalListener(handler http.Handler) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	settings.SetInternalBaseURL("http://" + listener.Addr().String())
	srv := registerServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		media := strings.HasPrefix(r.URL.Path, "/play/") || r.URL.Path == "/stream" || strings.HasPrefix(r.URL.Path, "/stream/")
		if !torr.IsInternalProbe(r) || !media || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
			http.Error(w, "internal media access required", http.StatusForbidden)
			return
		}
		handler.ServeHTTP(w, r)
	}))
	reportServe(func() error { return srv.Serve(listener) })
	return nil
}

func certIPs() []string {
	var bound []string
	for _, addr := range settings.IPs {
		ip := net.ParseIP(addr)
		if ip == nil || ip.IsUnspecified() {
			return GetLocalIps()
		}
		bound = append(bound, ip.String())
	}
	if len(bound) == 0 {
		return GetLocalIps()
	}
	return bound
}

func sslCertPaths() (string, string) {
	s := settings.BTsets()
	if s == nil {
		return "", ""
	}
	return s.SslCert, s.SslKey
}

func startupError(err error) {
	listenersReady.Store(false)
	log.TLogln("Flow startup error:", err)
	shutdownListeners()
	select {
	case waitChan <- err:
	default:
	}
}

func shutdownListeners() {
	if stopRenew != nil {
		close(stopRenew)
		<-renewDone
		stopRenew, renewDone = nil, nil
	}
	settings.SetInternalBaseURL("")
	serversMu.Lock()
	current := servers
	servers = nil
	serversMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, srv := range current {
		if err := srv.Shutdown(ctx); err != nil {
			_ = srv.Close()
		}
	}
}

// echo godoc
//
//	@Summary		Tests server status
//	@Description	Tests whether server is alive or not
//
//	@Tags			API
//
//	@Produce		plain
//	@Success		200	{string}	string	"Server version"
//	@Router			/echo [get]
func echo(c *gin.Context) {
	c.String(200, "%v", version.Version)
}

func GetLocalIps() []string {
	ifaces, err := anet.Interfaces()
	if err != nil {
		log.TLogln("Error get local IPs")
		return nil
	}
	var list []string
	for _, i := range ifaces {
		addrs, _ := anet.InterfaceAddrsByInterface(&i)
		if i.Flags&net.FlagUp == net.FlagUp {
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() {
					list = append(list, ip.String())
				}
			}
		}
	}
	sort.Strings(list)
	return list
}
