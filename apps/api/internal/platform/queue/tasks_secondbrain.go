package queue

// TaskSecondBrainOutbound delivers a short text to a linked chat (cmd/jobworker).
const TaskSecondBrainOutbound = "secondbrain:outbound"

// SecondBrainOutboundPayload references the user and channel for outbound delivery.
type SecondBrainOutboundPayload struct {
	UserID  string `json:"user_id"`
	Channel string `json:"channel"` // telegram | mattermost
	Text    string `json:"text"`
}
