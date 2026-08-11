package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadEnvFile reads a KEY=VALUE .env file into the process environment. It splits on the
// first '=', so DSN values containing '=', '(', ')', '&' or '@' survive intact (the very
// thing that breaks `source .env`). Comments (#), blank lines and an optional `export `
// prefix are handled; surrounding quotes are stripped. Variables already set in the
// environment are left untouched, so an explicit `DATABASE_DSN=... go run ...` wins. A
// missing file is not an error - callers still work with env-only.
//
// The server gets its env from systemd/Makefile, but the one-off cmd/ tools are run by
// hand, so they call this to pick up the local .env before config.Load().
func LoadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if len(val) >= 2 && (val[0] == '"' && val[len(val)-1] == '"' || val[0] == '\'' && val[len(val)-1] == '\'') {
			val = val[1 : len(val)-1]
		}
		if _, ok := os.LookupEnv(key); ok {
			continue // don't override an already-set variable
		}
		if err := os.Setenv(key, val); err != nil {
			return err
		}
	}
	return sc.Err()
}
