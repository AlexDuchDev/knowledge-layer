package queue

// Asynq task types consumed by cmd/connectorworker.

const (
	TaskConnectorSourceSync      = "connector:source_sync"
	TaskIngestionProcessArtifact = "ingestion:process_artifact"
	TaskRetrievalEmbedChunk      = "retrieval:embed_chunk"
	TaskGraphRAGExtractEntity    = "graphrag:extract_entity"
)

// ConnectorSourceSyncPayload references a source feed sync run (enqueued by API or scheduler).
type ConnectorSourceSyncPayload struct {
	SourceFeedID string `json:"source_feed_id"`
	ConnectorID  string `json:"connector_id,omitempty"`
}

// IngestionProcessArtifactPayload references a raw artifact to normalize (connectorworker: ProcessQueuedRawArtifact).
type IngestionProcessArtifactPayload struct {
	RawArtifactID string `json:"raw_artifact_id"`
}

// RetrievalEmbedChunkPayload references a chunk to embed.
type RetrievalEmbedChunkPayload struct {
	ChunkID string `json:"chunk_id"`
}

// GraphRAGExtractEntityPayload references an entity to extract into the Neo4j graph store.
type GraphRAGExtractEntityPayload struct {
	EntityID string `json:"entity_id"`
}
