package main

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestListenAddr(t *testing.T) {
	tests := []struct {
		page string
		want string
	}{
		{page: "http://localhost:3000", want: "localhost:3000"},
		{page: "http://127.0.0.1:8080/feed", want: "127.0.0.1:8080"},
		{page: "https://grumble.example", want: "grumble.example:443"},
		{page: "http://grumble.example", want: "grumble.example:80"},
	}
	for _, tt := range tests {
		t.Run(tt.page, func(t *testing.T) {
			got, err := listenAddr(tt.page)
			if err != nil {
				t.Fatalf("listenAddr(%q): %v", tt.page, err)
			}
			if got != tt.want {
				t.Fatalf("listenAddr(%q) = %q, want %q", tt.page, got, tt.want)
			}
		})
	}
}

func TestListenAddrRejectsMissingHost(t *testing.T) {
	if _, err := listenAddr("not a url"); err == nil {
		t.Fatal("listenAddr accepted a URL without a host")
	}
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return ln
}

func TestRunOpensPageOnceServerIsReady(t *testing.T) {
	ln := listen(t)
	defer func() { _ = ln.Close() }()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	oldOpen := openPage
	defer func() { openPage = oldOpen }()
	var opened string
	openPage = func(_ context.Context, page string) error {
		opened = page
		return nil
	}

	page := "http://" + ln.Addr().String()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := run(ctx, page, 2*time.Second); err != nil {
		t.Fatalf("run: %v", err)
	}
	if opened != page {
		t.Fatalf("opened %q, want %q", opened, page)
	}
}

func TestWaitForServer(t *testing.T) {
	ln := listen(t)
	defer func() { _ = ln.Close() }()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := waitForServer(ctx, ln.Addr().String()); err != nil {
		t.Fatalf("waitForServer: %v", err)
	}
}

func TestWaitForServerTimesOut(t *testing.T) {
	ln := listen(t)
	addr := ln.Addr().String()
	_ = ln.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	err := waitForServer(ctx, addr)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waitForServer error = %v, want context deadline exceeded", err)
	}
}
