package contracts

type RepoAnalysisBundle struct {
	Version         int                             `json:"version"`
	Repo            RepoAnalysisRepo                `json:"repo"`
	Summary         RepoAnalysisSummary             `json:"summary"`
	Sessions        []RepoAnalysisSession           `json:"sessions"`
	Files           []RepoAnalysisFile              `json:"files"`
	Subsystems      []RepoAnalysisSubsystem         `json:"subsystems"`
	SeenWithGroups  []RepoAnalysisRelation          `json:"seen_with_groups"`
	Relationships   []RepoAnalysisRelationshipGroup `json:"relationships"`
	TracePatterns   []RepoAnalysisTrace             `json:"trace_patterns"`
	TestSignals     RepoAnalysisTestSignals         `json:"test_signals"`
	Discoverability RepoAnalysisDiscoverability     `json:"discoverability"`
	Docs            []RepoAnalysisDoc               `json:"docs"`
}

type RepoAnalysisRepo struct {
	Name        string `json:"name"`
	Root        string `json:"root"`
	RangeDays   int    `json:"range_days"`
	GeneratedAt string `json:"generated_at"`
}

type RepoAnalysisSummary struct {
	Sessions             int     `json:"sessions"`
	Prompts              int     `json:"prompts"`
	ToolCalls            int     `json:"tool_calls"`
	FileReads            int     `json:"file_reads"`
	FileEdits            int     `json:"file_edits"`
	ContextTokens        int64   `json:"context_tokens"`
	TotalTokens          int64   `json:"total_tokens"`
	CostUSD              float64 `json:"cost_usd"`
	AvgPromptsPerSession float64 `json:"avg_prompts_per_session"`
	AvgReadsPerSession   float64 `json:"avg_reads_per_session"`
	AvgEditsPerSession   float64 `json:"avg_edits_per_session"`
	AvgCostPerSession    float64 `json:"avg_cost_per_session"`
}

type RepoAnalysisSession struct {
	ID           string                `json:"id"`
	Title        string                `json:"title"`
	Prompts      []string              `json:"prompts,omitempty"`
	Interactions []SessionInteraction  `json:"interactions,omitempty"`
	Commands     []RepoAnalysisCommand `json:"commands,omitempty"`
	Timestamp    string                `json:"timestamp,omitempty"`
}

// RepoAnalysisCommand is a shell command observed in a session, with how often
// it ran and how often it failed (linked tool_result with is_error).
type RepoAnalysisCommand struct {
	Text     string `json:"text"`
	Runs     int    `json:"runs"`
	Failures int    `json:"failures,omitempty"`
}

type SessionInteraction struct {
	Role string `json:"role"` // "user" or "assistant"
	Text string `json:"text"`
}

type RepoAnalysisFile struct {
	Path                          string                   `json:"path"`
	Kind                          string                   `json:"kind"`
	Sessions                      int                      `json:"sessions"`
	Reads                         int                      `json:"reads"`
	ReadsPerSession               float64                  `json:"reads_per_session"`
	Edits                         int                      `json:"edits"`
	EditSessions                  int                      `json:"edit_sessions"`
	EditRate                      float64                  `json:"edit_rate"`
	ContextTokens                 int64                    `json:"context_tokens"`
	TokensPerEdit                 *float64                 `json:"tokens_per_edit"`
	LastSeen                      string                   `json:"last_seen,omitempty"`
	FirstSeen                     string                   `json:"first_seen,omitempty"`
	AvgReadsBeforeFirstEdit       *float64                 `json:"avg_reads_before_first_edit"`
	AvgPromptsBeforeFirstEdit     *float64                 `json:"avg_prompts_before_first_edit"`
	AvgContextBeforeFirstEdit     *float64                 `json:"avg_context_before_first_edit"`
	TestReads                     int                      `json:"test_reads,omitempty"`
	TestEdits                     int                      `json:"test_edits,omitempty"`
	TestReadBeforeEdit            int                      `json:"test_read_before_edit,omitempty"`
	SourceEditWithoutTestRead     int                      `json:"source_edit_without_test_read,omitempty"`
	SourceEditWithoutTestEdit     int                      `json:"source_edit_without_test_edit,omitempty"`
	RelatedTests                  []RepoAnalysisFileLink   `json:"related_tests,omitempty"`
	ReadBeforeEditOf              []string                 `json:"read_before_edit_of,omitempty"`
	Classification                []string                 `json:"classification,omitempty"`
	FoundViaSearchSessions        int                      `json:"found_via_search_sessions,omitempty"`
	AvgSearchesBeforeEdit         float64                  `json:"avg_searches_before_edit,omitempty"`
	AvgReadsAfterSearchBeforeEdit float64                  `json:"avg_reads_after_search_before_edit,omitempty"`
	TopQueries                    []RepoAnalysisQueryCount `json:"top_queries,omitempty"`
}

