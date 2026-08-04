package experience

import (
	"fmt"
	"strings"
	"testing"

	"github.com/repoguide/repoguide-core/model"
)

func TestBuildTaskPackageUsesSimilarSessionBehavior(t *testing.T) {
	topic := model.TopicContext{
		Name:      "OAuth Login and Web Auth UI",
		StartHere: []model.TopicStartFile{{Path: "web/src/App.jsx"}, {Path: "backend/auth/auth_handlers.go"}, {Path: "backend/auth/jwt.go"}},
		ImportantFiles: model.TopicImportantFiles{
			EditTargets: []string{"web/src/App.jsx", "backend/auth/auth_handlers.go", "backend/auth/jwt.go"},
		},
	}
	sessions := []model.RepoSessionEvents{
		session("hide login button for authenticated users", []string{"web/src/App.jsx", "backend/auth/auth_handlers.go"}, 10),
		session("button should disappear when user is logged in", []string{"web/src/App.jsx", "backend/auth/auth_handlers.go"}, 12),
		session("hide authenticated navigation button", []string{"web/src/App.jsx", "backend/auth/jwt.go"}, 11),
		session("rotate production oauth secrets", []string{"backend/auth/jwt.go"}, 20),
	}

	pkg := BuildTaskPackage("button shouldn't appear when logged in", "/repo", topic, sessions)
	if !pkg.Behavioral || pkg.SimilarSessions != 3 {
		t.Fatalf("behavioral=%v sessions=%d, want true/3", pkg.Behavioral, pkg.SimilarSessions)
	}
	if len(pkg.Files) < 2 || pkg.Files[0].Path != "web/src/App.jsx" || pkg.Files[0].Sessions != 3 {
		t.Fatalf("unexpected files: %#v", pkg.Files)
	}
	if len(pkg.Avoid) != 1 || pkg.Avoid[0].Path != "backend/auth/jwt.go" {
		t.Fatalf("rare file should be separated from typical files, avoid=%#v", pkg.Avoid)
	}
	if pkg.MedianFiles != 2 || pkg.MedianToolCalls != 11 {
		t.Fatalf("median files/tool calls = %d/%d, want 2/11", pkg.MedianFiles, pkg.MedianToolCalls)
	}
	if !strings.Contains(pkg.Boundary, "Cross-layer") {
		t.Fatalf("boundary = %q, want cross-layer", pkg.Boundary)
	}
}

func TestBuildTaskPackageMarksRareTopicFilesAsAvoid(t *testing.T) {
	topic := model.TopicContext{
		Name:      "OAuth Login and Web Auth UI",
		StartHere: []model.TopicStartFile{{Path: "web/src/App.jsx"}, {Path: "backend/auth/jwt.go"}},
	}
	var sessions []model.RepoSessionEvents
	for i := 0; i < 6; i++ {
		files := []string{"web/src/App.jsx"}
		if i == 0 {
			files = append(files, "backend/auth/jwt.go")
		}
		sessions = append(sessions, session("hide auth button", files, 8+i))
	}
	pkg := BuildTaskPackage("hide auth button", "/repo", topic, sessions)
	if len(pkg.Avoid) != 1 || pkg.Avoid[0].Path != "backend/auth/jwt.go" || pkg.Avoid[0].Sessions != 1 {
		t.Fatalf("avoid = %#v, want jwt.go at 1/6", pkg.Avoid)
	}
	if pkg.Boundary != "Frontend only (5/6 sessions)" && pkg.Boundary != "Usually frontend only (5/6 sessions)" {
		t.Fatalf("boundary = %q", pkg.Boundary)
	}
}

