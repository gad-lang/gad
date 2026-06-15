//go:build !noide

package main

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	cc "github.com/moisespsena-go/command-context"

	webapp "github.com/gad-lang/gad/web/app"
	"github.com/gad-lang/gad/web/ide"
)

// ideApp holds the bundled single-file IDE served when no external --static
// directory is provided. It talks to the /api/ide/* endpoints.
//
//go:embed ideapp
var ideApp embed.FS

func init() { registerCommand("ide", ideCommand) }

// ideCommand is `gad ide [flags] [PATH]`: it starts a local web IDE rooted at
// PATH (a directory, or a single file to edit; defaults to the current
// directory) and opens it in the browser.
func ideCommand() *cc.Command {
	var (
		addr   *string
		static *string
		noOpen *bool
	)
	return &cc.Command{
		Name:  "ide",
		Usage: "[flags] [PATH]",
		Description: "Start a local web IDE for Gad.\n" +
			"\nPATH is a workspace directory, or a single file to edit; it defaults to the\n" +
			"current directory. The IDE offers a file tree, multi-file tabs, formatting,\n" +
			"running and debugging, with formatter and layout settings stored in .gad.yaml.",
		New: func(ctx *cc.CommandContext) error {
			addr = ctx.Flags().String("addr", "0.0.0.0:17000", "listen address (host:port); if the port is busy the next free port is used")
			static = ctx.Flags().String("static", "", "serve a pre-built web app from this directory instead of the bundled UI")
			noOpen = ctx.Flags().Bool("no-open", false, "do not open the browser automatically")
			return nil
		},
		Run: func(ctx *cc.CommandContext) error {
			path := "."
			if len(ctx.Args) > 0 {
				path = ctx.Args[0]
			}
			srv, err := ide.New(path)
			if err != nil {
				return fmt.Errorf("ide: %w", err)
			}

			handler := srv.Handler()
			ui := "bundled UI"
			switch {
			case *static != "":
				// Serve a pre-built app directory at the site root.
				srv.Static = *static
				handler = srv.Handler()
				ui = "static " + *static
			default:
				if assets, ok := webapp.Assets(); ok {
					// Production build: serve the embedded React app (SPA).
					handler = withAppFallback(srv.Handler(), spaFSServer(assets))
					ui = "embedded React UI"
				} else {
					// Development: serve the bundled, build-free UI.
					sub, err := fs.Sub(ideApp, "ideapp")
					if err != nil {
						return err
					}
					handler = withAppFallback(srv.Handler(), http.FileServer(http.FS(sub)))
				}
			}

			ln, err := listenWithFallback(*addr)
			if err != nil {
				return fmt.Errorf("ide: listen %s: %w", *addr, err)
			}
			url := "http://" + browserHost(ln.Addr()) + "/"
			fmt.Fprintf(ctx.Out, "Gad IDE for %s (%s)\nopen %s\n", srv.Root, ui, url)
			if !*noOpen {
				go openBrowser(url)
			}
			return http.Serve(ln, handler)
		},
	}
}

// browserHost turns a listen address into a URL host a browser can reach,
// mapping wildcard binds (0.0.0.0, ::, empty) to 127.0.0.1.
func browserHost(addr net.Addr) string {
	host, port, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	return net.JoinHostPort(host, port)
}

// listenWithFallback listens on addr; if its port is already in use it scans
// forward for the next free port (up to maxPortScan tries), printing an alert to
// STDERR when it falls back. A ":0" port (pick-any) is honoured as-is.
func listenWithFallback(addr string) (net.Listener, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return net.Listen("tcp", addr) // let net report the malformed address
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port == 0 {
		return net.Listen("tcp", addr)
	}

	const maxPortScan = 100
	first := port
	for ; port < first+maxPortScan; port++ {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err == nil {
			if port != first {
				fmt.Fprintf(os.Stderr, "ALERT: port %d is busy; using next free port %d\n", first, port)
			}
			return ln, nil
		}
		if !errors.Is(err, syscall.EADDRINUSE) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("no free port in range %d-%d", first, first+maxPortScan-1)
}

// spaFSServer serves a built single-page app from fsys, falling back to
// index.html for unknown (client-routed) paths.
func spaFSServer(fsys fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(fsys))
	index, _ := fs.ReadFile(fsys, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p != "" {
			if info, err := fs.Stat(fsys, p); err == nil && !info.IsDir() {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
}

// withAppFallback serves the API via primary and everything else (the bundled
// single-page app) via static.
func withAppFallback(primary, static http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
			primary.ServeHTTP(w, r)
			return
		}
		static.ServeHTTP(w, r)
	})
}

// openBrowser tries to open url in the default browser, ignoring failures (the
// URL is also printed to stdout).
func openBrowser(url string) {
	time.Sleep(300 * time.Millisecond)
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "cmd", []string{"/c", "start"}
	default:
		cmd = "xdg-open"
	}
	args = append(args, url)
	if err := exec.Command(cmd, args...).Start(); err != nil {
		fmt.Fprintln(os.Stderr, "could not open browser:", err)
	}
}
