package ingestion_connectors

// SourceFeedGovernanceService is the application port for governed source feed validation and lifecycle.
// Persistence methods remain on Service; this type anchors orchestration and HTTP layers to a named port.
type SourceFeedGovernanceService struct{ *Service }

// SourceFeedGovernance returns the source feed governance port.
func (s *Service) SourceFeedGovernance() SourceFeedGovernanceService {
	return SourceFeedGovernanceService{s}
}

// ValidateCreateInput runs governance validation without persisting.
func (s SourceFeedGovernanceService) ValidateCreateInput(in CreateSourceFeedInput) error {
	return ValidateCreateSourceFeedInput(in)
}