func TestBuildTaskPackageFallsBackToTopicStartFiles(t *testing.T) {
	topic := model.TopicContext{
		Name:     "Profiles",
		Evidence: model.TopicEvidence{Sessions: 1},
		StartHere: []model.TopicStartFile{
			{Path: "model/user.go"}, {Path: "api/users.go"}, {Path: "web/Profile.jsx"},
		},
		KnownWorkflows: []model.TopicGuidanceItem{
			{ID: "workflow-profile", Text: "Update the user model and its profile form."},
			{ID: "workflow-deploy", Text: "Run every deployment check."},
		},
		AvoidWastingTime: []model.TopicGuidanceItem{
			{ID: "avoid-generated", Text: "Do not edit generated clients."},
			{ID: "avoid-oauth", Text: "Do not change OAuth secrets."},
			{ID: "avoid-third", Text: "A third warning."},
		},
		ScopeBoundaries: []model.TopicGuidanceItem{{ID: "scope-profile", Text: "Profile fields and display-name changes."}},
	}
	pkg := BuildTaskPackage("add first and last name to user profiles", "", topic, nil)
	if pkg.Behavioral {
		t.Fatalf("fallback package must not claim session behavior: %#v", pkg)
	}
	if len(adviceOfKind(pkg.SelectedAdvice, "start_file")) == 0 {
		t.Fatalf("fallback must return topic start-file orientation: %#v", pkg.SelectedAdvice)
	}
	for _, item := range pkg.SelectedAdvice {
		if item.Source != "topic_context" {
			t.Fatalf("fallback advice must be sourced from topic context: %#v", pkg.SelectedAdvice)
		}
	}
	// Curated guidance that shares the task vocabulary is surfaced, highest
	// confidence first; guidance from unrelated work is not.
	if !hasAdviceText(pkg.SelectedAdvice, "Update the user model and its profile form.") {
		t.Fatalf("relevant curated workflow must be surfaced: %#v", pkg.SelectedAdvice)
	}
	if hasAdviceText(pkg.SelectedAdvice, "Run every deployment check.") {
		t.Fatalf("unrelated curated guidance must not be surfaced: %#v", pkg.SelectedAdvice)
	}
}

func hasAdviceText(items []AdviceItem, text string) bool {
	for _, item := range items {
		if item.Text == text {
			return true
		}
	}
	return false
}

func TestBuildTaskPackageWithOneSessionReturnsOrientationOnly(t *testing.T) {
	topic := model.TopicContext{ID: "hooks", Name: "MCP Hooks", StartHere: []model.TopicStartFile{{Path: "internal/mcp/hooks.go"}}}
	sessions := []model.RepoSessionEvents{{ID: "hook-session", Events: []model.SessionEvent{
		{Kind: "prompt", Text: "install local mcp hook"},
		{Kind: "tool_call", WritePaths: []string{"/repo/internal/mcp/hooks.go", "/repo/run-local.sh"}},
		{Kind: "tool_call", CommandText: "./run-local.sh"},
	}}}
	pkg := BuildTaskPackage("install local mcp hook", "/repo", topic, sessions)
	if !pkg.Behavioral || pkg.SimilarSessions != 1 || len(pkg.Workflow) != 0 || pkg.Boundary != "" || len(pkg.Avoid) != 0 {
		t.Fatalf("one session must produce orientation only: %#v", pkg)
	}
	for _, advice := range pkg.CandidateAdvice {
		if advice.Kind != "start_file" && advice.Kind != "test" {
			t.Fatalf("one session emitted unsupported advice: %#v", advice)
		}
	}
	if got := RenderTaskPackage(topic, pkg); !strings.Contains(got, "No high-confidence task-specific workflow") {
		t.Fatalf("orientation response must explain its limit: %q", got)
	}
}

func TestBuildTaskPackageWithoutEvidenceSurfacesCuratedStartFiles(t *testing.T) {
	topic := model.TopicContext{Name: "New Topic", StartHere: []model.TopicStartFile{{Path: "guessed.go"}}}
	pkg := BuildTaskPackage("unrelated task", "", topic, nil)
	// A freshly matched topic starts from its curated orientation rather than
	// returning empty; feedback degrades it if it proves unhelpful.
	if len(adviceOfKind(pkg.SelectedAdvice, "start_file")) == 0 {
		t.Fatalf("matched topic must surface curated start files: %#v", pkg.SelectedAdvice)
	}
	if got := RenderTaskPackage(topic, pkg); !strings.Contains(got, "curated orientation") {
		t.Fatalf("no-session response must explain it is curated orientation: %q", got)
	}
}

func TestBuildTaskPackageWithEmptyTopicReturnsNoAdvice(t *testing.T) {
	topic := model.TopicContext{Name: "Empty Topic"}
	pkg := BuildTaskPackage("unrelated task", "", topic, nil)
	if len(pkg.SelectedAdvice) != 0 {
		t.Fatalf("topic with no curated content must not invent advice: %#v", pkg.SelectedAdvice)
	}
	if got := RenderTaskPackage(topic, pkg); !strings.Contains(got, "No completed sessions currently provide enough evidence") {
		t.Fatalf("no-evidence response must say so: %q", got)
	}
}

