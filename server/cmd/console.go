package main

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	"server/console"
	"server/diagnostics"
	"server/log"
	"server/lt"
	"server/settings"
	"server/torr"
	"server/torr/storage/torrstor"
	"server/version"
	"server/web"
)

func consoleStartup() {
	scheme, port := "http", settings.Port
	if settings.Ssl {
		scheme, port = "https", settings.SslPort
	}
	access := []string{}
	for _, u := range console.AccessURLs(settings.IPs, web.GetLocalIps(), scheme, port) {
		access = append(access, fmt.Sprintf("%-14s %s", u.Label+":", u.URL))
	}
	access = append(access, "Phone: use a reachable address on the same network in Lampa.")
	if settings.Ssl {
		access = append(access, "HTTPS: clients must trust your configured or generated certificate.")
	}
	if settings.Args.ForceHTTPS {
		access = append(access, "HTTP redirects to HTTPS.")
	}
	sets := settings.BTsets()
	cache := "RAM"
	if sets.UseDisk {
		cache = "disk"
	}
	flow := "disabled"
	if sets.Flow != nil && sets.Flow.Enabled {
		flow = fmt.Sprintf("enabled | swarm %s | warm retention %ds", sets.Flow.SwarmProfile, sets.Flow.WarmSessionTimeoutSec)
	}
	auth := "disabled"
	if settings.HttpAuth {
		auth = "enabled"
	}
	db := "read/write"
	if params.RDB {
		db = "read-only"
	}
	state, err := filepath.Abs(settings.Path)
	if err != nil {
		state = settings.Path
	}
	startup := diagnostics.Startup()
	log.ConsolePanel("TORRSERVER FLOW | "+version.Version, []console.Section{
		{Title: "CONNECTION", Rows: access},
		{Title: "CONFIGURATION", Rows: []string{
			"State: " + state,
			fmt.Sprintf("Cache: %s | configured per-torrent budget %s", cache, console.Bytes(uint64(max(0, sets.CacheSize)))),
			"Flow: " + flow,
			"Authentication: " + auth + " | database: " + db,
		}},
		{Title: "RUNTIME", Rows: []string{
			fmt.Sprintf("libtorrent %s | %s | %s/%s | %d CPUs", lt.Version(), runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU()),
			fmt.Sprintf("Listeners ready in %dms | engine ready in %dms", startup.ListenerReadyMs, startup.EngineReadyMs),
		}},
		{Title: "QUICK HELP", Rows: []string{
			"Web UI: Dashboard for live speed, peers and playback buffer details.",
			"Settings > Advanced > Shut down stops the server cleanly.",
			"--help: all options | --doctor: local prerequisite checks (server stopped).",
			"--console plain: no colors | --console off: legacy logs",
			"--console-interval 0: no periodic status (default: 30 seconds)",
			"LAN candidates may include VPN adapters. No tracker reply yet is normal when idle.",
		}},
	})
}

func consoleStatusKey() string {
	startup := diagnostics.Startup()
	network := web.BTS.FlowNetworkStatus()
	return fmt.Sprintf("%t/%s/%s/%d/%t", startup.EngineReady, network.State, network.Connectivity, torr.GetActiveStreams(), torr.FlowIsPaused())
}

func consoleStatus() string {
	startup := diagnostics.Startup()
	engine := "ready"
	if !startup.EngineReady {
		engine = "starting/stopped"
	}
	network := web.BTS.FlowNetworkStatus()
	streams := torr.GetActiveStreams()
	paused := torr.FlowIsPaused()
	cache := torrstor.Global().Allocations()
	memory := diagnostics.Memory()
	rss := "unavailable"
	if memory.RSSAvailable {
		rss = console.Bytes(memory.RSSBytes)
	}
	message := fmt.Sprintf("Uptime %s | engine %s | addresses %s | trackers %s\nStreams %d | Flow paused %t | cache resident %s (%d active, %d warm)\nProcess RSS %s | Go heap %s | goroutines %d",
		time.Since(startup.StartedAt).Truncate(time.Second), engine, network.State, network.Connectivity,
		streams, paused, console.Bytes(uint64(max(0, cache.ResidentBytes))), cache.ActiveCaches, cache.WarmCaches,
		rss, console.Bytes(memory.GoHeapBytes), memory.Goroutines)
	return message
}

func startConsoleStatus(interval int) func() {
	if interval == 0 {
		return func() {}
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		console.RunStatus(ctx, time.Duration(interval)*time.Second, consoleStatusKey, consoleStatus, func(s string) { log.Event("INFO", "Status", s) })
	}()
	return func() { cancel(); <-done }
}
