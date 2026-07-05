package runtime

import (
	"fmt"
	"log/slog"

	"github.com/ARCOOON/arx-ca/internal/api/middleware"
	"github.com/ARCOOON/arx-ca/internal/config"
	"github.com/ARCOOON/arx-ca/internal/events"
	"github.com/ARCOOON/arx-ca/internal/logging"
)

// Applier hot-reloads runtime-safe configuration without process restart.
type Applier struct {
	firewall *middleware.Firewall
	events   *events.Manager
}

// NewApplier constructs an applier bound to the shared firewall instance.
func NewApplier(firewall *middleware.Firewall, eventManager *events.Manager) *Applier {
	applier := &Applier{
		firewall: firewall,
		events:   eventManager,
	}
	if eventManager != nil {
		eventManager.Subscribe(events.EventSystemConfigUpdated, func(evt events.Event) {
			cfg := config.ServerConfigFromViper()
			if err := applier.Apply(cfg); err != nil {
				logging.Logger().Error("config: runtime apply after event",
					slog.Any("error", err),
				)
			}
		})
	}
	return applier
}

// Apply updates runtime state from the active configuration snapshot.
func (a *Applier) Apply(cfg config.ServerConfig) error {
	if a == nil {
		return nil
	}

	logging.Configure(cfg.Server.LogLevel)

	if err := config.ApplyProvisionerRuntimeEnv(cfg.CA.EffectiveProvisioners()); err != nil {
		return fmt.Errorf("apply provisioner runtime env: %w", err)
	}

	if a.firewall != nil {
		rules, err := middleware.FirewallFromAllowlist(
			cfg.Security.Firewall.Enabled,
			cfg.Security.Firewall.Allowlist,
		)
		if err != nil {
			return fmt.Errorf("parse firewall allowlist: %w", err)
		}
		a.firewall.Update(rules)
	}

	return nil
}