func TestApplyAdviceFeedbackRaisesAndLowersItemConfidence(t *testing.T) {
	pkg := TaskPackage{
		CandidateAdvice: []AdviceItem{
			{ID: "helpful", Support: 8, Total: 10, Confidence: 0.8},
			{ID: "unhelpful", Support: 8, Total: 10, Confidence: 0.8},
		},
		Budget: BudgetForTask(1, false),
	}
	feedback := []model.MCPFeedback{
		{AdviceEvaluation: &model.AdviceEvaluation{HelpfulAdviceIDs: []string{"helpful"}}},
		{AdviceEvaluation: &model.AdviceEvaluation{UnhelpfulAdviceIDs: []string{"unhelpful"}}},
	}

	pkg = ApplyAdviceFeedback(pkg, feedback)
	if pkg.CandidateAdvice[0].Confidence <= 0.8 {
		t.Fatalf("helpful confidence = %.3f, want > 0.8", pkg.CandidateAdvice[0].Confidence)
	}
	if pkg.CandidateAdvice[1].Confidence >= 0.8 {
		t.Fatalf("unhelpful confidence = %.3f, want < 0.8", pkg.CandidateAdvice[1].Confidence)
	}
}

func TestApplyAdviceFeedbackMapsTextAndFilesToStableAdviceIDs(t *testing.T) {
	pkg := TaskPackage{
		CandidateAdvice: []AdviceItem{
			{ID: "workflow", Text: "Update the Go model and TypeScript mirror together", Steps: []string{"Edit models.go", "Edit api.ts"}, Confidence: 0.7, Total: 5},
			{ID: "avoid", Text: "vite.config.ts is rarely involved", Files: []string{"vite.config.ts"}, Confidence: 0.7, Total: 5},
		},
		Budget: BudgetForTask(1, false),
	}
	feedback := []model.MCPFeedback{{AdviceEvaluation: &model.AdviceEvaluation{
		UsefulAdvice:   "The Go model and TypeScript mirror workflow was useful.",
		UnhelpfulFiles: []string{"vite.config.ts"},
	}}}

	pkg = ApplyAdviceFeedback(pkg, feedback)
	if pkg.CandidateAdvice[0].HelpfulTextMatches != 1 || pkg.CandidateAdvice[0].HelpfulFeedback != 1 {
		t.Fatalf("workflow text feedback was not traced: %#v", pkg.CandidateAdvice[0])
	}
	if pkg.CandidateAdvice[1].UnhelpfulTextMatches != 1 || pkg.CandidateAdvice[1].UnhelpfulFeedback != 1 {
		t.Fatalf("file feedback was not traced: %#v", pkg.CandidateAdvice[1])
	}
}

func TestFeedbackForTopicKeepsAdviceLearningScoped(t *testing.T) {
	feedback := []model.MCPFeedback{
		{TopicID: "wanted", Task: "matching"},
		{TopicID: "other", Task: "unrelated"},
		{Task: "legacy without a route"},
	}
	got := FeedbackForTopic("wanted", feedback)
	if len(got) != 1 || got[0].Task != "matching" {
		t.Fatalf("filtered feedback = %#v", got)
	}
}

func TestFeedbackForSessionsKeepsCorrectionsInRetrievedPopulation(t *testing.T) {
	feedback := []model.MCPFeedback{
		{SessionID: "matched", Task: "matching correction"},
		{SessionID: "other", Task: "unrelated correction"},
	}
	got := FeedbackForSessions([]string{"matched"}, feedback)
	if len(got) != 1 || got[0].Task != "matching correction" {
		t.Fatalf("filtered feedback = %#v", got)
	}
}

