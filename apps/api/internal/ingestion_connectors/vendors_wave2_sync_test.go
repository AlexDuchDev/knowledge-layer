package ingestion_connectors

import (
	"encoding/json"
	"testing"
)

func TestParseAsanaFeedConfig(t *testing.T) {
	if _, err := ParseAsanaFeedConfig([]byte(`{}`)); err == nil {
		t.Fatal("expected error")
	}
	_, err := ParseAsanaFeedConfig([]byte(`{"asana_personal_access_token":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateAsanaSourceFeedForActivation(t *testing.T) {
	raw := json.RawMessage(`{"asana_personal_access_token":"tok"}`)
	feed := &SourceFeed{ConnectorConfigJSON: raw, ExternalRef: "proj"}
	if err := ValidateAsanaSourceFeedForActivation(feed); err != nil {
		t.Fatal(err)
	}
	feed.ExternalRef = ""
	if ValidateAsanaSourceFeedForActivation(feed) == nil {
		t.Fatal("expected error")
	}
}

func TestParseLinearFeedConfig(t *testing.T) {
	if _, err := ParseLinearFeedConfig([]byte(`{}`)); err == nil {
		t.Fatal("expected error")
	}
	_, err := ParseLinearFeedConfig([]byte(`{"linear_api_key":"k"}`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateLinearSourceFeedForActivation(t *testing.T) {
	raw := json.RawMessage(`{"linear_api_key":"k"}`)
	feed := &SourceFeed{ConnectorConfigJSON: raw, ExternalRef: "team-uuid"}
	if err := ValidateLinearSourceFeedForActivation(feed); err != nil {
		t.Fatal(err)
	}
	feed.ExternalRef = ""
	if ValidateLinearSourceFeedForActivation(feed) == nil {
		t.Fatal("expected error")
	}
}

func TestParseHubSpotFeedConfig(t *testing.T) {
	if _, err := parseHubSpotFeedConfig([]byte(`{"hubspot_access_token":"t","hubspot_feed_kind":"contacts"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := parseHubSpotFeedConfig([]byte(`{"hubspot_feed_kind":"contacts"}`)); err == nil {
		t.Fatal("expected error")
	}
	if _, err := parseHubSpotFeedConfig([]byte(`{"hubspot_private_app_token":"t","hubspot_feed_kind":"bad"}`)); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseZendeskFeedConfig(t *testing.T) {
	raw := `{"zendesk_subdomain":"acme","zendesk_email":"a@b.c","zendesk_api_token":"tok"}`
	if _, err := parseZendeskFeedConfig([]byte(raw)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateZendeskSourceFeedForActivation_view(t *testing.T) {
	base := json.RawMessage(`{"zendesk_subdomain":"acme","zendesk_email":"a@b.c","zendesk_api_token":"tok","zendesk_feed_kind":"view"}`)
	feed := &SourceFeed{ConnectorConfigJSON: base, ExternalRef: "123"}
	if err := ValidateZendeskSourceFeedForActivation(feed); err != nil {
		t.Fatal(err)
	}
	feed.ExternalRef = ""
	if ValidateZendeskSourceFeedForActivation(feed) == nil {
		t.Fatal("expected error")
	}
}

func TestHubSpotDisplayTitle(t *testing.T) {
	if got := hubspotDisplayTitle("contacts", map[string]string{"firstname": "A", "lastname": "B"}); got != "A B" {
		t.Fatalf("%q", got)
	}
}
