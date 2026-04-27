package neo4j

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Config struct {
	URL      string
	Username string
	Password string
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.URL) == "" {
		return fmt.Errorf("neo4j: missing url")
	}
	if strings.TrimSpace(c.Username) == "" {
		return fmt.Errorf("neo4j: missing username")
	}
	if strings.TrimSpace(c.Password) == "" {
		return fmt.Errorf("neo4j: missing password")
	}
	return nil
}

func Connect(ctx context.Context, cfg Config) (neo4j.DriverWithContext, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	driver, err := neo4j.NewDriverWithContext(cfg.URL, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		return nil, fmt.Errorf("neo4j: new driver: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := driver.VerifyConnectivity(pingCtx); err != nil {
		_ = driver.Close(ctx)
		return nil, fmt.Errorf("neo4j: verify connectivity: %w", err)
	}
	return driver, nil
}
