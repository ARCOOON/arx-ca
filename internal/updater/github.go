package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ARCOOON/arx-ca/internal/version"
)

const (
	githubOwner    = "ARCOOON"
	githubRepo     = "arx-ca"
	requestTimeout = 2 * time.Minute
)

// releaseLatestAPIURL is the GitHub REST endpoint for the newest release (overridable in tests).
var releaseLatestAPIURL = "https://api.github.com/repos/ARCOOON/arx-ca/releases/latest"

type gitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []gitHubAsset `json:"assets"`
}

type gitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func fetchLatestRelease(ctx context.Context, client *http.Client) (*gitHubRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseLatestAPIURL, nil)
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
