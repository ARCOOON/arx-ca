package ui

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	arxconfig "github.com/ARCOOON/arx-ca/internal/config"
	"github.com/ARCOOON/arx-ca/internal/version"
)

const (
	githubOwner       = "ARCOOON"
	githubRepo        = "arx-ca"
	webUIAssetName    = "webui-dist.tar.gz"
	defaultRequestDur = 2 * time.Minute
)

// releaseAPIBase is the GitHub REST releases collection (overridable in tests).
var releaseAPIBase = "https://api.github.com/repos/ARCOOON/arx-ca/releases"

type gitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []gitHubAsset `json:"assets"`
}

type gitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// DownloadAndBootstrapWebUI fetches webui-dist.tar.gz from GitHub for the given release tag,
// extracts it into the configured webui.ui_dir, and enables the WebUI block in server.yaml.
// When requestedVersion is empty, the release tag is derived from the running arx binary version.
func DownloadAndBootstrapWebUI(configPath, requestedVersion string) error {
	return downloadAndBootstrap(context.Background(), nil, os.Stdout, configPath, requestedVersion)
}

func downloadAndBootstrap(ctx context.Context, client *http.Client, out io.Writer, configPath, requestedVersion string) error {
	if out == nil {
		out = os.Stdout
	}
	if client == nil {
		client = &http.Client{Timeout: defaultRequestDur}
	}

	path, err := arxconfig.ResolveServerConfigPath(configPath)
	if err != nil {
		return fmt.Errorf("resolve server config path: %w", err)
	}

	cfg, found, err := arxconfig.ReadServerConfig(configPath)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("server configuration not found at %s: run 'arx-ca server config init' first", path)
	}

	uiDir := strings.TrimSpace(cfg.WebUI.UIDir)
	if uiDir == "" {
		uiDir = arxconfig.DefaultServerConfig().WebUI.UIDir
	}
	absUIDir, err := filepath.Abs(uiDir)
	if err != nil {
		return fmt.Errorf("resolve webui ui_dir %q: %w", uiDir, err)
	}

	requestedVersion = strings.TrimSpace(requestedVersion)
	var releaseTarget, displayVersion string
	if requestedVersion != "" {
		releaseTarget, displayVersion = resolveReleaseTarget(requestedVersion)
		fmt.Fprintf(out, "Using requested WebUI version: %s\n", displayVersion)
		fmt.Fprintf(out, "Using GitHub release target: %s (requested version %s)\n", releaseTarget, displayVersion)
	} else {
		fmt.Fprintln(out, "Detecting server version...")
		binaryVersion := version.Current()
		releaseTarget, displayVersion = resolveReleaseTarget(binaryVersion)
		fmt.Fprintf(out, "Using GitHub release target: %s (binary version %s)\n", releaseTarget, displayVersion)
	}

	fmt.Fprintln(out, "Downloading webui-dist.tar.gz from GitHub...")
	release, err := fetchRelease(ctx, client, releaseTarget)
	if err != nil {
		return err
	}

	downloadURL, err := findAssetURL(release, webUIAssetName)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(absUIDir, 0o755); err != nil {
		return fmt.Errorf("create webui directory %s: %w", absUIDir, err)
	}

	resp, err := downloadAsset(ctx, client, downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Fprintf(out, "Extracting assets to %s...\n", absUIDir)
	if err := extractTarGz(resp.Body, absUIDir); err != nil {
		return fmt.Errorf("extract %s: %w", webUIAssetName, err)
	}

	cfg.WebUI.Enabled = true
	cfg.WebUI.UIDir = absUIDir
	if err := arxconfig.PersistServerConfig(path, cfg); err != nil {
		return fmt.Errorf("update server configuration: %w", err)
	}

	fmt.Fprintln(out, "WebUI successfully enabled in server.yaml!")
	return nil
}

func resolveReleaseTarget(ver string) (apiPathSuffix, display string) {
	ver = strings.TrimSpace(ver)
	if ver == "" {
		ver = version.Default
	}
	display = ver
	if ver == "" || ver == version.Default || ver == "v0.0.0-dev" {
		return "latest", display
	}
	tag := ver
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return "tags/" + tag, display
}

func releaseAPIURL(target string) string {
	switch target {
	case "latest":
		return releaseAPIBase + "/latest"
	default:
		return releaseAPIBase + "/" + target
	}
}

func fetchRelease(ctx context.Context, client *http.Client, target string) (*gitHubRelease, error) {
	url := releaseAPIURL(target)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build GitHub API request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("GitHub API %s: %s", resp.Status, msg)
	}

	var release gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode GitHub release metadata: %w", err)
	}
	if strings.TrimSpace(release.TagName) == "" {
		return nil, fmt.Errorf("GitHub release response missing tag_name")
	}
	return &release, nil
}

func findAssetURL(release *gitHubRelease, assetName string) (string, error) {
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			url := strings.TrimSpace(asset.BrowserDownloadURL)
			if url == "" {
				return "", fmt.Errorf("release %s asset %q has no download URL", release.TagName, assetName)
			}
			return url, nil
		}
	}
	return "", fmt.Errorf("release %s has no asset named %q", release.TagName, assetName)
}

func downloadAsset(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build download request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent())

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", webUIAssetName, err)
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return nil, fmt.Errorf("download %s: HTTP %s: %s", webUIAssetName, resp.Status, msg)
	}
	return resp, nil
}

func extractTarGz(r io.Reader, destDir string) error {
	destDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("resolve destination directory: %w", err)
	}

	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		if err := extractTarEntry(tr, hdr, destDir); err != nil {
			return err
		}
	}
}

func extractTarEntry(tr *tar.Reader, hdr *tar.Header, destDir string) error {
	name := filepath.Clean(hdr.Name)
	if name == "." || name == "" {
		return nil
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, "..") || strings.Contains(name, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid tar entry path: %q", hdr.Name)
	}

	target := filepath.Join(destDir, name)
	if !withinBase(destDir, target) {
		return fmt.Errorf("tar entry escapes destination directory: %q", hdr.Name)
	}

	switch hdr.Typeflag {
	case tar.TypeDir:
		if err := os.MkdirAll(target, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", target, err)
		}
		return nil
	case tar.TypeReg, tar.TypeRegA:
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create parent directory for %s: %w", target, err)
		}
		f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return fmt.Errorf("create file %s: %w", target, err)
		}
		if _, err := io.Copy(f, tr); err != nil {
			_ = f.Close()
			return fmt.Errorf("write file %s: %w", target, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close file %s: %w", target, err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported tar entry type %c for %q", hdr.Typeflag, hdr.Name)
	}
}

func withinBase(base, target string) bool {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	if target == base {
		return true
	}
	sep := string(os.PathSeparator)
	return strings.HasPrefix(target, base+sep)
}

func userAgent() string {
	return fmt.Sprintf("arx-ca-webui-bootstrap/%s (%s/%s)", version.Current(), githubOwner, githubRepo)
}
