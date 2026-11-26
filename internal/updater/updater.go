package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	client := &http.Client{Timeout: 20 * time.Second}
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