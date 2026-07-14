package experience

import (
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
		Name: "Profiles",
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
	if len(pkg.SelectedAdvice) == 0 || pkg.SelectedAdvice[0].Kind != "start_file" || pkg.SelectedAdvice[0].Text != "model/user.go" {
		t.Fatalf("fallback must return the topic's best start file: %#v", pkg.SelectedAdvice)
	}
	for _, item := range pkg.SelectedAdvice {
		if item.Source != "topic_context" {
			t.Fatalf("fallback must not emit unrelated topic guidance: %#v", pkg.SelectedAdvice)
		}
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
