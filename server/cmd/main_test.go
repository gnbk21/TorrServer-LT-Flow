package main

import (
	"testing"

	"github.com/alexflint/go-arg"
)

func TestIPsSeparateFlag(t *testing.T) {
	var params args
	p, err := arg.NewParser(arg.Config{}, &params)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Parse([]string{"--ip", "127.0.0.1", "--ip", "192.168.1.100"}); err != nil {
		t.Fatal(err)
	}
	want := []string{"127.0.0.1", "192.168.1.100"}
	if len(params.IPs) != len(want) {
		t.Fatalf("IPs = %#v, want %#v", params.IPs, want)
	}
	for i := range want {
		if params.IPs[i] != want[i] {
			t.Fatalf("IPs = %#v, want %#v", params.IPs, want)
		}
	}
}

func TestConsoleFlagDefaultsAndDisable(t *testing.T) {
	for _, tc := range []struct {
		argv     []string
		mode     string
		interval int
	}{
		{nil, "auto", 30},
		{[]string{"--console", "plain", "--console-interval", "0"}, "plain", 0},
		{[]string{"--console=off", "--console-interval=60"}, "off", 60},
	} {
		var parsed args
		parser, err := arg.NewParser(arg.Config{}, &parsed)
		if err != nil {
			t.Fatal(err)
		}
		if err := parser.Parse(tc.argv); err != nil {
			t.Fatal(err)
		}
		if parsed.Console != tc.mode || parsed.ConsoleInterval != tc.interval {
			t.Fatalf("%v: %s/%d", tc.argv, parsed.Console, parsed.ConsoleInterval)
		}
	}
}
