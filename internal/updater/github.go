package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ARCOOON/arx-ca/internal/version"
)

const (
	githubOwner               = "ARCOOON"
	githubRepo                = "arx-ca"
	requestTimeout            = 2 * time.Minute
	releaseNotesClientTimeout = 5 * time.Second
)

// releaseLatestAPIURL is the GitHub REST endpoint for the newest stable release (overridable in tests).
var releaseLatestAPIURL = "https://api.github.com/repos/ARCOOON/arx-ca/releases/latest"

// releasesListAPIURL lists GitHub releases for channel resolution (overridable in tests).
var releasesListAPIURL = "https://api.github.com/repos/ARCOOON/arx-ca/releases"

type gitHubRelease struct {
	TagName    string        `json:"tag_name"`
	Body       string        `json:"body"`
	Prerelease bool          `json:"prerelease"`
	Assets     []gitHubAsset `json:"assets"`
}

var (
	releaseNotesMu           sync.RWMutex
	releaseNotesCacheVersion string
	releaseNotesCacheBody    string
	releaseNotesCacheErr     error
)

// releaseTagAPIURL builds the GitHub REST URL for a release tag (overridable in tests).
var releaseTagAPIURL = func(tag string) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", githubOwner, githubRepo, normalizeTag(tag))
}

type gitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func fetchReleaseForChannel(ctx context.Context, client *http.Client, channel string) (*gitHubRelease, error) {
	channel = strings.TrimSpace(strings.ToLower(channel))
	if channel == "" || channel == "main" || channel == "stable" {
		return fetchLatestStableRelease(ctx, client)
	}
	if channel == "develop" || channel == "beta" || channel == "pre" || channel == "prerelease" {
		return fetchLatestPrerelease(ctx, client)
	}
	return fetchReleaseByTag(ctx, client, channel)
}

func fetchLatestStableRelease(ctx context.Context, client *http.Client) (*gitHubRelease, error) {
	release, err := fetchReleaseURL(ctx, client, releaseLatestAPIURL)
	if err != nil {
		return nil, err
	}
	if release.Prerelease {
		return nil, fmt.Errorf("latest GitHub release %q is a prerelease; use channel develop for prereleases", release.TagName)
	}
	return release, nil
}

func fetchLatestPrerelease(ctx context.Context, client *http.Client) (*gitHubRelease, error) {
	releases, err := fetchReleasesPage(ctx, client, releasesListAPIURL)
	if err != nil {
		return nil, err
	}
	for i := range releases {
		if releases[i].Prerelease {
			return &releases[i], nil
		}
	}
	return fetchLatestStableRelease(ctx, client)
}

// FetchReleaseNotes retrieves markdown release notes for version from the GitHub Releases API.
// Results are cached in memory for the process lifetime to avoid repeated API calls.
func FetchReleaseNotes(version string) (string, error) {
	tag := normalizeTag(version)
	if tag == "" {
		return "", fmt.Errorf("version is required")
	}

	releaseNotesMu.RLock()
	if releaseNotesCacheVersion == tag {
		body, err := releaseNotesCacheBody, releaseNotesCacheErr
		releaseNotesMu.RUnlock()
		return body, err
	}
	releaseNotesMu.RUnlock()

	releaseNotesMu.Lock()
	defer releaseNotesMu.Unlock()

	if releaseNotesCacheVersion == tag {
		return releaseNotesCacheBody, releaseNotesCacheErr
	}

	ctx, cancel := context.WithTimeout(context.Background(), releaseNotesClientTimeout)
	defer cancel()

	client := &http.Client{Timeout: releaseNotesClientTimeout}
	release, err := fetchReleaseURL(ctx, client, releaseTagAPIURL(tag))
	if err != nil {
		releaseNotesCacheVersion = tag
		releaseNotesCacheBody = ""
		releaseNotesCacheErr = err
		return "", err
	}

	body := strings.TrimSpace(release.Body)
	if body == "" {
		err := fmt.Errorf("release %s has no release notes body", tag)
		releaseNotesCacheVersion = tag
		releaseNotesCacheBody = ""
		releaseNotesCacheErr = err
		return "", err
	}

	releaseNotesCacheVersion = tag
	releaseNotesCacheBody = body
	releaseNotesCacheErr = nil
	return body, nil
}

func fetchReleaseByTag(ctx context.Context, client *http.Client, tag string) (*gitHubRelease, error) {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil, fmt.Errorf("release tag is required")
	}
	return fetchReleaseURL(ctx, client, releaseTagAPIURL(tag))
}

func fetchReleaseURL(ctx context.Context, client *http.Client, url string) (*gitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent())

	resp, err := client.Do(req)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		if isRateLimited(resp) {
			return nil, &RateLimitError{
				Message: fmt.Sprintf("GitHub API rate limit exceeded (HTTP %d)", resp.StatusCode),
			}
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, &NetworkError{Err: fmt.Errorf("GitHub API %s: %s", resp.Status, msg)}
	}

	var release gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, &NetworkError{Err: fmt.Errorf("decode release metadata: %w", err)}
	}
	if strings.TrimSpace(release.TagName) == "" {
		return nil, fmt.Errorf("release response missing tag_name")
	}
	return &release, nil
}

func fetchReleasesPage(ctx context.Context, client *http.Client, url string) ([]gitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent())

	resp, err := client.Do(req)
	if err != nil {
		return nil, &NetworkError{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		if isRateLimited(resp) {
			return nil, &RateLimitError{
				Message: fmt.Sprintf("GitHub API rate limit exceeded (HTTP %d)", resp.StatusCode),
			}
		}
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, &NetworkError{Err: fmt.Errorf("GitHub API %s: %s", resp.Status, msg)}
	}

	var releases []gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, &NetworkError{Err: fmt.Errorf("decode releases list: %w", err)}
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no GitHub releases found")
	}
	return releases, nil
}

func isRateLimited(resp *http.Response) bool {
	remaining := resp.Header.Get("X-Ratelimit-Remaining")
	if remaining == "0" {
		return true
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true
	}
	retryAfter := resp.Header.Get("Retry-After")
	return retryAfter != ""
}

func userAgent() string {
	return fmt.Sprintf("arx-ca-updater/%s (%s/%s)", version.Current(), githubOwner, githubRepo)
}

func findAssetURL(release *gitHubRelease, assetName string) (string, error) {
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			if strings.TrimSpace(asset.BrowserDownloadURL) == "" {
				return "", fmt.Errorf("asset %q has no download URL", assetName)
			}
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("release %s has no asset named %q", release.TagName, assetName)
}
