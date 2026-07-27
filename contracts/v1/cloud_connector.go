package contracts

// CloudConnector is the set of calls the CLI makes against the repoguide
// cloud backend. repoguide-cli's CloudClient (internal/sessionimport/repo_sync.go)
// implements this against the real HTTP API.
type CloudConnector interface {
	// GetRepo fetches repo metadata (GET /api/repos/{repoID}). Returns (nil, nil)
	// if unauthenticated or the repo isn't registered yet (404).
	GetRepo(repoID string) (*RepoInfo, error)
	// RegisterRepo creates the repo on the backend (POST /api/repos). A 409
	// (already registered) is treated as success.
	RegisterRepo(repoID, repoRoot string) error
	// DeleteRepo removes the repo from the backend (DELETE /api/repos/{repoID}).
	DeleteRepo(repoID string) error
	// UploadRepoEvents zips session events since the repo's last sync and
	// uploads them (POST /api/repos/{repoID}/events/), then advances the
	// last-synced timestamp. No-op if there's nothing new to send.
	UploadRepoEvents(repoID, repoRoot string) error
	// TriggerAnalyze kicks off backend analysis for the repo
	// (POST /api/repos/{repoID}/analyze[?force=true]).
	TriggerAnalyze(repoID string, force bool) error
	// GetLimits fetches the caller's plan/usage limits (GET /api/limits).
	GetLimits() (*LimitsResponse, error)
	// GetMe fetches the authenticated user's account info (GET /api/auth/me).
	GetMe() (*MeResponse, error)
	// DeleteMe deletes the authenticated user's account (DELETE /api/auth/me).
	DeleteMe() error
	// GetMCPTopics lists MCP topics for a repo (GET /api/repos/{repoID}/mcp/topics).
	// Served from local disk instead of the backend for local-mode repos.
	GetMCPTopics(repoID string) ([]MCPTopicSummary, error)
	// GetMCPSearchContext runs an MCP search within a topic
	// (GET /api/repos/{repoID}/mcp/search?topic_id=...&query=...). Served
	// locally for local-mode repos.
	GetMCPSearchContext(repoID, topicID, query string) (*MCPSearchContext, error)
	// GetMCPTopicContext fetches the full context for one MCP topic
	// (GET /api/repos/{repoID}/mcp/topics/{topicID}). Served locally for
	// local-mode repos.
	GetMCPTopicContext(repoID, topicID string) (*MCPTopicContext, error)
	// GetMCPUnderstandTask asks the backend to route a task description to the
	// relevant topic(s) (POST /api/repos/{repoID}/mcp/understand-task). Served
	// locally for local-mode repos. knownFiles is the caller's git-computed
	// file listing for its configured branch, used to filter stale paths out
	// of the response (see MCPUnderstandTaskRequest.KnownFiles).
	GetMCPUnderstandTask(repoID, task, topicID string, prompts []string, knownFiles []string) (*MCPUnderstandTaskResult, error)
	// CreateMCPCall records that an MCP tool call happened
	// (POST /api/repos/{repoID}/mcp-calls). Served locally for local-mode repos.
	CreateMCPCall(repoID string, req MCPCallCreateRequest) (*MCPCallCreateResponse, error)
	// RecordMCPFeedback submits feedback tied to a prior MCP call
	// (POST /api/repos/{repoID}/mcp/feedback). Served locally for local-mode repos.
	RecordMCPFeedback(repoID string, req MCPFeedbackRequest) error
	// CheckRequiredVersion queries the backend's minimum supported CLI version
	// (GET {BaseURL}/version).
	CheckRequiredVersion() (string, error)
}
