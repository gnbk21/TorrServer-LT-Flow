package server

import (
	"os"
	"server/cachefs"
	"server/flow"
	"strconv"

	"server/tgbot"

	"server/log"
	"server/netbind"
	"server/settings"
	"server/web"
)

func Start() {
	settings.InitSets(settings.Args.RDB, settings.Args.SearchWA, settings.Args.StreamWA)
	// https checks
	if settings.Args.Ssl {
		// set settings ssl enabled
		settings.Ssl = settings.Args.Ssl
		if settings.Args.SslPort == "" {
			dbSSlPort := strconv.Itoa(settings.BTsets().SslPort)
			if dbSSlPort != "0" {
				settings.Args.SslPort = dbSSlPort
			} else {
				settings.Args.SslPort = settings.DefaultSslPort
			}
		} else { // Apply the startup override to the in-memory settings snapshot.
			dbSSlPort, err := strconv.Atoi(settings.Args.SslPort)
			if err == nil {
				next := settings.CloneSettings(settings.BTsets())
				next.SslPort = dbSSlPort
				settings.StoreBTsets(next)
			}
		}
		// Pass a partial CLI pair through to EnsureCert so it fails clearly;
		// silently ignoring it could generate a different certificate instead.
		if settings.Args.SslCert != "" || settings.Args.SslKey != "" {
			next := settings.CloneSettings(settings.BTsets())
			next.SslCert, next.SslKey = settings.Args.SslCert, settings.Args.SslKey
			settings.StoreBTsets(next)
		}
		log.TLogln("Check web ssl port", settings.Args.SslPort)
		if err := netbind.CheckPort(settings.Args.IPs, settings.Args.SslPort); err != nil {
			log.Event("ERROR", "HTTPS", "Cannot bind port "+settings.Args.SslPort+": "+err.Error()+". Choose another --sslport or stop the other listener.")
			log.Close()
			os.Exit(1)
		}
	}
	// http checks
	if settings.Args.Port == "" {
		settings.Args.Port = settings.DefaultPort
	}

	if settings.HTTPEnabled() {
		log.TLogln("Check web port", settings.Args.Port, "on", netbind.Normalize(settings.Args.IPs))
		if err := netbind.CheckPort(settings.Args.IPs, settings.Args.Port); err != nil {
			log.Event("ERROR", "HTTP", "Cannot bind port "+settings.Args.Port+": "+err.Error()+". Choose another --port or stop the other listener.")
			log.Close()
			os.Exit(1)
		}
	}
	// remove old disk caches
	go cleanCache()
	// set settings http and https ports. Start web server.
	settings.Port = settings.Args.Port
	settings.SslPort = settings.Args.SslPort
	settings.IPs = settings.Args.IPs

	web.Start()
	if settings.Args.TGToken != "" && web.ListenersReady() {
		if err := tgbot.Start(settings.Args.TGToken); err != nil {
			log.TLogln("tg bot start failed", err)
		}
	}
}

func cleanCache() {
	if !settings.BTsets().UseDisk || settings.BTsets().TorrentsSavePath == "/" || settings.BTsets().TorrentsSavePath == "" {
		return
	}

	config := settings.BTsets()
	root, err := cachefs.Open(config.TorrentsSavePath, false)
	if err != nil {
		return
	}
	defer root.Close()
	directory, err := root.Open(".")
	if err != nil {
		return
	}
	dirs, err := directory.ReadDir(-1)
	directory.Close()
	if err != nil {
		return
	}

	torrs := settings.ListTorrent()

	log.TLogln("Remove unused cache in dir:", settings.BTsets().TorrentsSavePath)
	keep := map[string]bool{}
	for _, d := range dirs {
		if !flow.ValidInfoHash(d.Name()) {
			// Not a hash
			continue
		}

		if !config.RemoveCacheOnDrop {
			keep[d.Name()] = true
			for _, t := range torrs {
				if d.IsDir() && t.TorrentSpec != nil && d.Name() == t.TorrentSpec.InfoHash {
					keep[d.Name()] = false
					break
				}
			}
			for hash, del := range keep {
				if del && hash == d.Name() {
					log.TLogln("Remove unused cache:", d.Name())
					if err := cachefs.RemoveFlat(root, d.Name()); err != nil {
						log.TLogln("Cache cleanup retained unsafe or busy entry:", d.Name(), err)
					}
				}
			}
		} else {
			if d.IsDir() {
				log.TLogln("Remove unused cache:", d.Name())
				if err := cachefs.RemoveFlat(root, d.Name()); err != nil {
					log.TLogln("Cache cleanup retained unsafe or busy entry:", d.Name(), err)
				}
			}
		}
	}
}

func WaitServer() string {
	err := web.Wait()
	if err != nil {
		return err.Error()
	}
	return ""
}

func Stop() {
	web.Stop()
	settings.CloseDB()
}
