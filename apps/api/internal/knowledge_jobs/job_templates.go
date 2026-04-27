package knowledge_jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ErrUnknownJobTemplate is returned when template_id does not match a catalog entry.
var ErrUnknownJobTemplate = errors.New("unknown job template")

// JobTemplate describes a first-class job preset for inspectors and creation flows.
type JobTemplate struct {
	ID                     string          `json:"id"`
	Title                  string          `json:"title"`
	Description            string          `json:"description"`
	JobType                string          `json:"job_type"`
	DefaultName            string          `json:"default_name"`
	Purpose                string          `json:"purpose"`
	DefaultPublicationMode string          `json:"default_publication_mode"`
	DefaultProcessingMode  string          `json:"default_processing_mode"`
	ReviewRequired         bool            `json:"review_required"`
	OutputSensitivity      int             `json:"output_sensitivity"`
	DefaultCitations       bool            `json:"default_citations_required"`
	DefaultProvenance      bool            `json:"default_provenance_required"`
	DefaultSourceScope     json.RawMessage `json:"default_source_scope_json"`
	DefaultConfig          json.RawMessage `json:"default_config_json"`
}

// JobTemplatePublic is the API shape for listing templates.
type JobTemplatePublic struct {
	ID                    string `json:"id"`
	Title                 string `json:"title"`
	Description           string `json:"description"`
	JobType               string `json:"job_type"`
	ProcessorImplemented  bool   `json:"processor_implemented"`
	DefaultName           string `json:"default_name"`
	DefaultReviewReq      bool   `json:"default_review_required"`
	DefaultOutputSens     int    `json:"default_output_sensitivity"`
	DefaultProcessingMode string `json:"default_processing_mode"`
	DefaultCitationsReq   bool   `json:"default_citations_required"`
	DefaultProvenanceReq  bool   `json:"default_provenance_required"`
	SourceScopeHintJSON   string `json:"source_scope_hint_json"`
	DefaultConfigPreview  string `json:"default_config_preview_json"`
}

