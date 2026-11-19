// Package selfupdate replaces the running stamper binary with the latest
// GitHub release. The release source is set at build time:
//
//	go build -ldflags "-X github.com/pipejesus/stamper/internal/selfupdate.Repo=OWNER/stamper"
//
// Releases must attach raw binaries named stamper_<GOOS>_<GOARCH>.
package selfupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Repo is the GitHub "owner/name" releases come from; empty disables updates.
var Repo = "pipejesus/stamper"

// ErrNotConfigured means this build has no release source.
var ErrNotConfigured = errors.New("this stamper build has no release source (built without -X github.com/pipejesus/stamper/internal/selfupdate.Repo=OWNER/REPO); rebuild from source to update")

// Release is the subset of the GitHub release API stamper needs.
type Release struct {
	Tag    string `json:"tag_name"`
	Assets []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// AssetName is the binary name expected for this platform.
func AssetName() string { return fmt.Sprintf("stamper_%s_%s", runtime.GOOS, runtime.GOARCH) }

// Latest fetches the latest release.
func Latest(ctx context.Context) (*Release, error) {
	if Repo == "" {
		return nil, ErrNotConfigured
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+Repo+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "stamper-cli")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases: %s", resp.Status)
	}
	var rel Release
	return &rel, json.NewDecoder(resp.Body).Decode(&rel)
}

// Newer reports whether release tag a is newer than version b (semver-ish,
// leading "v" optional; non-numeric builds such as "dev" are always older).
func Newer(a, b string) bool {
	pa, pb := parse(a), parse(b)
	if pb == nil {
		return pa != nil
	}
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func parse(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, _, _ = strings.Cut(v, "-")
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return nil
	}
	out := make([]int, 3)
	for i := 0; i < 3 && i < len(parts); i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return nil
		}
		out[i] = n
	}
	return out
}

// Apply downloads rel's binary for this platform and atomically replaces
// the running executable.
func Apply(ctx context.Context, rel *Release) (string, error) {
	var url string
	for _, a := range rel.Assets {
		if a.Name == AssetName() {
			url = a.URL
		}
	}
	if url == "" {
		return "", fmt.Errorf("release %s has no asset %s", rel.Tag, AssetName())
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if exe, err = filepath.EvalSymlinks(exe); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "stamper-cli")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: %s", url, resp.Status)
	}
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".stamper-update-*")
	if err != nil {
		return "", fmt.Errorf("cannot write next to %s (%w); reinstall with your package manager or sudo", exe, err)
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return "", err
	}
	return exe, os.Rename(tmp.Name(), exe)
}

// cache remembers the last release check so boot stays instant.
type cache struct {
	Checked time.Time `json:"checked"`
	Latest  string    `json:"latest"`
}

func cachePath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "stamper", "release.json")
}

// CachedLatest returns the latest known release tag, refreshing the cache at
// most once per day with a short timeout. Errors are swallowed: this only
// feeds a hint.
func CachedLatest(now time.Time) string {
	if Repo == "" {
		return ""
	}
	p := cachePath()
	var c cache
	if data, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(data, &c)
	}
	if now.Sub(c.Checked) < 24*time.Hour {
		return c.Latest
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	c.Checked = now
	if rel, err := Latest(ctx); err == nil {
		c.Latest = rel.Tag
	}
	if data, err := json.Marshal(c); err == nil && p != "" {
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, data, 0o644)
	}
	return c.Latest
}
