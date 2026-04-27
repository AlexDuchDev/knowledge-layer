package ingestion_connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// GoogleDriveFolderRef is a child folder for onboarding (pick folder_id).
type GoogleDriveFolderRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListGoogleDriveFolders lists direct child folders of parentID ("root" if empty).
func (s *Service) ListGoogleDriveFolders(ctx context.Context, serviceAccountJSON []byte, parentID string) ([]GoogleDriveFolderRef, error) {
	if len(serviceAccountJSON) == 0 {
		return nil, fmt.Errorf("google_drive: service_account json required")
	}
	parentID = strings.TrimSpace(parentID)
	if parentID == "" {
		parentID = "root"
	}
	creds, err := google.CredentialsFromJSON(ctx, serviceAccountJSON, drive.DriveReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("google drive credentials: %w", err)
	}
	client := oauth2.NewClient(ctx, creds.TokenSource)
	svc, err := drive.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	q := fmt.Sprintf("mimeType='application/vnd.google-apps.folder' and '%s' in parents and trashed=false", parentID)
	list, err := svc.Files.List().Q(q).PageSize(100).Fields("files(id,name)").Do()
	if err != nil {
		return nil, err
	}
	out := make([]GoogleDriveFolderRef, 0, len(list.Files))
	for _, f := range list.Files {
		if f.Id != "" {
			out = append(out, GoogleDriveFolderRef{ID: f.Id, Name: f.Name})
		}
	}
	return out, nil
}

// ServiceAccountJSONFromConnectorConfig extracts service_account bytes from connector_config_json.
func ServiceAccountJSONFromConnectorConfig(raw json.RawMessage) ([]byte, error) {
	var m struct {
		ServiceAccount json.RawMessage `json:"service_account"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if len(m.ServiceAccount) == 0 {
		return nil, fmt.Errorf("google: service_account missing in connector_config_json")
	}
	return m.ServiceAccount, nil
}
