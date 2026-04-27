// Package work_mgmt defines the work management connector family (Jira, Trello, etc.).
package work_mgmt

// RecordTypeWorkItem is normalized_records.record_type for an issue/card.
const RecordTypeWorkItem = "work_item"

// ArtifactKindIssue is raw payload for a vendor issue object.
const ArtifactKindIssue = "work_mgmt_issue"

const ArtifactAsanaTask = "asana_task"
const ArtifactAsanaStory = "asana_story"
const ArtifactLinearIssue = "linear_issue"
const ArtifactLinearComment = "linear_comment"