func TestCandidatesCarryOnlyRetrievedSessionEvidence(t *testing.T) {
	topic := model.TopicContext{ID: "auth", StartHere: []model.TopicStartFile{{Path: "web/App.jsx"}}, KnownWorkflows: []model.TopicGuidanceItem{{ID: "parent", Text: "Do unrelated profile work", Provenance: model.TopicProvenance{SessionID: "other"}}}}
	sessions := []model.RepoSessionEvents{
		{ID: "a", Events: []model.SessionEvent{{Kind: "prompt", Text: "hide login autofill"}, {Kind: "tool_call", WritePaths: []string{"/repo/web/App.jsx", "/repo/web/auth.test.jsx"}}, {Kind: "tool_call", CommandText: "npm test -- auth"}}},
		{ID: "b", Events: []model.SessionEvent{{Kind: "prompt", Text: "hide login autofill"}, {Kind: "tool_call", WritePaths: []string{"/repo/web/App.jsx", "/repo/web/auth.test.jsx"}}, {Kind: "tool_call", CommandText: "npm test -- auth"}}},
		{ID: "unrelated", Events: []model.SessionEvent{{Kind: "prompt", Text: "rotate oauth secrets"}, {Kind: "tool_call", WritePaths: []string{"/repo/auth/jwt.go"}}}},
	}
	pkg := BuildTaskPackage("hide login autofill", "/repo", topic, sessions)
	if pkg.SimilarSessions != 2 || len(pkg.MatchingSessionIDs) != 2 {
		t.Fatalf("retrieved sessions = %#v", pkg)
	}
	methods := map[string]bool{}
	for _, candidate := range pkg.CandidateAdvice {
		if candidate.Evidence.Population != 2 || candidate.Evidence.Support > candidate.Evidence.Population {
			t.Fatalf("invalid candidate evidence: %#v", candidate)
		}
		for _, id := range candidate.Evidence.MatchingSessionIDs {
			if id != "a" && id != "b" {
				t.Fatalf("candidate leaked non-retrieved session %q: %#v", id, candidate)
			}
		}
		if candidate.ID == "parent" || candidate.Source == "known_workflows" {
			t.Fatalf("parent workflow leaked into runtime candidates: %#v", candidate)
		}
		methods[candidate.Evidence.ExtractionMethod] = true
	}
	for _, method := range []string{"edited_file", "coedited_sequence", "observed_command"} {
		if !methods[method] {
			t.Fatalf("missing session-derived %s candidate: %#v", method, pkg.CandidateAdvice)
		}
	}
}

func TestAdviceForClientOmitsInternalSessionIDs(t *testing.T) {
	items := AdviceForClient([]AdviceItem{{ID: "a", Evidence: CandidateEvidence{MatchingSessionIDs: []string{"internal-session"}, Support: 1, Population: 2, ExtractionMethod: "edited_file"}}})
	if len(items[0].Evidence.MatchingSessionIDs) != 0 || items[0].Evidence.Support != 1 || items[0].Evidence.Population != 2 {
		t.Fatalf("client advice should retain aggregate evidence only: %#v", items)
	}
}

func TestSelectAdviceAlwaysKeepsStartFileAndRejectsInventedIDs(t *testing.T) {
	pkg := TaskPackage{
		CandidateAdvice: []AdviceItem{
			{ID: "known", Text: "App.tsx", Kind: "start_file", Confidence: 0.9},
			{ID: "workflow", Text: "Run the relevant test", Kind: "workflow", Confidence: 0.8},
		},
		Budget: BudgetForTask(1, false),
	}

	empty := SelectAdvice(pkg, AdviceSelectionResponse{})
	if len(empty.SelectedAdvice) == 0 || empty.SelectedAdvice[0].ID != "known" {
		t.Fatalf("empty selector response must fall back to a start file: %#v", empty.SelectedAdvice)
	}

	withoutStart := SelectAdvice(pkg, AdviceSelectionResponse{Workflows: []string{"workflow"}})
	if len(withoutStart.SelectedAdvice) != 2 || withoutStart.SelectedAdvice[0].ID != "known" {
		t.Fatalf("selector response without a start file must receive one: %#v", withoutStart.SelectedAdvice)
	}

	invented := SelectAdvice(pkg, AdviceSelectionResponse{StartFiles: []string{"invented"}})
	if len(invented.SelectedAdvice) == 0 || invented.SelectedAdvice[0].ID != "known" {
		t.Fatalf("invalid selector response should use deterministic fallback: %#v", invented.SelectedAdvice)
	}
}

