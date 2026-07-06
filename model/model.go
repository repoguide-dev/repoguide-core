package model

import (
	"encoding/json"
	"fmt"
	"time"
)

// ── Repo ─────────────────────────────────────────────────────────────────────
//
// Repo is the CLI's storage-layer entity (referenced by contracts.RepoStore in
// contracts/v1/store.go). It intentionally excludes cloud-only fields like AI
// analysis history, which live in repoguide-cloud/backend/model's own Repo type.

type Repo struct {
	UserID      string    `bson:"user_id"   json:"user_id,omitempty"`
	RepoID      string    `bson:"repo_id"   json:"repo_id"`
	RepoRoot    string    `bson:"repo_root" json:"repo_root"`
	RepoName    string    `bson:"repo_name" json:"repo_name"`
	ActivatedAt time.Time `bson:"activated_at" json:"activated_at"`
	UpdatedAt   time.Time `bson:"updated_at"   json:"updated_at"`
}

// ── Sessions & Events ────────────────────────────────────────────────────────

type RepoSessionEvents struct {
	Agent     string         `bson:"agent"              json:"agent"`
	ID        string         `bson:"id"                 json:"id"`
	Name      string         `bson:"name,omitempty"     json:"name,omitempty"`
	Events    []SessionEvent `bson:"events"             json:"events"`
	UpdatedAt time.Time      `bson:"updatedAt"          json:"updatedAt"`
}

type SessionEvent struct {
	Index        int               `bson:"index"                  json:"index"`
	Timestamp    string            `bson:"timestamp,omitempty"    json:"timestamp,omitempty"`
	Kind         string            `bson:"kind"                   json:"kind"`
	Role         string            `bson:"role,omitempty"         json:"role,omitempty"`
	Text         string            `bson:"text,omitempty"         json:"text,omitempty"`
	Model        string            `bson:"model,omitempty"        json:"model,omitempty"`
	ToolName     string            `bson:"toolName,omitempty"     json:"toolName,omitempty"`
	ToolCallID   string            `bson:"toolCallId,omitempty"   json:"toolCallId,omitempty"`
	IsError      bool              `bson:"isError,omitempty"      json:"isError,omitempty"`
	TokenUsage   *TokenUsage       `bson:"tokenUsage,omitempty"   json:"tokenUsage,omitempty"`
	ReadPaths    []string          `bson:"readPaths,omitempty"    json:"readPaths,omitempty"`
	WritePaths   []string          `bson:"writePaths,omitempty"   json:"writePaths,omitempty"`
	Command      []string          `bson:"command,omitempty"      json:"command,omitempty"`
	CommandText  string            `bson:"commandText,omitempty"  json:"commandText,omitempty"`
	SearchQuery  string            `bson:"searchQuery,omitempty"  json:"searchQuery,omitempty"`
	LinesAdded   int               `bson:"linesAdded,omitempty"   json:"linesAdded,omitempty"`
	LinesRemoved int               `bson:"linesRemoved,omitempty" json:"linesRemoved,omitempty"`
	Metadata     map[string]string `bson:"metadata,omitempty"     json:"metadata,omitempty"`
}

type TokenUsage struct {
	InputTokens      int64 `bson:"inputTokens"      json:"inputTokens"`
	OutputTokens     int64 `bson:"outputTokens"     json:"outputTokens"`
	CacheReadTokens  int64 `bson:"cacheReadTokens"  json:"cacheReadTokens"`
	CacheWriteTokens int64 `bson:"cacheWriteTokens" json:"cacheWriteTokens"`
	Cumulative       bool  `bson:"cumulative"       json:"cumulative"`
}

// ── Topics & MCP context ─────────────────────────────────────────────────────

type TopicStartFile struct {
	Path string `bson:"path" json:"path"`
	Role string `bson:"role" json:"role"`
	Why  string `bson:"why"  json:"why"`
}

type TopicImportantFiles struct {
	EditTargets       []string `bson:"edit_targets,omitempty"       json:"edit_targets,omitempty"`
	ReferenceFiles    []string `bson:"reference_files,omitempty"    json:"reference_files,omitempty"`
	TestFiles         []string `bson:"test_files,omitempty"         json:"test_files,omitempty"`
	CrossCuttingFiles []string `bson:"cross_cutting_files,omitempty" json:"cross_cutting_files,omitempty"`
}

