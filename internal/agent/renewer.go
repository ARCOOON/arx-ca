package agent

import (
	"context"

	"github.com/ARCOOON/arx-ca/internal/config"
)

// Renewer renews a managed certificate when its remaining TTL is below the threshold.
type Renewer interface {
	Renew(ctx context.Context, managed config.ManagedCert) error
}