func TestFilterTopicToKnownFiles(t *testing.T) {
	topic := model.TopicContext{
		StartHere: []model.TopicStartFile{{Path: "keep.go"}, {Path: "gone.go"}},
		ImportantFiles: model.TopicImportantFiles{
			EditTargets: []string{"keep.go", "gone.go"},
		},
		Evidence: model.TopicEvidence{RepeatedEditedFiles: []string{"keep.go", "gone.go"}},
		RiskFlags: []model.TopicGuidanceItem{
			{ID: "r1", Text: "careful here", Files: []string{"keep.go", "gone.go"}},
		},
	}

	if got := FilterTopicToKnownFiles(topic, nil); len(got.StartHere) != 2 {
		t.Fatalf("nil known should skip filtering, got %#v", got.StartHere)
	}

	filtered := FilterTopicToKnownFiles(topic, map[string]bool{"keep.go": true})
	if len(filtered.StartHere) != 1 || filtered.StartHere[0].Path != "keep.go" {
		t.Fatalf("StartHere = %#v, want only keep.go", filtered.StartHere)
	}
	if len(filtered.ImportantFiles.EditTargets) != 1 || filtered.ImportantFiles.EditTargets[0] != "keep.go" {
		t.Fatalf("EditTargets = %#v, want only keep.go", filtered.ImportantFiles.EditTargets)
	}
	if len(filtered.Evidence.RepeatedEditedFiles) != 1 {
		t.Fatalf("RepeatedEditedFiles = %#v, want only keep.go", filtered.Evidence.RepeatedEditedFiles)
	}
	if len(filtered.RiskFlags) != 1 || len(filtered.RiskFlags[0].Files) != 1 || filtered.RiskFlags[0].Files[0] != "keep.go" {
		t.Fatalf("RiskFlags = %#v, want files trimmed to keep.go", filtered.RiskFlags)
	}
}

func TestFilterTaskPackageFiles(t *testing.T) {
	pkg := TaskPackage{
		Files: []FilePattern{{Path: "keep.go", Sessions: 3}, {Path: "gone.go", Sessions: 1}},
		CandidateAdvice: []AdviceItem{
			{ID: "a", Files: []string{"keep.go", "gone.go"}},
		},
	}

	if got := FilterTaskPackageFiles(pkg, nil); len(got.Files) != 2 {
		t.Fatalf("nil known should skip filtering, got %#v", got.Files)
	}

	filtered := FilterTaskPackageFiles(pkg, map[string]bool{"keep.go": true})
	if len(filtered.Files) != 1 || filtered.Files[0].Path != "keep.go" {
		t.Fatalf("Files = %#v, want only keep.go", filtered.Files)
	}
	if len(filtered.CandidateAdvice[0].Files) != 1 || filtered.CandidateAdvice[0].Files[0] != "keep.go" {
		t.Fatalf("CandidateAdvice[0].Files = %#v, want only keep.go", filtered.CandidateAdvice[0].Files)
	}
}

func session(prompt string, files []string, toolCalls int) model.RepoSessionEvents {
	events := []model.SessionEvent{{Kind: "prompt", Text: prompt}}
	for i := 0; i < toolCalls-1; i++ {
		events = append(events, model.SessionEvent{Kind: "tool_call", ToolName: "Read"})
	}
	abs := make([]string, 0, len(files))
	for _, file := range files {
		abs = append(abs, "/repo/"+file)
	}
	events = append(events, model.SessionEvent{Kind: "tool_call", ToolName: "Edit", WritePaths: abs})
	return model.RepoSessionEvents{Events: events}
}

// A real session's tool results carry role "user" as well as its prompts. When
// sessionPromptTokens counted them, the candidate token set grew from dozens to
// thousands and tokenSimilarity's sqrt(len(task)*len(candidate)) denominator
// pushed every session under minimumSessionSimilarity, so no repository ever
// produced task-similar evidence.
func TestBuildTaskPackageIgnoresToolOutputWhenScoringSimilarity(t *testing.T) {
	topic := model.TopicContext{
		Name:      "OAuth Login and Web Auth UI",
		StartHere: []model.TopicStartFile{{Path: "web/src/App.jsx"}},
	}
	// Distinct tokens, not a repeated phrase: tokenSet dedupes, and it's the
	// breadth of tool output that inflates the similarity denominator.
	var noise strings.Builder
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&noise, "symbol%d ", i)
	}
	var sessions []model.RepoSessionEvents
	for i := 0; i < 3; i++ {
		s := session("hide login button for authenticated users", []string{"web/src/App.jsx"}, 9+i)
		s.Events = append(s.Events, model.SessionEvent{Kind: "tool_result", Role: "user", Text: noise.String()})
		sessions = append(sessions, s)
	}

	pkg := BuildTaskPackage("hide login button for authenticated users", "/repo", topic, sessions)
	if !pkg.Behavioral || pkg.SimilarSessions != 3 {
		t.Fatalf("behavioral=%v sessions=%d, want true/3 - tool output must not dilute prompt similarity", pkg.Behavioral, pkg.SimilarSessions)
	}
}

