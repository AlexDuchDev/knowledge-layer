package privacy

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type stubChat struct {
	lastUser string
}

func (s *stubChat) ChatScenario(ctx context.Context, scenario, system, user string) (string, error) {
	_ = scenario
	_ = system
	s.lastUser = user
	return `{"answer":"ok","citations":[]}`, nil
}

func (s *stubChat) ChatScenarioWithMetaUserContent(ctx context.Context, scenario, system string, userContent any) (string, map[string]any, error) {
	_ = scenario
	_ = system
	s.lastUser = fmt.Sprint(userContent)
	return `{"answer":"ok","citations":[]}`, map[string]any{}, nil
}

func TestGatewayInvoke_SanitizesBeforeChat(t *testing.T) {
	g := NewPrivacyGateway(GatewayConfig{Sanitize: NewSanitizationService(nil)})
	stub := &stubChat{}
	_, err := g.InvokeOpenAI(context.Background(), stub, InvokeInput{
		System: "sys",
		Segments: []TextSegment{
			{Field: "body", Text: "mail me at x@y.co"},
		},
		PolicyCtx:       PolicyContext{},
		CorrelationID:   uuid.New().String(),
		Principal:       uuid.New(),
		RehydrationMode: RehydrationPartial,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stub.lastUser, "x@y.co") {
		t.Fatalf("user prompt leaked email: %s", stub.lastUser)
	}
	if !strings.Contains(stub.lastUser, "EMAIL_") {
		t.Fatalf("expected placeholder in: %s", stub.lastUser)
	}
}
