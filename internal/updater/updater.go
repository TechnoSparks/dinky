package updater

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/blang/semver"
)

// Release describes the latest published GitHub release.
type Release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

const repository = "TechnoSparks/dinky"

// LatestRelease fetches the latest published release from GitHub.
func LatestRelease(ctx context.Context) (Release, error) {
	url := "https://api.github.com/repos/" + repository + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			Resolver:  termuxSafeResolver(),
		}).DialContext,
	}
	if caPath := caBundlePath(); caPath != "" {
		if pool, err := loadCertPool(caPath); err == nil {
			transport.TLSClientConfig = &tls.Config{RootCAs: pool}
		}
	}
	client := &http.Client{Timeout: 20 * time.Second, Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return Release{}, fmt.Errorf("GitHub API: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return Release{}, err
	}
	if rel.TagName == "" {
		return Release{}, fmt.Errorf("no release tag found")
	}
	return rel, nil
}

// IsNewer reports whether latestTag is newer than currentVersion.
// Dev/unknown/untagged builds are treated as older, so an update is offered.
func IsNewer(currentVersion, latestTag string) bool {
	latest, ok := parseVersion(latestTag)
	if !ok {
		return false
	}
	current, ok := parseVersion(currentVersion)
	if !ok {
		return true
	}
	return latest.GT(current)
}

func parseVersion(s string) (semver.Version, bool) {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, " ("); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimPrefix(s, "v")
	v, err := semver.Make(s)
	if err != nil {
		return semver.Version{}, false
	}
	return v, true
}

// caBundlePath looks for a CA certificate bundle in locations used by Termux
// and other minimal environments, where Go's default system pool may be empty.
func caBundlePath() string {
	candidates := []string{}
	if v := os.Getenv("SSL_CERT_FILE"); v != "" {
		candidates = append(candidates, v)
	}
	if prefix := os.Getenv("PREFIX"); prefix != "" {
		candidates = append(candidates, prefix+"/etc/tls/cert.pem")
	}
	candidates = append(candidates,
		"/data/data/com.termux/files/usr/etc/tls/cert.pem",
		"/etc/ssl/certs/ca-certificates.crt",
		"/etc/pki/tls/certs/ca-bundle.crt",
	)
	for _, p := range candidates {
		if p == "" {
			continue
		}
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

func loadCertPool(path string) (*x509.CertPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("no certificates found in %s", path)
	}
	return pool, nil
}

// termuxSafeResolver returns a resolver that talks directly to public DNS
// servers. On Termux the system resolver (127.0.0.1/::1) often refuses
// connections, and a UDP dial to it can still "succeed" (UDP is
// connectionless) only to fail on read, so we bypass it entirely.
func termuxSafeResolver() *net.Resolver {
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := &net.Dialer{Timeout: 5 * time.Second}
			for _, fb := range []string{"8.8.8.8:53", "1.1.1.1:53"} {
				c, e := d.DialContext(ctx, network, fb)
				if e == nil {
					return c, nil
				}
			}
			// Last resort: try the system resolver as-is.
			return d.DialContext(ctx, network, address)
		},
	}
}