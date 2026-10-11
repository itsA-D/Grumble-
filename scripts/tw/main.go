// Command tw fetches, verifies and runs the pinned Tailwind CSS standalone CLI.
//
// It maps the host OS/arch to a release asset, downloads it once into .cache,
// verifies its SHA-256 against scripts/tailwind.lock, then execs it with the
// arguments it was given. This is one cross-platform code path for Windows,
// Linux CI and the Docker build, with no Node in the toolchain.
//
// Usage (from the repo root):
//
//	go run ./scripts/tw -i web/src/app.css -o web/static/app.css --minify
//
// Environment overrides: TW_OS, TW_ARCH, TW_MUSL (1/0), TW_LOCK.
package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	releaseBaseURL  = "https://github.com/tailwindlabs/tailwindcss/releases/download"
	downloadTimeout = 5 * time.Minute
)

type asset struct {
	name string
	sum  string
}

type lock struct {
	version string
	assets  map[string]asset
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "tw:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	root, err := repoRoot()
	if err != nil {
		return err
	}

	lockPath := os.Getenv("TW_LOCK")
	if lockPath == "" {
		lockPath = filepath.Join(root, "scripts", "tailwind.lock")
	}
	lk, err := readLock(lockPath)
	if err != nil {
		return err
	}

	key := platformKey()
	a, ok := lk.assets[key]
	if !ok {
		return fmt.Errorf("no pinned Tailwind asset for %q; add it to %s", key, lockPath)
	}

	bin, err := ensure(ctx, root, lk.version, a)
	if err != nil {
		return err
	}
	return execCLI(ctx, bin, os.Args[1:])
}

// repoRoot walks up from the working directory to the module root.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found in any parent directory")
		}
		dir = parent
	}
}

func platformKey() string {
	goos := envOr("TW_OS", runtime.GOOS)
	goarch := envOr("TW_ARCH", runtime.GOARCH)
	key := goos + "/" + goarch
	if goos == "linux" && isMusl() {
		key += "/musl"
	}
	return key
}

func isMusl() bool {
	switch os.Getenv("TW_MUSL") {
	case "1":
		return true
	case "0":
		return false
	}
	_, err := os.Stat("/etc/alpine-release")
	return err == nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func readLock(path string) (lock, error) {
	f, err := os.Open(path)
	if err != nil {
		return lock{}, fmt.Errorf("open lock file: %w", err)
	}
	defer func() { _ = f.Close() }()

	lk := lock{assets: make(map[string]asset)}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if fields[0] == "version" && len(fields) == 2 {
			lk.version = fields[1]
			continue
		}
		if len(fields) != 3 {
			return lock{}, fmt.Errorf("malformed lock line: %q", line)
		}
		lk.assets[fields[0]] = asset{name: fields[1], sum: strings.ToLower(fields[2])}
	}
	if err := sc.Err(); err != nil {
		return lock{}, err
	}
	if lk.version == "" {
		return lock{}, errors.New("lock file is missing the version line")
	}
	return lk, nil
}

// ensure returns the path to a verified, cached copy of the asset, downloading
// it if necessary.
func ensure(ctx context.Context, root, version string, a asset) (string, error) {
	dir := filepath.Join(root, ".cache", "tailwind", version)
	bin := filepath.Join(dir, a.name)

	if ok, err := verify(bin, a.sum); err == nil && ok {
		return bin, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/%s/%s", releaseBaseURL, version, a.name)
	if err := download(ctx, url, bin, a.sum); err != nil {
		return "", err
	}
	return bin, nil
}

func verify(path, want string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	return hex.EncodeToString(h.Sum(nil)) == want, nil
}

func download(ctx context.Context, url, dest, want string) error {
	client := &http.Client{Timeout: downloadTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}

	tmp := dest + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, h), resp.Body); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		_ = os.Remove(tmp)
		return fmt.Errorf("checksum mismatch for %s: got %s want %s", dest, got, want)
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func execCLI(ctx context.Context, bin string, args []string) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		return err
	}
	return nil
}
