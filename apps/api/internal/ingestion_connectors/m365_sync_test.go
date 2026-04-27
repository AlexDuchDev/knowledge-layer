package ingestion_connectors

import (
	"encoding/json"
	"testing"
)

func TestM365DriveChildrenGraphPath(t *testing.T) {
	p, err := m365DriveChildrenGraphPath("")
	if err != nil || p != "/me/drive/root/children" {
		t.Fatalf("empty ref: %q %v", p, err)
	}
	p, err = m365DriveChildrenGraphPath("me|root")
	if err != nil || p != "/me/drive/root/children" {
		t.Fatalf("me|root: %q %v", p, err)
	}
	p, err = m365DriveChildrenGraphPath("driveX|root")
	if err != nil || p != "/drives/driveX/root/children" {
		t.Fatalf("drive|root: %q %v", p, err)
	}
	p, err = m365DriveChildrenGraphPath("driveX|itemY")
	if err != nil || p != "/drives/driveX/items/itemY/children" {
		t.Fatalf("drive|item: %q %v", p, err)
	}
	if _, err := m365DriveChildrenGraphPath("onlyone"); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateMicrosoft365SourceFeedForActivation_onedriveRef(t *testing.T) {
	raw := json.RawMessage(`{"m365_product":"onedrive","graph_access_token":"tok"}`)
	feed := &SourceFeed{ConnectorConfigJSON: raw, ExternalRef: "me|root"}
	if err := ValidateMicrosoft365SourceFeedForActivation(feed); err != nil {
		t.Fatal(err)
	}
	feed.ExternalRef = "bad"
	if ValidateMicrosoft365SourceFeedForActivation(feed) == nil {
		t.Fatal("expected error")
	}
}

func TestValidateMicrosoft365SourceFeedForActivation_searchRequiresQuery(t *testing.T) {
	raw := json.RawMessage(`{"m365_product":"sharepoint","graph_access_token":"tok","m365_files_scope":"search"}`)
	feed := &SourceFeed{ConnectorConfigJSON: raw, ExternalRef: ""}
	if ValidateMicrosoft365SourceFeedForActivation(feed) == nil {
		t.Fatal("expected error for missing search query")
	}
	raw2 := json.RawMessage(`{"m365_product":"sharepoint","graph_access_token":"tok","m365_files_scope":"search","m365_search_query":"budget"}`)
	feed2 := &SourceFeed{ConnectorConfigJSON: raw2, ExternalRef: ""}
	if err := ValidateMicrosoft365SourceFeedForActivation(feed2); err != nil {
		t.Fatal(err)
	}
}

func TestParseM365FeedConfig_products(t *testing.T) {
	for _, prod := range []string{"outlook", "teams", "onedrive", "sharepoint", "calendar"} {
		cfg, err := parseM365FeedConfig(json.RawMessage(`{"m365_product":"` + prod + `","graph_access_token":"x"}`))
		if err != nil || cfg.Product != prod {
			t.Fatalf("%s: %v %#v", prod, err, cfg)
		}
	}
	if _, err := parseM365FeedConfig(json.RawMessage(`{"m365_product":"nope","graph_access_token":"x"}`)); err == nil {
		t.Fatal("expected error")
	}
}
