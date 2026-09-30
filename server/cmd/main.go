package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/browser"

	"server"
	"server/diagnostics"
	"server/docs"
	"server/log"
	"server/lt"
	"server/settings"
	"server/torr"
	"server/version"
)

type args struct {
	Port           string   `arg:"-p" help:"web server port (default 8090)"`
	IPs            []string `arg:"-i,--ip,separate" help:"web server bind addr (repeatable; default empty binds all interfaces)"`
	Ssl            bool     `help:"enables https"`
	SslPort        string   `help:"web server ssl port, If not set, will be set to default 8091 or taken from db(if stored previously). Accepted if --ssl enabled."`
	SslCert        string   `help:"path to ssl cert file. If not set, will be taken from db(if stored previously) or default self-signed certificate/key will be generated. Accepted if --ssl enabled."`
	SslKey         string   `help:"path to ssl key file. If not set, will be taken from db(if stored previously) or default self-signed certificate/key will be generated. Accepted if --ssl enabled."`
	Path           string   `arg:"-d" help:"database and config dir path"`
	LogPath        string   `arg:"-l" help:"server log file path"`
	WebLogPath     string   `arg:"-w" help:"web access log file path"`
	RDB            bool     `arg:"-r" help:"start in read-only DB mode"`
	HttpAuth       bool     `arg:"-a" help:"enable http auth on all requests"`
	DontKill       bool     `arg:"-k" help:"don't kill server on signal"`
	UI             bool     `arg:"-u" help:"open torrserver page in browser"`
	TorrentsDir    string   `arg:"-t" help:"autoload torrents from dir"`
	TorrentAddr    string   `help:"Torrent client address, like 127.0.0.1:1337 (default :PeersListenPort)"`
	PubIPv4        string   `arg:"-4" help:"set public IPv4 addr"`
	PubIPv6        string   `arg:"-6" help:"set public IPv6 addr"`
	SearchWA       bool     `arg:"-s" help:"search without auth"`
	StreamWA       bool     `arg:"--streamwa" help:"stream play and m3u without auth (auto-add torrents for external players)"`
	MaxSize        string   `arg:"-m" help:"max allowed stream size (in Bytes)"`
	TGToken        string   `arg:"-T" help:"telegram bot token"`
	FusePath       string   `arg:"-f" help:"fuse mount path"`
	WebDAV         bool     `help:"web dav enable"`
	ProxyURL       string   `help:"proxy URL for BitTorrent traffic (http, socks4, socks5, socks5h), e.g. socks5://user:password@127.0.0.1:8080"`
	ProxyMode      string   `help:"proxy mode: tracker (only HTTP trackers, default), peers (only peer connections), or full (all traffic)"`
	ForceHTTPS     bool     `arg:"--force-https" help:"redirect all HTTP requests to HTTPS (requires --ssl)"`
	Service        string   `arg:"--service" help:"Windows service command: install, start, stop, restart, uninstall, or run"`
	ProfileAddress string   `arg:"--profile-address" help:"opt-in profiling listener on a numeric loopback address, e.g. 127.0.0.1:6060"`
	Doctor         bool     `arg:"--doctor" help:"check local state, port, authentication and optional dependencies without starting the server"`
}

func (args) Version() string {
	return "TorrServer-LT " + version.Version + " (libtorrent " + lt.Version() + ")"
}

