package updater

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ARCOOON/arx-ca/internal/config"
	"github.com/ARCOOON/arx-ca/internal/db"
	"github.com/ARCOOON/arx-ca/internal/logging"
	"github.com/ARCOOON/arx-ca/internal/notifications"
	"github.com/ARCOOON/arx-ca/internal/version"
)

// Engine polls GitHub releases on a schedule and notifies operators or auto-applies updates.
type Engine struct {
	cfg       config.UpdaterConfig
	audit     *db.AuditStore
	notify    *notifications.Dispatcher
	onRestart func()
	client    *http.Client

	mu              sync.Mutex
	lastNotifiedVer string
	lastAppliedVer  string
}

// NewEngine constructs a background updater bound to server configuration and notification hooks.
func NewEngine(
	cfg config.UpdaterConfig,
	audit *db.AuditStore,
	notify *notifications.Dispatcher,
	onRestart func(),
) *Engine {
	return &Engine{
		cfg:       cfg,
		audit:     audit,
		notify:    notify,
		onRestart: onRestart,
		client:    &http.Client{Timeout: requestTimeout},
	}
}

// Run executes the polling loop until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) {
	log := logging.Logger()

	interval, err := e.cfg.CheckIntervalDuration()
	if err != nil {
		log.Error("updater: invalid check_interval", slog.Any("error", err))
		return
	}

	channel := e.cfg.NormalizedChannel()
	log.Info("updater: background checker started",
		slog.String("channel", channel),
		slog.Duration("interval", interval),
		slog.Bool("notify_only", e.cfg.NotifyOnly),
		slog.String("current_version", version.Current()),
	)

	e.poll(ctx, log, channel)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("updater: background checker stopped")
			return
		case <-ticker.C:
			e.poll(ctx, log, channel)
		}
	}
}

func (e *Engine) poll(ctx context.Context, log *slog.Logger, channel string) {
	if ctx.Err() != nil {
		return
	}

	checkCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	current := version.Current()
	release, err := fetchReleaseForChannel(checkCtx, e.client, channel)
	if err != nil {
		log.Warn("updater: release check failed",
			slog.String("channel", channel),
			slog.Any("error", err),
		)
		return
	}

	remoteTag := normalizeTag(release.TagName)
	cmp, err := compareVersions(current, remoteTag)
	if err != nil {
		log.Warn("updater: version comparison failed",
			slog.String("current", current),
			slog.String("remote", remoteTag),
			slog.Any("error", err),
		)
		return
	}
	if cmp <= 0 {
		log.Debug("updater: no newer release",
			slog.String("channel", channel),
			slog.String("current", normalizeTag(current)),
			slog.String("remote", remoteTag),
		)
		return
	}

	if e.cfg.NotifyOnly {
		e.notifyUpdateAvailable(log, channel, current, remoteTag, release.TagName)
		return
	}

	e.applyUpdate(checkCtx, log, channel, current, remoteTag, release)
}

func (e *Engine) notifyUpdateAvailable(log *slog.Logger, channel, current, remoteTag, rawTag string) {
	e.mu.Lock()
	if e.lastNotifiedVer == remoteTag {
		e.mu.Unlock()
		return
	}
	e.lastNotifiedVer = remoteTag
	e.mu.Unlock()

	message := fmt.Sprintf("Update %s available on channel %q (running %s)", remoteTag, channel, normalizeTag(current))
	log.Info("updater: new release available",
		slog.String("channel", channel),
		slog.String("version", remoteTag),
		slog.String("message", message),
	)

	notifications.RecordSystemEvent(e.audit, e.notify, db.ActionSysUpdateAvailable, map[string]any{
		"message":           message,
		"channel":           channel,
		"current_version":   normalizeTag(current),
		"available_version": remoteTag,
		"release_tag":       rawTag,
	})
}

func (e *Engine) applyUpdate(ctx context.Context, log *slog.Logger, channel, current, remoteTag string, release *gitHubRelease) {
	e.mu.Lock()
	if e.lastAppliedVer == remoteTag {
		e.mu.Unlock()
		return
	}
	e.lastAppliedVer = remoteTag
	e.mu.Unlock()

	assetName := expectedAssetName(ComponentArx)
	downloadURL, err := findAssetURL(release, assetName)
	if err != nil {
		log.Error("updater: release asset missing",
			slog.String("channel", channel),
			slog.String("version", remoteTag),
			slog.String("asset", assetName),
			slog.Any("error", err),
		)
		return
	}

	log.Info("updater: downloading release",
		slog.String("channel", channel),
		slog.String("version", remoteTag),
		slog.String("asset", assetName),
	)

	if err := downloadAndApply(ctx, e.client, downloadURL); err != nil {
		log.Error("updater: binary update failed",
			slog.String("channel", channel),
			slog.String("version", remoteTag),
			slog.Any("error", err),
		)
		e.mu.Lock()
		e.lastAppliedVer = ""
		e.mu.Unlock()
		return
	}

	log.Info("updater: binary updated successfully",
		slog.String("channel", channel),
		slog.String("previous_version", normalizeTag(current)),
		slog.String("new_version", remoteTag),
	)

	notifications.RecordSystemEvent(e.audit, e.notify, db.ActionSysUpdateApplied, map[string]any{
		"message":          fmt.Sprintf("Updated arx from %s to %s on channel %q", normalizeTag(current), remoteTag, channel),
		"channel":          channel,
		"previous_version": normalizeTag(current),
		"new_version":      remoteTag,
	})

	if e.onRestart != nil {
		e.onRestart()
	}
}
