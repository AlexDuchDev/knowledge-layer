package ingestion_connectors

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// GoogleCalendarListRef is a calendar list entry for onboarding (external_ref = calendar id).
type GoogleCalendarListRef struct {
	ID          string `json:"id"`
	Summary     string `json:"summary"`
	Description string `json:"description,omitempty"`
	Primary     bool   `json:"primary,omitempty"`
}

// ListGoogleCalendars lists calendars visible to the service account.
func (s *Service) ListGoogleCalendars(ctx context.Context, serviceAccountJSON []byte) ([]GoogleCalendarListRef, error) {
	if len(serviceAccountJSON) == 0 {
		return nil, fmt.Errorf("google_calendar: service_account json required")
	}
	creds, err := google.CredentialsFromJSON(ctx, serviceAccountJSON, calendar.CalendarReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("google calendar credentials: %w", err)
	}
	client := oauth2.NewClient(ctx, creds.TokenSource)
	svc, err := calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	list, err := svc.CalendarList.List().MaxResults(250).Do()
	if err != nil {
		return nil, err
	}
	var out []GoogleCalendarListRef
	for _, it := range list.Items {
		if it.Id == "" {
			continue
		}
		out = append(out, GoogleCalendarListRef{
			ID:          it.Id,
			Summary:     it.Summary,
			Description: it.Description,
			Primary:     it.Primary,
		})
	}
	return out, nil
}