var jobTemplates = []JobTemplate{
	{
		ID:                     "weekly_digest",
		Title:                  "Weekly digest",
		Description:            "Aggregates recent normalized records from one source feed into a derived Insight and opens a review task. Requires domain_id and source_feed_id in scope.",
		JobType:                "weekly_digest",
		DefaultName:            "Weekly digest",
		Purpose:                "Roll up normalized feed content for the last 7 days into an inspectable derived Insight.",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "summarize",
		ReviewRequired:         true,
		OutputSensitivity:      0,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"window_days":7}`),
	},
	{
		ID:                     "weekly_daily_digest",
		Title:                  "Weekly / daily digest",
		Description:            "Alias for weekly digest template (same processor); use for daily-leaning presets via config_json.",
		JobType:                "weekly_digest",
		DefaultName:            "Digest",
		Purpose:                "Roll up normalized feed content into an inspectable derived Insight.",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "summarize",
		ReviewRequired:         true,
		OutputSensitivity:      0,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"window_days":7,"cadence":"weekly"}`),
	},
	{
		ID:                     "planning_summary",
		Title:                  "Planning summary",
		Description:            "Summarizes planning-oriented signals from one governed source feed into a derived Insight and review task.",
		JobType:                "planning_summary",
		DefaultName:            "Planning summary",
		Purpose:                "Structured planning-oriented synthesis from in-scope knowledge (runner TBD).",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "summarize",
		ReviewRequired:         true,
		OutputSensitivity:      0,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"focus":"planning"}`),
	},
	{
		ID:                     "decision_extraction",
		Title:                  "Decision extraction",
		Description:            "Tracks candidate decisions and rationale extracted from one governed source feed.",
		JobType:                "decision_extraction",
		DefaultName:            "Decision extraction",
		Purpose:                "Isolate explicit decisions, owners, and rationale from ingested or canonical text.",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "extract",
		ReviewRequired:         true,
		OutputSensitivity:      0,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"mode":"decision_mentions"}`),
	},
	{
		ID:                     "blocker_detection",
		Title:                  "Blocker detection",
		Description:            "Detect blockers and dependencies from operational signals. Runner stubbed until monitoring pipeline is wired.",
		JobType:                "blocker_detection",
		DefaultName:            "Blocker detection",
		Purpose:                "Surface blockers and dependency risks for review.",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "detect",
		ReviewRequired:         true,
		OutputSensitivity:      0,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"signal_types":["status","risk"],"lookback_days":14}`),
	},
	{
		ID:                     "incident_summary",
		Title:                  "Incident summary",
		Description:            "Post-incident structured summary. Runner stubbed for v1.",
		JobType:                "incident_summary",
		DefaultName:            "Incident summary",
		Purpose:                "Capture timeline, impact, and follow-ups after incidents.",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "summarize",
		ReviewRequired:         true,
		OutputSensitivity:      1,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"template":"incident","include_timeline":true}`),
	},
	{
		ID:                     "executive_consolidation",
		Title:                  "Executive consolidation",
		Description:            "Multi-source executive brief; citations expected when published.",
		JobType:                "executive_consolidation",
		DefaultName:            "Executive consolidation",
		Purpose:                "Consolidate governed sources into a leadership-facing brief.",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "consolidate",
		ReviewRequired:         true,
		OutputSensitivity:      1,
		DefaultCitations:       true,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"format":"executive_brief","max_sections":7}`),
	},
	{
		ID:                     "support_trends_extraction",
		Title:                  "Support trends extraction",
		Description:            "Extracts recurring support themes from one governed source feed into a derived Insight and review task.",
		JobType:                "support_trends_extraction",
		DefaultName:            "Support trends extraction",
		Purpose:                "Extract recurring themes from support corpus.",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "extract",
		ReviewRequired:         true,
		OutputSensitivity:      0,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"domain":"support","aggregation":"weekly"}`),
	},
	{
		ID:                     "retro_summary",
		Title:                  "Retro summary",
		Description:            "Retrospective themes and action items. Runner stubbed for v1.",
		JobType:                "retro_summary",
		DefaultName:            "Retro summary",
		Purpose:                "Summarize retro inputs into actions and owners.",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "summarize",
		ReviewRequired:         true,
		OutputSensitivity:      0,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"retro_format":"standard","include_action_items":true}`),
	},
	{
		ID:                     "stale_scan",
		Title:                  "Stale knowledge scan",
		Description:            "Checks age and ownership drift across in-scope entities and produces a governance-ready Insight.",
		JobType:                "stale_scan",
		DefaultName:            "Stale knowledge scan",
		Purpose:                "Flag stale or unowned knowledge objects for governance follow-up.",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "detect",
		ReviewRequired:         true,
		OutputSensitivity:      0,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"max_age_days":365}`),
	},
	{
		ID:                     "leadership_brief",
		Title:                  "Leadership brief",
		Description:            "Short executive-style digest template; runner stubbed until synthesis pipeline is wired.",
		JobType:                "leadership_brief",
		DefaultName:            "Leadership brief",
		Purpose:                "Condensed briefing from governed in-scope material with citations (future).",
		DefaultPublicationMode: PublicationModeReviewedPublish,
		DefaultProcessingMode:  "summarize",
		ReviewRequired:         true,
		OutputSensitivity:      1,
		DefaultCitations:       true,
		DefaultProvenance:      true,
		DefaultSourceScope:     json.RawMessage(`{}`),
		DefaultConfig:          json.RawMessage(`{"format":"brief","max_sections":5}`),
	},
}

var jobTemplateIndex map[string]JobTemplate

func init() {
	jobTemplateIndex = make(map[string]JobTemplate, len(jobTemplates))
	for _, t := range jobTemplates {
		jobTemplateIndex[t.ID] = t
	}
}

