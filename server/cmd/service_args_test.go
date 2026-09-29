package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestServicePathsSurviveDifferentWorkingDirectory(t *testing.T) {
	log, _ := filepath.Abs("logs/server.log")
	cert, _ := filepath.Abs("tls/cert.pem")
	got, err := serviceRunArgs([]string{"--service=install", "-d", "state", "-l", "logs/server.log", "--sslcert=tls/cert.pem", "--port", "8090"}, "absolute-state")
	want := []string{"--service", "run", "-l", log, "--sslcert", cert, "--port", "8090", "--path", "absolute-state"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("args=%v err=%v", got, err)
	}
	if _, err := serviceRunArgs([]string{"--logpath"}, "state"); err == nil {
		t.Fatal("missing value accepted")
	}
}
