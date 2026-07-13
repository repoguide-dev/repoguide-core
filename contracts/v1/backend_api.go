package contracts

import (
	"time"

	"github.com/repoguide/repoguide-core/model"
)

// This file holds the wire request/response types shared by repoguide-cli's
// backend HTTP client (repoguide-cli/internal/sessionimport/repo_sync.go) and
// repoguide-cloud/backend's HTTP handlers (server/repo_handlers.go and
// auth/auth_handlers.go). These types cross the CLI<->cloud wire boundary, so
// keeping a single canonical definition avoids field-name/shape drift between
// the two independently-maintained sides.
//
// Where the CLI-observed shape and the cloud's current encoded shape genuinely
// differed, the CLI's shape won (see repoguide-core/contracts/v1 backend_api.go
// review notes accompanying this change) to preserve current client behavior;
// any such case is called out on the corresponding type below.

// RepoInfo is the response body for GET /api/repos/{repo_id}.
type RepoInfo struct {
	RepoID     string    `json:"repo_id"`
	RepoName   string    `json:"repo_name"`
	TeamID     string    `json:"team_id,omitempty"`
	LastSynced time.Time `json:"last_synced"`
}

// LimitInfo describes a single usage limit within LimitsResponse.
type LimitInfo struct {
	Type    string    `json:"type"`
	Used    int       `json:"used"`
	Max     int       `json:"max"`
	ResetAt time.Time `json:"reset_at"`
}

// LimitsResponse is the response body for GET /api/limits (and is reused
// server-side for the admin user list's per-user usage payload).
type LimitsResponse struct {
	Plan   string `json:"plan"`
	Period struct {
		Start   time.Time `json:"start"`
		End     time.Time `json:"end"`
		ResetAt time.Time `json:"reset_at"`
	} `json:"period"`
	Limits []LimitInfo `json:"limits"`
}

// MeResponse is the response body for GET /api/auth/me.
type MeResponse struct {
	Email     string `json:"email"`
	UserID    string `json:"user_id"`
	Plan      string `json:"plan"`
	IsAdmin   bool   `json:"is_admin"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// AuthSessionResponse is the response body for login/register/refresh-style
// auth endpoints that mint a bearer token for the current user.
type AuthSessionResponse struct {
	Token   string `json:"token"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin,omitempty"`
}

// MCPTopicSummary is the shape of each entry in the "topics" array returned by
// GET /api/repos/{repo_id}/mcp/topics.
type MCPTopicSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

// MCPTopicStartFile, MCPTopicImportantFiles, MCPTopicTests, and MCPTopicContext
// mirror model.TopicStartFile / TopicImportantFiles / TopicTests / TopicContext
// field-for-field, EXCEPT that MCPTopicContext omits model.TopicContext's
// Evidence field.
//
// MISMATCH (flagged, left unresolved by design): the cloud handler
// server/repo_handlers.go:687 mcpTopicContext (GET
// /api/repos/{repo_id}/mcp/topics/{topic_id}) currently does
// `writeJSON(w, 200, t)` where t is a model.TopicContext, which DOES include
// the "evidence" field on the wire. repoguide-cli/internal/sessionimport/repo_sync.go:483-496
// (MCPTopicContext) has never modeled that field, so it is silently dropped by
// the CLI's JSON decode today - not a decode error, just data the CLI has
// never looked at. Per the safety rule (prefer preserving current behavior
// over cleaning up), MCPTopicContext below stays in its CLI-observed shape
// (no Evidence field), and the cloud handler is intentionally NOT switched to
// encode via contracts.MCPTopicContext for its response - it keeps writing the
// richer model.TopicContext value directly, so the wire bytes the cloud sends
// are unchanged. Only the CLI-side decode target now aliases this type.
type MCPTopicStartFile struct {
	Path string `json:"path"`
	Role string `json:"role"`
	Why  string `json:"why"`
}

type MCPTopicImportantFiles struct {
	EditTargets       []string `json:"edit_targets,omitempty"`
	ReferenceFiles    []string `json:"reference_files,omitempty"`
	TestFiles         []string `json:"test_files,omitempty"`
	CrossCuttingFiles []string `json:"cross_cutting_files,omitempty"`
}

type MCPTopicTests struct {
	StartWith []string `json:"start_with,omitempty"`
	Signal    string   `json:"signal,omitempty"`
	Notes     []string `json:"notes,omitempty"`
	Commands  []string `json:"commands,omitempty"`
}

type MCPTopicContext struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Summary          string                 `json:"summary"`
	Confidence       float64                `json:"confidence"`
	WhenToUse        []string               `json:"when_to_use,omitempty"`
	PromptKeywords   []string               `json:"prompt_keywords,omitempty"`
	StartHere        []MCPTopicStartFile    `json:"start_here,omitempty"`
	ImportantFiles   MCPTopicImportantFiles `json:"important_files"`
	Tests            MCPTopicTests          `json:"tests"`
	KnownWorkflows   []string               `json:"known_workflows,omitempty"`
	AvoidWastingTime []string               `json:"avoid_wasting_time,omitempty"`
	RiskFlags        []string               `json:"risk_flags,omitempty"`
}

// MCPSearchHeavyTarget is one entry of MCPSearchContext.SearchHeavyTargets.
// It is NOT identical to model.BundleSearchTarget: it adds a "use_for" field
// that the cloud handler (server/repo_handlers.go:781-795 mcpSearchContext)
// populates from the selected topic's StartHere[].Why, so it is kept as its
// own type rather than reusing model.BundleSearchTarget.
type MCPSearchHeavyTarget struct {
	Path                   string   `json:"path"`
	FoundViaSearchSessions int      `json:"found_via_search_sessions"`
	TopQueries             []string `json:"top_queries,omitempty"`
	UseFor                 string   `json:"use_for,omitempty"`
}

