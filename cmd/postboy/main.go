// Command postboy is a keyboard-driven terminal HTTP/API client.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/tui"
)

// Set at build time via -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "run" {
		os.Exit(runCmd(os.Args[2:]))
	}

	showVersion := flag.Bool("version", false, "print version information and exit")
	demo := flag.Bool("demo", false, "start with a canned request/response pair (no config, keychain or network)")
	dataDir := flag.String("data-dir", "", "directory for config and history (default "+config.Dir()+")")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `PostBoy - terminal HTTP client

Usage:
  postboy [flags]                 start the TUI
  postboy [flags] import <file>   import a Postman v2.1 collection
  postboy run <collection>        execute a saved collection headlessly

Flags:
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("postboy %s (commit %s, built %s)\n", version, commit, date)
		return
	}

	dir := *dataDir
	if dir == "" {
		dir = config.Dir()
	}
	if *demo {
		// Demo mode never touches the filesystem.
		dir = ""
	}

	switch flag.Arg(0) {
	case "":
	case "import":
		if flag.NArg() != 2 {
			fmt.Fprintln(os.Stderr, "usage: postboy import <collection.json>")
			os.Exit(2)
		}
		summary, err := tui.ImportFile(dir, flag.Arg(1))
		if err != nil {
			fmt.Fprintf(os.Stderr, "import failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(summary)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", flag.Arg(0))
		flag.Usage()
		os.Exit(2)
	}

	p := tea.NewProgram(tui.New(tui.Options{Dir: dir, Version: version, Demo: *demo}), tea.WithAltScreen())
	final, err := p.Run()
	if m, ok := final.(tui.Model); ok {
		m.Close()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}
