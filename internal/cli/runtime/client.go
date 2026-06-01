package runtime

import (
	"fmt"
	"strings"

	cliapi "github.com/your-org/arx-ca/internal/cli/api"
	clicfg "github.com/your-org/arx-ca/internal/cli/config"
)

// NewAuthenticatedClient resolves the server URL and builds an API client using the stored JWT.
func NewAuthenticatedClient(flagURL string) (*cliapi.Client, error) {
	url, err := ResolveServerURL(URLResolveOptions{
		FlagOverride: flagURL,
		PersistFlag:  strings.TrimSpace(flagURL) != "",
	})
	if err != nil {
		return nil, err
	}

	cfg, err := clicfg.Load()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return nil, fmt.Errorf("not logged in; run arx login first")
	}

	return cliapi.NewClient(url, cfg.BearerToken())
}
