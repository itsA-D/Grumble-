// Command devopen waits for the local dev server to accept connections and
// then opens the site in the default browser.
//
// `task dev` launches it in the background so the browser opens once, after the
// first Air build has finished booting the server. It is not used in production.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	page := flag.String("url", defaultURL(), "page to open")
	timeout := flag.Duration("timeout", 90*time.Second, "how long to wait for the server")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *page, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, "devopen:", err)
		os.Exit(1)
	}
}

// defaultURL mirrors the dev defaults in .air.toml and config.Load.
func defaultURL() string {
	if site := os.Getenv("SITE_URL"); site != "" {
		return site
	}
	return "http://localhost:3000"
}

func run(ctx context.Context, page string, timeout time.Duration) error {
	addr, err := listenAddr(page)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := waitForServer(ctx, addr); err != nil {
		return err
	}
	if err := openPage(ctx, page); err != nil {
		return err
	}
	fmt.Println("devopen: opened", page)
	return nil
}

// listenAddr turns the page URL into the TCP address the server listens on.
func listenAddr(page string) (string, error) {
	u, err := url.Parse(page)
	if err != nil {
		return "", fmt.Errorf("parse %q: %w", page, err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("parse %q: missing host", page)
	}
	if u.Port() != "" {
		return u.Host, nil
	}
	port := "80"
	if u.Scheme == "https" {
		port = "443"
	}
	return net.JoinHostPort(u.Hostname(), port), nil
}

// waitForServer polls until the address accepts a TCP connection or ctx ends.
func waitForServer(ctx context.Context, addr string) error {
	dialer := net.Dialer{Timeout: time.Second}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("server at %s not ready: %w", addr, ctx.Err())
		case <-ticker.C:
		}
	}
}

// openPage is a variable so tests can observe it without launching a browser.
var openPage = openInBrowser

func openInBrowser(ctx context.Context, page string) error {
	// The launcher must outlive run's timeout context; otherwise the deferred
	// cancel would kill the process before it opens the page.
	ctx = context.WithoutCancel(ctx)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// rundll32 opens the URL with the user's default browser without
		// starting a console window.
		cmd = exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", page)
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", page)
	default:
		cmd = exec.CommandContext(ctx, "xdg-open", page)
	}
	return cmd.Start()
}
