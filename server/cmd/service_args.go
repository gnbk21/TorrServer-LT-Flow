package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SCM runs with a different working directory. Resolve file arguments at
// installation time so relative paths keep their command-line meaning.
func serviceRunArgs(input []string, stateDir string) ([]string, error) {
	pathFlags := map[string]bool{
		"--logpath": true, "-l": true, "--weblogpath": true, "-w": true,
		"--torrentsdir": true, "-t": true, "--sslcert": true, "--sslkey": true,
		"--fusepath": true, "-f": true,
	}
	out := []string{"--service", "run"}
	for i := 0; i < len(input); i++ {
		flag, value, inline := strings.Cut(input[i], "=")
		if flag == "--service" || flag == "--path" || flag == "-d" || pathFlags[flag] {
			if !inline {
				i++
				if i >= len(input) {
					return nil, fmt.Errorf("missing value for %s", flag)
				}
				value = input[i]
			}
			if pathFlags[flag] {
				if value != "" {
					var err error
					value, err = filepath.Abs(value)
					if err != nil {
						return nil, err
					}
				}
				out = append(out, flag, value)
			}
			continue
		}
		out = append(out, input[i])
	}
	return append(out, "--path", stateDir), nil
}