type TopicTests struct {
	StartWith []string `bson:"start_with,omitempty" json:"start_with,omitempty"`
	Signal    string   `bson:"signal,omitempty"     json:"signal,omitempty"`
	Notes     []string `bson:"notes,omitempty"      json:"notes,omitempty"`
	Commands  []string `bson:"commands,omitempty"   json:"commands,omitempty"`
}

type TopicEvidence struct {
	Sessions    int    `bson:"sessions"     json:"sessions"`
	EditedFiles int    `bson:"edited_files" json:"edited_files"`
	ReadFiles   int    `bson:"read_files"   json:"read_files"`
	LastActive  string `bson:"last_active,omitempty" json:"last_active,omitempty"` // date of most recent evidence session
}

type TopicContext struct {
	ID               string              `bson:"id"                         json:"id"`
	Name             string              `bson:"name"                       json:"name"`
	Summary          string              `bson:"summary"                    json:"summary"`
	Confidence       float64             `bson:"confidence"                 json:"confidence"`
	WhenToUse        []string            `bson:"when_to_use,omitempty"      json:"when_to_use,omitempty"`
	PromptKeywords   []string            `bson:"prompt_keywords,omitempty"  json:"prompt_keywords,omitempty"`
	StartHere        []TopicStartFile    `bson:"start_here,omitempty"       json:"start_here,omitempty"`
	ImportantFiles   TopicImportantFiles `bson:"important_files"            json:"important_files"`
	Tests            TopicTests          `bson:"tests"                      json:"tests"`
	KnownWorkflows   []string            `bson:"known_workflows,omitempty"  json:"known_workflows,omitempty"`
	AvoidWastingTime []string            `bson:"avoid_wasting_time,omitempty" json:"avoid_wasting_time,omitempty"`
	RiskFlags        []string            `bson:"risk_flags,omitempty"       json:"risk_flags,omitempty"`
	Evidence         TopicEvidence       `bson:"evidence"                   json:"evidence"`
}

type FileSummary struct {
	Path             string   `bson:"path"                        json:"path"`
	Classification   []string `bson:"classification,omitempty"    json:"classification,omitempty"`
	ReadBeforeEditOf []string `bson:"read_before_edit_of,omitempty" json:"read_before_edit_of,omitempty"`
	RelatedTests     []string `bson:"related_tests,omitempty"     json:"related_tests,omitempty"`
	CoEditedWith     []string `bson:"co_edited_with,omitempty"    json:"co_edited_with,omitempty"`
}

type BundleSearchTarget struct {
	Path                   string   `bson:"path"                     json:"path"`
	FoundViaSearchSessions int      `bson:"found_via_search_sessions" json:"found_via_search_sessions"`
	TopQueries             []string `bson:"top_queries,omitempty"    json:"top_queries,omitempty"`
}

type BundleAmbiguousSearch struct {
	Query               string `bson:"query"                 json:"query"`
	Searches            int    `bson:"searches"              json:"searches"`
	DistinctEditTargets int    `bson:"distinct_edit_targets" json:"distinct_edit_targets"`
}

type BundleSearchData struct {
	SearchHeavyTargets []BundleSearchTarget    `bson:"search_heavy_targets,omitempty" json:"search_heavy_targets,omitempty"`
	AmbiguousSearches  []BundleAmbiguousSearch `bson:"ambiguous_searches,omitempty"   json:"ambiguous_searches,omitempty"`
}