// ListJobTemplatesPublic returns catalog entries for admin UI and API clients.
func ListJobTemplatesPublic() []JobTemplatePublic {
	out := make([]JobTemplatePublic, 0, len(jobTemplates))
	for _, t := range jobTemplates {
		dpm := t.DefaultProcessingMode
		if dpm == "" {
			dpm = "summarize"
		}
		out = append(out, JobTemplatePublic{
			ID:                    t.ID,
			Title:                 t.Title,
			Description:           t.Description,
			JobType:               t.JobType,
			ProcessorImplemented:  IsKnowledgeJobProcessorImplemented(t.JobType),
			DefaultName:           t.DefaultName,
			DefaultReviewReq:      t.ReviewRequired,
			DefaultOutputSens:     t.OutputSensitivity,
			DefaultProcessingMode: dpm,
			DefaultCitationsReq:   t.DefaultCitations,
			DefaultProvenanceReq:  t.DefaultProvenance,
			SourceScopeHintJSON:   string(t.DefaultSourceScope),
			DefaultConfigPreview:  string(t.DefaultConfig),
		})
	}
	return out
}

// ApplyJobTemplate merges template defaults into a create payload. Unknown template returns an error.
// Template is authoritative for job_type, review_required, and output_sensitivity when template_id is set.
// Name and purpose use explicit request values when provided. JSON objects are deep-merged (request wins on key clashes).
func ApplyJobTemplate(in CreateJobInput) (CreateJobInput, error) {
	if in.TemplateID == "" {
		return in, nil
	}
	t, ok := jobTemplateIndex[in.TemplateID]
	if !ok {
		return in, fmt.Errorf("%w: %s", ErrUnknownJobTemplate, in.TemplateID)
	}
	out := in
	out.JobType = t.JobType
	out.ReviewRequired = t.ReviewRequired
	out.OutputSensitivity = t.OutputSensitivity
	if strings.TrimSpace(in.ProcessingMode) == "" && strings.TrimSpace(t.DefaultProcessingMode) != "" {
		out.ProcessingMode = t.DefaultProcessingMode
	}
	out.CitationsRequired = in.CitationsRequired || t.DefaultCitations
	if in.ProvenanceRequired == nil && t.DefaultProvenance {
		v := true
		out.ProvenanceRequired = &v
	} else {
		out.ProvenanceRequired = in.ProvenanceRequired
	}
	if in.Name == "" {
		out.Name = t.DefaultName
	}
	purposeFromRequest := ""
	if in.Purpose != nil {
		purposeFromRequest = strings.TrimSpace(*in.Purpose)
	}
	if purposeFromRequest == "" {
		p := t.Purpose
		out.Purpose = &p
	}
	out.SourceScopeJSON = mergeJSONObjects(t.DefaultSourceScope, in.SourceScopeJSON)
	out.ConfigJSON = mergeJSONObjects(t.DefaultConfig, in.ConfigJSON)
	if strings.TrimSpace(in.PublicationMode) == "" && strings.TrimSpace(t.DefaultPublicationMode) != "" {
		out.PublicationMode = t.DefaultPublicationMode
	}
	return out, nil
}

func mergeJSONObjects(base, overlay json.RawMessage) json.RawMessage {
	if len(overlay) == 0 || string(overlay) == "null" {
		return cloneRawJSON(base)
	}
	var bm, om map[string]any
	_ = json.Unmarshal(base, &bm)
	_ = json.Unmarshal(overlay, &om)
	if bm == nil {
		bm = map[string]any{}
	}
	if om == nil {
		om = map[string]any{}
	}
	for k, v := range om {
		bm[k] = v
	}
	out, err := json.Marshal(bm)
	if err != nil {
		return cloneRawJSON(base)
	}
	return out
}

func cloneRawJSON(b json.RawMessage) json.RawMessage {
	if len(b) == 0 {
		return []byte("{}")
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
