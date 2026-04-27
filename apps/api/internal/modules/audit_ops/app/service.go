package app

import (
	"context"

	"github.com/google/uuid"

	"github.com/knowledgelayer/api/internal/audit"
)

// Service is the audit_ops application service: append-only writes and operator reads.
type Service struct {
	inner *audit.Service
}

func NewService(inner *audit.Service) *Service {
	return &Service{inner: inner}
}

func (s *Service) Write(ctx context.Context, in audit.WriteInput) error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.Write(ctx, in)
}

func (s *Service) List(ctx context.Context, eventType, targetType string, limit int) ([]audit.Event, error) {
	if s == nil || s.inner == nil {
		return nil, nil
	}
	return s.inner.List(ctx, eventType, targetType, limit)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*audit.Event, error) {
	if s == nil || s.inner == nil {
		return nil, nil
	}
	return s.inner.Get(ctx, id)
}