type RepoContextEntry struct {
	ContextID string    `json:"context_id"`
	Content   string    `json:"content"`
	BundleID  string    `json:"bundle_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ── Feedback ─────────────────────────────────────────────────────────────────

const (
	FeedbackKindQualityOnly      = "quality_only"
	FeedbackKindPatchTopic       = "patch_topic"
	FeedbackKindPatchRepo        = "patch_repo"
	FeedbackKindTopicCandidate   = "topic_candidate"
	FeedbackKindRefreshCandidate = "refresh_candidate"
	FeedbackKindUnclear          = "unclear"

	FeedbackQualityGood           = "good"
	FeedbackQualityGoodButMissing = "good_but_missing"
	FeedbackQualityBadWrong       = "bad_wrong"
	FeedbackQualityBadStale       = "bad_stale"
	FeedbackQualityBadMissing     = "bad_missing"
	FeedbackQualityLowValue       = "low_value"
	FeedbackQualityNeutral        = "neutral"
	FeedbackQualityUnclear        = "unclear"

	FeedbackSeverityInfo   = "info"
	FeedbackSeverityLow    = "low"
	FeedbackSeverityMedium = "medium"
	FeedbackSeverityHigh   = "high"
)

var validFeedbackKinds = map[string]bool{
	FeedbackKindQualityOnly: true, FeedbackKindPatchTopic: true, FeedbackKindPatchRepo: true,
	FeedbackKindTopicCandidate: true, FeedbackKindRefreshCandidate: true, FeedbackKindUnclear: true,
}
var validFeedbackQualities = map[string]bool{
	FeedbackQualityGood: true, FeedbackQualityGoodButMissing: true, FeedbackQualityBadWrong: true,
	FeedbackQualityBadStale: true, FeedbackQualityBadMissing: true, FeedbackQualityLowValue: true,
	FeedbackQualityNeutral: true, FeedbackQualityUnclear: true,
}
var validFeedbackSeverities = map[string]bool{
	FeedbackSeverityInfo: true, FeedbackSeverityLow: true, FeedbackSeverityMedium: true, FeedbackSeverityHigh: true,
}

type FeedbackActionableItem struct {
	Type       string   `bson:"type"               json:"type"`
	Text       string   `bson:"text"               json:"text"`
	Files      []string `bson:"files,omitempty"    json:"files,omitempty"`
	Confidence string   `bson:"confidence"         json:"confidence"`
}

type FeedbackClassification struct {
	Kind            string                   `bson:"kind"                      json:"kind"`
	Quality         string                   `bson:"quality"                   json:"quality"`
	Severity        string                   `bson:"severity"                  json:"severity"`
	ActionableItems []FeedbackActionableItem `bson:"actionable_items,omitempty" json:"actionable_items,omitempty"`
	Reason          string                   `bson:"reason"                    json:"reason"`
	Confidence      string                   `bson:"confidence"                json:"confidence"`
}

func (c *FeedbackClassification) Validate() error {
	if !validFeedbackKinds[c.Kind] {
		return fmt.Errorf("invalid classification kind %q", c.Kind)
	}
	if !validFeedbackQualities[c.Quality] {
		return fmt.Errorf("invalid classification quality %q", c.Quality)
	}
	if !validFeedbackSeverities[c.Severity] {
		return fmt.Errorf("invalid classification severity %q", c.Severity)
	}
	return nil
}

type MCPFeedback struct {
	FeedbackID          string                  `bson:"_id"                         json:"feedback_id"`
	UserID              string                  `bson:"user_id"                     json:"-"`
	RepoID              string                  `bson:"repo_id"                     json:"repo_id"`
	Task                string                  `bson:"task"                        json:"task"`
	Stars               int                     `bson:"stars,omitempty"             json:"stars,omitempty"`
	Helpfulness         string                  `bson:"helpfulness"                 json:"helpfulness"`
	HelpedWith          []string                `bson:"helped_with,omitempty"       json:"helped_with,omitempty"`
	Quote               string                  `bson:"quote,omitempty"             json:"quote,omitempty"`
	MissingContext      string                  `bson:"missing_context,omitempty"   json:"missing_context,omitempty"`
	TopicID             string                  `bson:"topic_id,omitempty"          json:"topic_id,omitempty"`
	SessionID           string                  `bson:"session_id,omitempty"        json:"session_id,omitempty"`
	MCPCallID           string                  `bson:"mcp_call_id,omitempty"       json:"mcp_call_id,omitempty"`
	WhatWentWrong       string                  `bson:"what_went_wrong,omitempty"   json:"what_went_wrong,omitempty"`
	WhatCouldBeImproved string                  `bson:"what_could_be_improved,omitempty" json:"what_could_be_improved,omitempty"`
	Classification      *FeedbackClassification `bson:"classification,omitempty"    json:"classification,omitempty"`
	ProcessedAt         *time.Time              `bson:"processed_at,omitempty"      json:"processed_at,omitempty"`
	ProcessingJobID     string                  `bson:"processing_job_id,omitempty" json:"processing_job_id,omitempty"`
	CreatedAt           time.Time               `bson:"created_at"                  json:"created_at"`
}

// ── MCP calls ────────────────────────────────────────────────────────────────

type MCPCall struct {
	CallID       string         `bson:"_id"                    json:"call_id"`
	UserID       string         `bson:"user_id"                json:"-"`
	RepoID       string         `bson:"repo_id"                json:"repo_id"`
	Command      string         `bson:"command"                json:"command"`
	Inputs       map[string]any `bson:"inputs,omitempty"       json:"inputs,omitempty"`
	Response     map[string]any `bson:"response,omitempty"     json:"response,omitempty"`
	AgentName    string         `bson:"agent_name,omitempty"   json:"agent_name,omitempty"`
	AgentVersion string         `bson:"agent_version,omitempty" json:"agent_version,omitempty"`
	SessionID    string         `bson:"session_id,omitempty"   json:"session_id,omitempty"`
	TopicID      string         `bson:"topic_id,omitempty"     json:"topic_id,omitempty"`
	CreatedAt    time.Time      `bson:"created_at"             json:"created_at"`
}

// ── Jobs ─────────────────────────────────────────────────────────────────────

const (
	PatchJobStatusQueued  = "QUEUED"
	PatchJobStatusRunning = "RUNNING"
	PatchJobStatusDone    = "DONE"
	PatchJobStatusError   = "ERROR"
	PatchJobStatusRetry   = "RETRY"
	PatchJobMaxRetries    = 3

	PatchJobTypeTopicPatch       = "topic_patch"
	PatchJobTypeNewTopic         = "new_topic"
	PatchJobTypeRepoPatch        = "repo_context_patch"
	PatchJobTypeClassifyFeedback = "classifyfeedback"
)

type ContextPatchJob struct {
	JobID        string    `bson:"_id"                   json:"job_id"`
	UserID       string    `bson:"user_id"               json:"-"`
	RepoID       string    `bson:"repo_id"               json:"repo_id"`
	Type         string    `bson:"type"                  json:"type"`
	TopicID      string    `bson:"topic_id,omitempty"    json:"topic_id,omitempty"`
	FeedbackIDs  []string  `bson:"feedback_ids,omitempty" json:"feedback_ids,omitempty"`
	Timestamp    time.Time `bson:"timestamp"             json:"timestamp"`
	Status       string    `bson:"status"                json:"status"`
	RetryCount   int       `bson:"retry_count"           json:"retry_count"`
	CreatedAt    time.Time `bson:"created_at"            json:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at"            json:"updated_at"`
	NewTopicID   string    `bson:"new_topic_id,omitempty"   json:"new_topic_id,omitempty"`
	NewTopicName string    `bson:"new_topic_name,omitempty" json:"new_topic_name,omitempty"`
	Reason       string    `bson:"reason,omitempty"         json:"reason,omitempty"`
	LastError    string    `bson:"last_error,omitempty"     json:"last_error,omitempty"`
}