// Scoring a session by the union of its prompts made the similarity
// denominator grow with session length, so a long session that did the exact
// task in one of its turns scored below a short session that merely brushed
// past it. Sessions are scored by their closest single prompt instead.
func TestBuildTaskPackageScoresSessionsByClosestPrompt(t *testing.T) {
	topic := model.TopicContext{
		Name:      "Integration Tests",
		StartHere: []model.TopicStartFile{{Path: "integration-test/run-all.sh"}},
	}
	var sessions []model.RepoSessionEvents
	for i := 0; i < 3; i++ {
		s := session("move the team-e2e folder into integration-test and wire it into run-all.sh",
			[]string{"integration-test/run-all.sh"}, 9+i)
		// The same long tail of unrelated turns every real session accumulates.
		// Each contributes fresh vocabulary - tokenSet dedupes, so repeating one
		// phrase would not grow the union the way real follow-ups do.
		for j := 0; j < 40; j++ {
			s.Events = append(s.Events, model.SessionEvent{
				Kind: "prompt",
				Text: fmt.Sprintf("follow%d up%d about%d topic%d word%d term%d phrase%d item%d", j, j, j, j, j, j, j, j),
			})
		}
		sessions = append(sessions, s)
	}

	pkg := BuildTaskPackage("move the team-e2e folder into integration-test and wire it into run-all.sh", "/repo", topic, sessions)
	if !pkg.Behavioral || pkg.SimilarSessions != 3 {
		t.Fatalf("behavioral=%v sessions=%d, want true/3 - later unrelated turns must not bury a matching prompt", pkg.Behavioral, pkg.SimilarSessions)
	}
}

// Behavioral packages used to build candidates purely from the session
// population, so a topic that gained enough matching sessions silently lost
// its curated warnings - the opposite of what more evidence should do.
func TestBehavioralPackageKeepsCuratedWarnings(t *testing.T) {
	topic := model.TopicContext{
		Name:      "Integration Tests",
		StartHere: []model.TopicStartFile{{Path: "integration-test/run-all.sh"}},
		AvoidWastingTime: []model.TopicGuidanceItem{
			{ID: "avoid-gitmv", Text: "git mv team-e2e integration-test/team-e2e failed twice; use plain mv instead."},
		},
	}
	var sessions []model.RepoSessionEvents
	for i := 0; i < 3; i++ {
		sessions = append(sessions, session("move team-e2e into integration-test and wire it into run-all.sh",
			[]string{"integration-test/run-all.sh"}, 9+i))
	}

	pkg := BuildTaskPackage("move team-e2e into integration-test and wire it into run-all.sh", "/repo", topic, sessions)
	if !pkg.Behavioral {
		t.Fatalf("expected a behavioral package, got %#v", pkg)
	}
	if !hasAdviceText(pkg.CandidateAdvice, "git mv team-e2e integration-test/team-e2e failed twice; use plain mv instead.") {
		t.Fatalf("curated warning must survive session matching: %#v", pkg.CandidateAdvice)
	}
	// Session evidence still outranks it.
	if pkg.CandidateAdvice[0].Source != "session_history" {
		t.Fatalf("session-derived advice must rank first, got %q", pkg.CandidateAdvice[0].Source)
	}
}

