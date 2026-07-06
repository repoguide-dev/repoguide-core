package contracts

import "encoding/json"

type Usage struct {
	Model        string
	InputTokens  int
	OutputTokens int
}

type TopicSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type SelectTopicResult struct {
	TopicID  string
	Status   string
	Reason   string
	Question string
}

type PriorSession struct {
	Task  string
	Files []string
}

type TopicCurationSuggestion struct {
	Kind                string          `json:"kind"`
	TargetField         string          `json:"target_field"`
	Path                string          `json:"path"`
	Value               json.RawMessage `json:"value"`
	Claim               string          `json:"claim"`
	EvidenceFeedbackIDs []string        `json:"evidence_feedback_ids"`
	Confidence          int             `json:"confidence"`
	Reason              string          `json:"reason"`
}

type TopicCurationSkip struct {
	FeedbackID string `json:"feedback_id"`
	Reason     string `json:"reason"`
}

type TopicSuggestionDecision struct {
	SuggestionID          string `json:"suggestion_id"`
	Decision              string `json:"decision"`
	MergeIntoSuggestionID string `json:"merge_into_suggestion_id,omitempty"`
	Confidence            int    `json:"confidence"`
	Reason                string `json:"reason"`
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
