package ingestion_connectors

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

func (s *Service) ListConnectors(ctx context.Context) ([]Connector, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, type, display_name, auth_mode, status, auth_config_ref, capabilities_json, config_json
		FROM connectors ORDER BY display_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Connector
	for rows.Next() {
		var c Connector
		if err := rows.Scan(&c.ID, &c.Type, &c.DisplayName, &c.AuthMode, &c.Status, &c.AuthConfigRef, &c.CapabilitiesJSON, &c.ConfigJSON); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *Service) GetConnector(ctx context.Context, id uuid.UUID) (*Connector, error) {
	var c Connector
	err := s.pool.QueryRow(ctx, `
		SELECT id, type, display_name, auth_mode, status, auth_config_ref, capabilities_json, config_json
		FROM connectors WHERE id=$1`, id).
		Scan(&c.ID, &c.Type, &c.DisplayName, &c.AuthMode, &c.Status, &c.AuthConfigRef, &c.CapabilitiesJSON, &c.ConfigJSON)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateConnector registers a new connector type (unique `type` column).
func (s *Service) CreateConnector(ctx context.Context, typ, displayName, authMode string, capabilities json.RawMessage) (*Connector, error) {
	if typ == "" || displayName == "" {
		return nil, errors.New("type and display_name required")
	}
	if len(capabilities) == 0 {
		capabilities = []byte("{}")
	}
	if authMode == "" {
		authMode = "unspecified"
	}
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO connectors (id, type, display_name, auth_mode, capabilities_json, status)
		VALUES ($1,$2,$3,$4,$5,'active')`, id, typ, displayName, authMode, capabilities)
	if err != nil {
		return nil, err
	}
	return s.GetConnector(ctx, id)
}

// PatchConnector updates display name, status, or auth mode.
func (s *Service) PatchConnector(ctx context.Context, id uuid.UUID, displayName *string, status *string, authMode *string) (*Connector, error) {
	if displayName != nil {
		_, err := s.pool.Exec(ctx, `UPDATE connectors SET display_name=$2, updated_at=now() WHERE id=$1`, id, *displayName)
		if err != nil {
			return nil, err
		}
	}
	if status != nil {
		_, err := s.pool.Exec(ctx, `UPDATE connectors SET status=$2, updated_at=now() WHERE id=$1`, id, *status)
		if err != nil {
			return nil, err
		}
	}
	if authMode != nil {
		_, err := s.pool.Exec(ctx, `UPDATE connectors SET auth_mode=$2, updated_at=now() WHERE id=$1`, id, *authMode)
		if err != nil {
			return nil, err
		}
	}
	return s.GetConnector(ctx, id)
}