type RepoAnalysisFileLink struct {
	Path  string `json:"path"`
	Reads int    `json:"reads,omitempty"`
	Edits int    `json:"edits,omitempty"`
}

type RepoAnalysisSessionRef struct {
	ID        string `json:"id"`
	Name      string `json:"name,omitempty"`
	Agent     string `json:"agent,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type RepoAnalysisSubsystem struct {
	Name                    string                   `json:"name"`
	Paths                   []string                 `json:"paths"`
	Sessions                int                      `json:"sessions"`
	Reads                   int                      `json:"reads"`
	Edits                   int                      `json:"edits"`
	SourceReads             int                      `json:"source_reads,omitempty"`
	SourceEdits             int                      `json:"source_edits,omitempty"`
	TestReads               int                      `json:"test_reads,omitempty"`
	TestEdits               int                      `json:"test_edits,omitempty"`
	SourceEditSessions      int                      `json:"source_edit_sessions,omitempty"`
	TestTouchedSessions     int                      `json:"test_touched_sessions,omitempty"`
	TestTouchRate           float64                  `json:"test_touch_rate,omitempty"`
	ContextTokens           int64                    `json:"context_tokens"`
	CostUSD                 float64                  `json:"cost_usd"`
	AvgReadsBeforeFirstEdit *float64                 `json:"avg_reads_before_first_edit"`
	TopFiles                []string                 `json:"top_files,omitempty"`
	Classification          []string                 `json:"classification,omitempty"`
	RelatedSessions         []RepoAnalysisSessionRef `json:"related_sessions,omitempty"`
}

type RepoAnalysisTestSignals struct {
	Summary             RepoAnalysisTestSummary    `json:"summary"`
	TestsAsSpec         []RepoAnalysisTestRelation `json:"tests_as_spec,omitempty"`
	SourceAndTestCoEdit []RepoAnalysisTestRelation `json:"source_and_test_co_edit,omitempty"`
	TestFriction        []RepoAnalysisTestFile     `json:"test_friction,omitempty"`
	TestChurn           []RepoAnalysisTestFile     `json:"test_churn,omitempty"`
}

type RepoAnalysisTestSummary struct {
	SourceEditSessions    int     `json:"source_edit_sessions"`
	SessionsWithTestReads int     `json:"sessions_with_test_reads"`
	SessionsWithTestEdits int     `json:"sessions_with_test_edits"`
	TestTouchRate         float64 `json:"test_touch_rate"`
}

type RepoAnalysisTestRelation struct {
	Type            string                   `json:"type"`
	Source          string                   `json:"source"`
	Test            string                   `json:"test"`
	Sessions        int                      `json:"sessions"`
	RelatedSessions []RepoAnalysisSessionRef `json:"related_sessions,omitempty"`
}

type RepoAnalysisTestFile struct {
	Type            string                   `json:"type"`
	Test            string                   `json:"test"`
	Sessions        int                      `json:"sessions"`
	Reads           int                      `json:"reads,omitempty"`
	Edits           int                      `json:"edits,omitempty"`
	SourceEdits     int                      `json:"source_edits,omitempty"`
	ContextTokens   int64                    `json:"context_tokens,omitempty"`
	RelatedSessions []RepoAnalysisSessionRef `json:"related_sessions,omitempty"`
}

type RepoAnalysisRelation struct {
	Type            string                   `json:"type"`
	Files           []string                 `json:"files,omitempty"`
	Source          string                   `json:"source,omitempty"`
	Target          string                   `json:"target,omitempty"`
	Sessions        int                      `json:"sessions"`
	Strength        float64                  `json:"strength,omitempty"`
	RelatedSessions []RepoAnalysisSessionRef `json:"related_sessions,omitempty"`
}

type RepoAnalysisRelationshipGroup struct {
	Name      string   `json:"name"`
	Files     []string `json:"files"`
	Types     []string `json:"types,omitempty"`
	Strength  float64  `json:"strength,omitempty"`
	EdgeCount int      `json:"edge_count,omitempty"`
}

type RepoAnalysisTrace struct {
	Type                 string                  `json:"type,omitempty"`
	Pattern              []string                `json:"pattern,omitempty"`
	PatternType          string                  `json:"pattern_type,omitempty"`
	Source               string                  `json:"source,omitempty"`
	Target               string                  `json:"target,omitempty"`
	File                 string                  `json:"file,omitempty"`
	Chain                string                  `json:"chain,omitempty"`
	Sessions             int                     `json:"sessions"`
	Reads                int                     `json:"reads,omitempty"`
	Edits                int                     `json:"edits,omitempty"`
	AvgCostUSD           float64                 `json:"avg_cost_usd,omitempty"`
	AvgContextTokens     float64                 `json:"avg_context_tokens,omitempty"`
	AvgReadsBeforeEdit   float64                 `json:"avg_reads_before_first_edit,omitempty"`
	AvgPromptsBeforeEdit float64                 `json:"avg_prompts_before_first_edit,omitempty"`
	TopPrecedingReads    []RepoAnalysisPathCount `json:"top_preceding_reads,omitempty"`
	TopRelatedSubsystems []RepoAnalysisNameCount `json:"top_related_subsystems,omitempty"`
}

type RepoAnalysisPathCount struct {
	Path     string `json:"path"`
	Sessions int    `json:"sessions"`
}

type RepoAnalysisNameCount struct {
	Name     string `json:"name"`
	Sessions int    `json:"sessions"`
}

type RepoAnalysisDiscoverability struct {
	SearchHeavyTargets []RepoAnalysisSearchTarget    `json:"search_heavy_targets,omitempty"`
	AmbiguousSearches  []RepoAnalysisAmbiguousSearch `json:"ambiguous_searches,omitempty"`
	DeadEndSearches    int                           `json:"dead_end_searches,omitempty"`
}

type RepoAnalysisSearchTarget struct {
	Path                          string                   `json:"path"`
	FoundViaSearchSessions        int                      `json:"found_via_search_sessions"`
	AvgSearchesBeforeEdit         float64                  `json:"avg_searches_before_edit"`
	AvgReadsAfterSearchBeforeEdit float64                  `json:"avg_reads_after_search_before_edit"`
	TopQueries                    []RepoAnalysisQueryCount `json:"top_queries,omitempty"`
	TargetConfidence              string                   `json:"target_confidence"`
}

type RepoAnalysisAmbiguousSearch struct {
	Query               string `json:"query"`
	Searches            int    `json:"searches"`
	DistinctReadTargets int    `json:"distinct_read_targets"`
	DistinctEditTargets int    `json:"distinct_edit_targets"`
}

type RepoAnalysisQueryCount struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

type RepoAnalysisDoc struct {
	Path             string   `json:"path"`
	Sessions         int      `json:"sessions"`
	Reads            int      `json:"reads"`
	EditSessions     int      `json:"edit_sessions"`
	ReadBeforeEditOf []string `json:"read_before_edit_of,omitempty"`
	Classification   []string `json:"classification,omitempty"`
}
