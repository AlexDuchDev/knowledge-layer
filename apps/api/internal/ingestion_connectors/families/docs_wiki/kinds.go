// Package docs_wiki defines the documentation / wiki connector family.
package docs_wiki

// RecordTypeDocsPage is normalized_records.record_type for a logical doc page.
const RecordTypeDocsPage = "docs_page"

// ArtifactKindPage is raw_artifacts.artifact_type for a fetched page payload.
const ArtifactKindPage = "docs_page"

// ArtifactKindPageRevision captures an immutable revision snapshot when available.
const ArtifactKindPageRevision = "docs_page_revision"

// Confluence raw artifact types (v1).
const (
	ArtifactConfluencePage         = "confluence_page"
	ArtifactConfluencePageBody     = "confluence_page_body"
	ArtifactConfluencePageMetadata = "confluence_page_metadata"
	ArtifactConfluenceChildPages   = "confluence_child_pages"
)
