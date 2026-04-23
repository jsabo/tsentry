package client

import (
	"context"
	"fmt"

	apiclient "github.com/gravitational/teleport/api/client"

	"github.com/jsabo/tsentry/internal/config"
)

// New creates a Teleport API client from the active tsh profile, or from an
// identity file when --identity is provided (for tbot).
func New(ctx context.Context, cfg *config.Config) (*apiclient.Client, error) {
	var creds apiclient.Credentials
	if cfg.IdentityFile != "" {
		creds = apiclient.LoadIdentityFile(cfg.IdentityFile)
	} else {
		creds = apiclient.LoadProfile(cfg.TSHProfileDir, cfg.Cluster)
	}

	clientCfg := apiclient.Config{
		Credentials: []apiclient.Credentials{creds},
	}
	if cfg.ProxyAddr != "" {
		clientCfg.Addrs = []string{cfg.ProxyAddr}
	}

	c, err := apiclient.New(ctx, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to Teleport: %w", err)
	}
	return c, nil
}