// ── Topic patch suggestions ───────────────────────────────────────────────────

const (
	TopicPatchSuggestionPending  = "pending"
	TopicPatchSuggestionApplied  = "applied"
	TopicPatchSuggestionRejected = "rejected"
)

// TopicPatchSuggestion is a proposed change to a topic, persisted across runs
// so the model can review and decide on it in a future job.
type TopicPatchSuggestion struct {
	SuggestionID        string          `bson:"_id"                    json:"suggestion_id"`
	RepoID              string          `bson:"repo_id"                json:"repo_id"`
	TopicID             string          `bson:"topic_id"               json:"topic_id"`
	Status              string          `bson:"status"                 json:"status"` // pending | applied | rejected
	CreatedAt           time.Time       `bson:"created_at"             json:"created_at"`
	UpdatedAt           time.Time       `bson:"updated_at"             json:"updated_at"`
	Kind                string          `bson:"kind"                   json:"kind"`
	TargetField         string          `bson:"target_field"           json:"target_field"`
	Path                string          `bson:"path"                   json:"path"`
	Value               json.RawMessage `bson:"value"                  json:"value"`
	Claim               string          `bson:"claim"                  json:"claim"`
	EvidenceFeedbackIDs []string        `bson:"evidence_feedback_ids"  json:"evidence_feedback_ids"`
	Confidence          int             `bson:"confidence"             json:"confidence"`
	Reason              string          `bson:"reason"                 json:"reason"`
}