var params args

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	arg.MustParse(&params)
	if params.ProfileAddress != "" {
		stop, err := diagnostics.StartProfiling(params.ProfileAddress)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Flow profiling:", err)
			os.Exit(1)
		}
		defer stop()
	}
	if params.Service != "" && params.Service != "run" {
		if err := serviceCommand(params.Service, &params); err != nil {
			fmt.Fprintln(os.Stderr, "Flow service:", err)
			os.Exit(1)
		}
		return
	}
	if params.Service == "run" {
		if params.Path == "" {
			params.Path = serviceDataDir()
		}
		if err := os.MkdirAll(params.Path, 0750); err != nil {
			fmt.Fprintln(os.Stderr, "Flow service state directory:", err)
			os.Exit(1)
		}
		if params.LogPath == "" {
			params.LogPath = filepath.Join(params.Path, "flow.log")
		}
	}

	if params.Path == "" {
		params.Path, _ = os.Getwd()
	}

	if params.Port == "" {
		params.Port = "8090"
	}

	settings.Path = params.Path
	if params.Doctor {
		checks, valid := diagnostics.Doctor(params.Path, params.Port, params.IPs, params.HttpAuth)
		json.NewEncoder(os.Stdout).Encode(checks)
		if !valid {
			os.Exit(1)
		}
		return
	}
	settings.HttpAuth = params.HttpAuth
	log.Init(params.LogPath, params.WebLogPath)

	log.TLogln("=========== START ===========")
	log.TLogln("TorrServer-LT", version.Version+",", "libtorrent", lt.Version()+",", runtime.Version()+",", "CPU Num:", runtime.NumCPU())
	if params.HttpAuth {
		log.TLogln("Use HTTP Auth file", settings.Path+"/accs.db")
	}
	if params.RDB {
		log.TLogln("Running in Read-only DB mode!")
	}
	docs.SwaggerInfo.Version = version.Version

	// External DNS is not a startup prerequisite. The system resolver remains
	// the default; libtorrent retries tracker and DHT work as the network comes up.

	Preconfig(params.DontKill)

	if params.UI {
		go func() {
			time.Sleep(time.Second)
			if params.Ssl {
				browser.OpenURL("https://127.0.0.1:" + params.SslPort)
			} else {
				browser.OpenURL("http://127.0.0.1:" + params.Port)
			}
		}()
	}

	if params.TorrentAddr != "" {
		settings.TorAddr = params.TorrentAddr
	}

	if params.PubIPv4 != "" {
		settings.PubIPv4 = params.PubIPv4
	}

	if params.PubIPv6 != "" {
		settings.PubIPv6 = params.PubIPv6
	}

	if params.TorrentsDir != "" {
		go watchTDir(params.TorrentsDir)
	}

	if params.MaxSize != "" {
		maxSize, err := strconv.ParseInt(params.MaxSize, 10, 64)
		if err == nil {
			settings.MaxSize = maxSize
		}
	}

	if params.ProxyURL != "" && params.ProxyMode == "" {
		params.ProxyMode = "tracker" // default
	}
	if params.ProxyMode != "" && params.ProxyMode != "tracker" && params.ProxyMode != "peers" && params.ProxyMode != "full" {
		log.TLogln("Invalid proxy mode, using default 'tracker'")
		params.ProxyMode = "tracker"
	}

	settings.Args = &settings.ExecArgs{
		Port:        params.Port,
		IPs:         params.IPs,
		Ssl:         params.Ssl,
		SslPort:     params.SslPort,
		SslCert:     params.SslCert,
		SslKey:      params.SslKey,
		Path:        params.Path,
		LogPath:     params.LogPath,
		WebLogPath:  params.WebLogPath,
		RDB:         params.RDB,
		HttpAuth:    params.HttpAuth,
		DontKill:    params.DontKill,
		UI:          params.UI,
		TorrentsDir: params.TorrentsDir,
		TorrentAddr: params.TorrentAddr,
		PubIPv4:     params.PubIPv4,
		PubIPv6:     params.PubIPv6,
		SearchWA:    params.SearchWA,
		StreamWA:    params.StreamWA,
		MaxSize:     params.MaxSize,
		TGToken:     params.TGToken,
		FusePath:    params.FusePath,
		WebDAV:      params.WebDAV,
		ProxyURL:    params.ProxyURL,
		ProxyMode:   params.ProxyMode,
		ForceHTTPS:  params.ForceHTTPS,
	}

	if params.ProxyURL != "" {
		log.TLogln("Proxy configured from CLI; mode:", settings.Args.ProxyMode)
	}

	if params.ForceHTTPS && !params.Ssl {
		log.TLogln("Error: --force-https requires --ssl")
		os.Exit(1)
	}

	if params.Service == "run" {
		if err := runWindowsService(); err != nil {
			log.TLogln("Flow service error:", err)
			os.Exit(1)
		}
		log.Close()
		return
	}
	server.Start()
	log.TLogln(server.WaitServer())
	log.Close()
	time.Sleep(time.Second * 3)
	os.Exit(0)
}

// watchTDir autoloads .torrent files dropped into dir, event-driven via fsnotify
// (upstream #753/#771 parity; the fork's engine swap had temporarily reverted it
// to a 5s polling loop). One anonymous function per file isolates panics; the DB
// save happens BEFORE the file is removed, so a crash between the two can only
// re-process, never lose the torrent. processTorrentFile is shared with the
// startup sweep below.
func watchTDir(dir string) {
	path, err := filepath.Abs(dir)
	if err != nil {
		path = dir
	}

	// fsnotify only reports files that appear AFTER the watch starts — sweep the
	// ones already sitting in the dir once (the polling version handled those).
	if files, err := os.ReadDir(path); err == nil {
		for _, file := range files {
			processTorrentFile(filepath.Join(path, file.Name()))
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.TLogln("Error creating watcher:", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(path); err != nil {
		log.TLogln("Error adding directory to watcher:", err)
		return
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Process only file creation or modification events.
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}
			processTorrentFile(event.Name)
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.TLogln("Watcher error:", err)
		}
	}
}

// processTorrentFile adds one autoload .torrent to the DB and removes the file.
// Recovers panics so one broken file can't kill the watcher goroutine.
func processTorrentFile(filename string) {
	defer func() {
		if r := recover(); r != nil {
			log.TLogln("Recovered from panic in watchTDir:", r)
		}
	}()
	if strings.ToLower(filepath.Ext(filename)) != ".torrent" {
		return
	}

	sp, err := torr.ParseTorrentFilePath(filename)
	if err != nil {
		log.TLogln("Error parse file name:", err)
		return
	}

	tor, err := torr.AddTorrent(sp, "", "", "", "")
	if err != nil {
		log.TLogln("Error parse torrent file:", err)
		return
	}

	if !tor.GotInfo() {
		log.TLogln("Error get info from torrent")
		return
	}

	if tor.Title == "" {
		tor.Title = tor.Name()
	}

	// Long database operation first; the file is removed only after the save, so
	// repeated filesystem events can at worst re-process an already-saved torrent.
	torr.SaveTorrentToDB(tor)
	tor.Close()

	if err := os.Remove(filename); err != nil {
		log.TLogln("Error removing torrent file:", err)
	}
}
