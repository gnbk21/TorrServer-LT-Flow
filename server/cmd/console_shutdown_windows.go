//go:build windows

package main

import (
	"os"
	"os/signal"
	"server"
	"server/torr"
	"syscall"
	"time"
)

func installConsoleShutdown(dontKill bool) {
	if dontKill || params.Service == "run" {
		return
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signals
		torr.RecordShutdownReason("SIGNAL")
		done := make(chan struct{})
		go func() { server.Stop(); close(done) }()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			torr.RecordShutdownReason("SHUTDOWN_TIMEOUT")
			os.Exit(1)
		}
	}()
}