// Observed commands were replayed verbatim as "test" advice, so sessions
// contributed machine-specific paths and pinned versions that are wrong by the
// time anyone reads them.
func TestTestAdviceRejectsUnreplayableCommands(t *testing.T) {
	topic := model.TopicContext{
		Name:      "CLI Release",
		StartHere: []model.TopicStartFile{{Path: "brew/VERSION"}},
	}
	commands := []string{
		"cd /Users/scy2be/Documents/repoguide && ls",     // machine-specific
		`git commit -m "chore: bump VERSION to v0.22.3"`, // stale pinned version
		"sed -n '720,920p' web/src/App.jsx",              // one person scrolling a file
		"go test ./...",                                  // reusable
	}
	var sessions []model.RepoSessionEvents
	for i := 0; i < 3; i++ {
		s := session("bump the cli version and align the brew tap", []string{"brew/VERSION"}, 9+i)
		s.ID = fmt.Sprintf("release-session-%d", i)
		for _, c := range commands {
			s.Events = append(s.Events, model.SessionEvent{Kind: "tool_call", CommandText: c})
		}
		sessions = append(sessions, s)
	}

	pkg := BuildTaskPackage("bump the cli version and align the brew tap", "/repo", topic, sessions)
	for _, item := range pkg.CandidateAdvice {
		if item.Kind != "test" {
			continue
		}
		if strings.Contains(item.Text, "/Users/") {
			t.Fatalf("machine-specific command surfaced as advice: %q", item.Text)
		}
		if versionLiteral.MatchString(item.Text) {
			t.Fatalf("pinned version surfaced as advice: %q", item.Text)
		}
		if lineRangeRead.MatchString(item.Text) {
			t.Fatalf("line-range read surfaced as advice: %q", item.Text)
		}
	}
	if !hasAdviceText(pkg.CandidateAdvice, "go test ./...") {
		t.Fatalf("reusable command must survive filtering: %#v", pkg.CandidateAdvice)
	}
}

// Commands were harvested from session history regardless of outcome, so a
// command that failed came back as advice - most visibly suggesting `git mv`
// right beside a curated warning saying git mv fails in this repo.
func TestTestAdviceExcludesFailedCommands(t *testing.T) {
	topic := model.TopicContext{
		Name:      "Integration Tests",
		StartHere: []model.TopicStartFile{{Path: "integration-test/run-all.sh"}},
	}
	var sessions []model.RepoSessionEvents
	for i := 0; i < 3; i++ {
		s := session("move team-e2e into integration-test and wire it into run-all.sh",
			[]string{"integration-test/run-all.sh"}, 9+i)
		s.ID = fmt.Sprintf("e2e-session-%d", i)
		s.Events = append(s.Events,
			model.SessionEvent{Kind: "tool_call", ToolCallID: "call-fail", CommandText: "git mv team-e2e integration-test/team-e2e"},
			model.SessionEvent{Kind: "tool_result", ToolCallID: "call-fail", IsError: true},
			model.SessionEvent{Kind: "tool_call", ToolCallID: "call-ok", CommandText: "mv team-e2e integration-test/team-e2e"},
			model.SessionEvent{Kind: "tool_result", ToolCallID: "call-ok"},
		)
		sessions = append(sessions, s)
	}

	pkg := BuildTaskPackage("move team-e2e into integration-test and wire it into run-all.sh", "/repo", topic, sessions)
	if hasAdviceText(pkg.CandidateAdvice, "git mv team-e2e integration-test/team-e2e") {
		t.Fatalf("a command that errored must not be advice: %#v", pkg.CandidateAdvice)
	}
	if !hasAdviceText(pkg.CandidateAdvice, "mv team-e2e integration-test/team-e2e") {
		t.Fatalf("the command that succeeded must survive: %#v", pkg.CandidateAdvice)
	}
}

// At one or two matching sessions every edit count is identical, so churn
// cannot order the files and the first one named was arbitrary - a task about
// `repoguide sessions` led with cmd/stats.go. Task-word overlap breaks that
// tie, without overriding a genuine churn signal.
func TestBehavioralStartFilesBreakChurnTiesByRelevance(t *testing.T) {
	topic := model.TopicContext{
		Name: "Session Views",
		StartHere: []model.TopicStartFile{
			{Path: "repoguide-cli/cmd/sessions.go"}, {Path: "repoguide-cli/cmd/stats.go"},
		},
	}
	var sessions []model.RepoSessionEvents
	for i := 0; i < 2; i++ {
		// Both files edited by both sessions: churn is a dead heat.
		sessions = append(sessions, session("add a filter flag to repoguide sessions",
			[]string{"repoguide-cli/cmd/stats.go", "repoguide-cli/cmd/sessions.go"}, 9+i))
	}

	pkg := BuildTaskPackage("add a --since filter flag to repoguide sessions", "/repo", topic, sessions)
	if len(pkg.Files) == 0 {
		t.Fatalf("expected ranked files, got none: %#v", pkg)
	}
	if pkg.Files[0].Path != "repoguide-cli/cmd/sessions.go" {
		t.Fatalf("start file = %q, want cmd/sessions.go - the task names sessions, not stats", pkg.Files[0].Path)
	}
}
