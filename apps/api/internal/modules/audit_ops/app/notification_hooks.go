package app

import "context"

// NotificationHookService dispatches audit events to external channels (email, webhooks). Stub until wired.
type NotificationHookService struct{}

func (NotificationHookService) OnAuditEvent(ctx context.Context, eventType string) error {
	_ = ctx
	_ = eventType
	return nil
}