// MCPAmbiguousSearch is one entry of MCPSearchContext.AmbiguousSearches. This
// shape is identical to model.BundleAmbiguousSearch (same fields/json tags),
// which is what the cloud handler already encodes directly for this field, so
// callers may use either interchangeably; it is kept as a distinct named type
// here only for symmetry with MCPSearchHeavyTarget's request/response pairing.
type MCPAmbiguousSearch struct {
	Query               string `json:"query"`
	Searches            int    `json:"searches"`
	DistinctEditTargets int    `json:"distinct_edit_targets"`
}

// MCPSearchContext is the response body for
// GET /api/repos/{repo_id}/mcp/search.
type MCPSearchContext struct {
	SearchHeavyTargets []MCPSearchHeavyTarget `json:"search_heavy_targets"`
	AmbiguousSearches  []MCPAmbiguousSearch   `json:"ambiguous_searches"`
}

// MCPUnderstandTaskRequest is the request body for
// POST /api/repos/{repo_id}/mcp/understand-task.
type MCPUnderstandTaskRequest struct {
	Task    string   `json:"task"`
	TopicID string   `json:"topic_id"`
	Prompts []string `json:"prompts"`
}

// MCPUnderstandTaskResult is the response body for
// POST /api/repos/{repo_id}/mcp/understand-task.
//
// NOTE: the cloud's local understandTaskResponse type (previously at
// server/repo_handlers.go:24) marks Explanation/TopicID/ContextText/Reason/
// Question as `omitempty`; the CLI's decode-only type never had omitempty
// (irrelevant for decoding). omitempty is preserved here so the cloud's
// current wire bytes (which fields get omitted when empty) do not change.
type MCPUnderstandTaskResult struct {
	Status            string   `json:"status"`
	Explanation       string   `json:"explanation,omitempty"`
	TopicID           string   `json:"topic_id,omitempty"`
	ContextText       string   `json:"context_text,omitempty"`
	Reason            string   `json:"reason,omitempty"`
	Question          string   `json:"question,omitempty"`
	CandidateTopicIDs []string `json:"candidate_topic_ids,omitempty"`
}

// MCPCallCreateRequest is the request body for
// POST /api/repos/{repo_id}/mcp-calls.
//
// NOTE: AgentName/AgentVersion/SessionID/FeedbackID are populated by the CLI
// but the cloud handler (server/repo_handlers.go:467-475 repoMCPCalls) does
// not decode them from the body - it reads agent name/version/session id from
// the X-Agent-* headers instead (set separately via setAgentHeaders), and does
// not use FeedbackID at all. These fields are harmlessly ignored by the
// server's JSON decode. Kept as-is (not a shape mismatch requiring
// resolution, just an unused subset).
type MCPCallCreateRequest struct {
	Command      string         `json:"command"`
	Inputs       map[string]any `json:"inputs,omitempty"`
	Response     map[string]any `json:"response,omitempty"`
	AgentName    string         `json:"agent_name,omitempty"`
	AgentVersion string         `json:"agent_version,omitempty"`
	SessionID    string         `json:"session_id,omitempty"`
	FeedbackID   string         `json:"feedback_id,omitempty"`
}

// MCPCallCreateResponse is the response body for
// POST /api/repos/{repo_id}/mcp-calls.
type MCPCallCreateResponse struct {
	CallID    string    `json:"call_id"`
	CreatedAt time.Time `json:"created_at"`
}

// MCPFeedbackRequest is the request body for
// POST /api/repos/{repo_id}/mcp/feedback.
type MCPFeedbackRequest struct {
	Task                string                  `json:"task"`
	Stars               int                     `json:"stars"`
	Helpfulness         string                  `json:"helpfulness,omitempty"`
	HelpedWith          []string                `json:"helped_with,omitempty"`
	Quote               string                  `json:"quote,omitempty"`
	MissingContext      string                  `json:"missing_context,omitempty"`
	WhatWentWrong       string                  `json:"what_went_wrong,omitempty"`
	WhatCouldBeImproved string                  `json:"what_could_be_improved,omitempty"`
	AdviceEvaluation    *model.AdviceEvaluation `json:"advice_evaluation,omitempty"`
	CandidateRule       *model.CandidateRule    `json:"candidate_rule,omitempty"`
	MCPCallID           string                  `json:"mcp_call_id,omitempty"`
	TopicID             string                  `json:"topic_id,omitempty"`
}

// TopicEntryStatusUpdateRequest is the request body for
// POST /api/repos/{repo_id}/topics/{topic_id}/entries/status.
type TopicEntryStatusUpdateRequest struct {
	Section string `json:"section"`
	ItemKey string `json:"item_key,omitempty"`
	Status  string `json:"status"`
}

// TopicEntryTextUpdateRequest is the request body for
// POST /api/repos/{repo_id}/topics/{topic_id}/entries/edit.
type TopicEntryTextUpdateRequest struct {
	Section string `json:"section"`
	ItemKey string `json:"item_key,omitempty"`
	Text    string `json:"text"`
}

// Note: no conversion helper between MCPTopicContext and model.TopicContext is
// provided here - the cloud side keeps encoding model.TopicContext directly
// for GET .../mcp/topics/{topic_id} (see MISMATCH note above), and the CLI's
// local (SQLite) mode already does its own json round-trip conversion in
// repoguide-cli/internal/sessionimport/repo_sync.go (jsonRoundTrip).
