package contracts

import (
	"encoding/json"

	"github.com/repoguide/repoguide-core/model"
)

type Usage struct {
	Model        string
	InputTokens  int
	OutputTokens int
}

type TopicSummary struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Summary        string   `json:"summary"`
	WhenToUse      []string `json:"when_to_use,omitempty"`
	PromptKeywords []string `json:"prompt_keywords,omitempty"`
}

// TopicMatch is a task-to-topic relevance score. Confidence is calibrated to
// the current task, not the topic's stored analysis confidence.
type TopicMatch struct {
	TopicID    string  `json:"topic_id"`
	Name       string  `json:"name,omitempty"`
	Confidence float64 `json:"confidence"`
}

// AdviceItem is an immutable, evidence-backed observation. The LLM selector
// may rank these items by ID, but may not author or rewrite their text.
type AdviceItem struct {
	ID                   string            `json:"id"`
	Text                 string            `json:"text"`
	Steps                []string          `json:"steps,omitempty"`
	Files                []string          `json:"files,omitempty"`
	Severity             string            `json:"severity,omitempty"`
	Kind                 string            `json:"kind"`
	Support              int               `json:"support"`
	Total                int               `json:"total"`
	Confidence           float64           `json:"confidence"`
	HelpfulFeedback      int               `json:"helpful_feedback,omitempty"`
	UnhelpfulFeedback    int               `json:"unhelpful_feedback,omitempty"`
	HelpfulTextMatches   int               `json:"helpful_text_matches,omitempty"`
	UnhelpfulTextMatches int               `json:"unhelpful_text_matches,omitempty"`
	Source               string            `json:"source,omitempty"`
	LastObservedAt       string            `json:"last_observed_at,omitempty"`
	Evidence             CandidateEvidence `json:"evidence,omitempty"`
}

// CandidateEvidence records the retrieved session population behind one
// candidate. It keeps runtime guidance auditable and prevents topic-wide
// aggregates from being presented as task-specific evidence.
type CandidateEvidence struct {
	MatchingSessionIDs []string `json:"matching_session_ids,omitempty"`
	Support            int      `json:"support"`
	Population         int      `json:"population"`
	ExtractionMethod   string   `json:"extraction_method"`
}

// TopicRoutingExample is feedback-qualified prior routing evidence. It helps
// the selector judge relevance; it is never implementation evidence.
type TopicRoutingExample struct {
	Task     string `json:"task"`
	TopicID  string `json:"topic_id"`
	Feedback string `json:"feedback"`
	Reason   string `json:"reason,omitempty"`
}

type SelectionBudget struct {
	MaxTotal         int `json:"max_total"`
	MaxPerCategory   int `json:"max_per_category"`
	MinDistinctKinds int `json:"min_distinct_kinds"`
	MaxCharacters    int `json:"max_characters"`
}

// AdviceSelectionResponse contains IDs only. Category membership and budgets
// are validated against the deterministic candidate set after the LLM call.
type AdviceSelectionResponse struct {
	StartFiles      []string `json:"start_files,omitempty"`
	Workflows       []string `json:"workflows,omitempty"`
	Avoid           []string `json:"avoid,omitempty"`
	Tests           []string `json:"tests,omitempty"`
	ScopeBoundaries []string `json:"scope_boundaries,omitempty"`
	Risks           []string `json:"risks,omitempty"`
}

type SelectTopicResult struct {
	TopicID           string
	Confidence        float64
	Status            string
	Reason            string
	Question          string
	CandidateTopics   []TopicMatch
	CandidateTopicIDs []string
}

type TopicCurationSuggestion struct {
	Kind                string               `json:"kind"`
	TargetField         string               `json:"target_field"`
	Path                string               `json:"path"`
	Value               json.RawMessage      `json:"value"`
	Claim               string               `json:"claim"`
	EvidenceFeedbackIDs []string             `json:"evidence_feedback_ids"`
	Confidence          int                  `json:"confidence"`
	Reason              string               `json:"reason"`
	CandidateRule       *model.CandidateRule `json:"candidate_rule,omitempty"`
}

type TopicCurationSkip struct {
	FeedbackID string `json:"feedback_id"`
	Reason     string `json:"reason"`
}

type TopicSuggestionDecision struct {
	SuggestionID           string   `json:"suggestion_id"`
	Decision               string   `json:"decision"`
	MergeIntoSuggestionID  string   `json:"merge_into_suggestion_id,omitempty"`
	SupportedByFeedbackIDs []string `json:"supported_by_feedback_ids,omitempty"`
	Confidence             int      `json:"confidence"`
	Reason                 string   `json:"reason"`
}

type TopicCuration struct {
	RepoID              string                    `json:"repo_id"`
	TopicID             string                    `json:"topic_id"`
	NewSuggestions      []TopicCurationSuggestion `json:"new_suggestions"`
	SuggestionDecisions []TopicSuggestionDecision `json:"suggestion_decisions"`
	SkippedFeedback     []TopicCurationSkip       `json:"skipped_feedback"`
}

type RepoContextSession struct {
	FeedbackID   string   `json:"feedback_id"`
	SessionID    string   `json:"session_id,omitempty"`
	UserPrompts  []string `json:"user_prompts,omitempty"`
	EditedFiles  []string `json:"edited_files,omitempty"`
	DiffSnippets []string `json:"diff_snippets,omitempty"`
}

// TopicCurationSession is the per-feedback session evidence passed to the
// topic curator: what the user actually asked, what the agent touched, and
// which commands ran or failed.
type TopicCurationSession struct {
	FeedbackID     string   `json:"feedback_id"`
	SessionID      string   `json:"session_id,omitempty"`
	Prompts        []string `json:"prompts,omitempty"`
	EditedFiles    []string `json:"edited_files,omitempty"`
	ReadFiles      []string `json:"read_files,omitempty"`
	Commands       []string `json:"commands,omitempty"`
	FailedCommands []string `json:"failed_commands,omitempty"`
	DiffSnippets   []string `json:"diff_snippets,omitempty"`
}

type RepoContextPatchEdit struct {
	Op                  string   `json:"op"`
	Section             string   `json:"section"`
	Value               string   `json:"value"`
	EvidenceFeedbackIDs []string `json:"evidence_feedback_ids"`
	Reason              string   `json:"reason"`
	Confidence          string   `json:"confidence"`
}

type RepoContextPatchSkip struct {
	FeedbackID string `json:"feedback_id"`
	Reason     string `json:"reason"`
}

type RepoContextPatch struct {
	RepoID  string                 `json:"repo_id"`
	Edits   []RepoContextPatchEdit `json:"edits"`
	Skipped []RepoContextPatchSkip `json:"skipped"`
}
