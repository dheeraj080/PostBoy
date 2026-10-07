// Headless collection runner.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dheeraj080/PostBoy/internal/collection"
	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/httpclient"
	"github.com/dheeraj080/PostBoy/internal/secrets"
)

func runUsage() {
	fmt.Fprintf(os.Stderr, `postboy run [flags] COLLECTION

Execute every request in a saved collection.

Flags:
`)
	fmt.Fprintf(os.Stderr, `	--data-dir DIR	config directory (default ~/.config/postboy)
`)
	fmt.Fprintf(os.Stderr, `	--env NAME	environment to use (default: active from config)
`)
	fmt.Fprintf(os.Stderr, `	--continue	keep after the first failed request
`)
	fmt.Fprintf(os.Stderr, `	--fast	fail on first error or non-2xx/3xx response
`)
}

// runCmd executes a collection headlessly and returns a process exit code.
func runCmd(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataDir := fs.String("data-dir", "", "config directory")
	envName := fs.String("env", "", "environment to use")
	cont := fs.Bool("continue", false, "continue after a failed request")
	verbose := fs.Bool("verbose", false, "print more details")
	err := fs.Parse(args)
	if err != nil || fs.NArg() != 1 {
		runUsage()
		return 2
	}

	collName := fs.Arg(0)
	dir := *dataDir
	if dir == "" {
		dir = config.Dir()
	}

	cfg, err := config.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		return 1
	}
	cols, err := collection.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "collections: %v\n", err)
		return 1
	}
	if collName == "" {
		fmt.Fprintln(os.Stderr, "missing collection name")
		runUsage()
		return 2
	}

	var coll *collection.Collection
	for i := range cols {
		if namesEqual(cols[i].Name, collName) || namesEqual(cols[i].ID, collName) {
			coll = &cols[i]
			break
		}
	}
	if coll == nil {
		fmt.Fprintf(os.Stderr, "collection %q not found. Available:\n", collName)
		for _, c := range cols {
			fmt.Fprintf(os.Stderr, "  - %s\n", c.Name)
		}
		return 2
	}

	envIdx := cfg.ActiveEnv
	if *envName != "" {
		found := false
		for i, e := range cfg.Environments {
			if e.Name == *envName {
				envIdx = i
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "environment %q not found\n", *envName)
			return 2
		}
	}
	env := cfg.Environments[envIdx].Vars
	sec := secrets.New()
	client := httpclient.NewWithConfig(cfg)

	var failures int
	for i, r := range coll.Requests {
		req := httpclient.FromConfig(r, env, sec)
		timeout := 15 * time.Second
		if r.TimeoutSeconds > 0 {
			timeout = time.Duration(r.TimeoutSeconds) * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		res, err := client.Do(ctx, req)
		cancel()

		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ #%d %s -> error: %v\n", i+1, r.Name, err)
			failures++
			if !*cont {
				return 1
			}
			continue
		}
		status := "✓"
		if res.StatusCode >= 400 {
			status = "✗"
			failures++
			if !*cont {
				fmt.Printf("%s #%d %s -> %d in %s\n", status, i+1, r.Name, res.StatusCode, res.Duration.Round(time.Millisecond))
				return 1
			}
		} else {
			fmt.Printf("%s #%d %s -> %d in %s\n", status, i+1, r.Name, res.StatusCode, res.Duration.Round(time.Millisecond))
		}
		if *verbose {
			fmt.Printf("  URL: %s\n", res.URL)
			if len(res.Body) > 0 {
				body := string(res.Body)
				if len(body) > 400 {
					body = body[:400] + "..."
				}
				fmt.Printf("  Body: %s\n", strings.ReplaceAll(body, "\n", "\\n"))
			}
		}
	}
	if failures > 0 {
		fmt.Fprintf(os.Stderr, "%d request(s) failed\n", failures)
		return 1
	}
	fmt.Printf("✓ %d request(s) completed\n", len(coll.Requests))
	return 0
}

func namesEqual(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
