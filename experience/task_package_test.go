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

func TestBuildTaskPackageFallsBackToBoundedTopicData(t *testing.T) {
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
		t.Fatal("expected bounded topic fallback")
	}
	if len(pkg.Files) != 3 || len(pkg.Workflow) != 1 || len(pkg.Warnings) != 2 || pkg.Boundary == "" {
		t.Fatalf("unexpected fallback: %#v", pkg)
	}
	rendered := RenderTaskPackage(topic, pkg)
	if len(pkg.SelectedAdvice) > pkg.Budget.MaxTotal || len([]rune(rendered)) > pkg.Budget.MaxCharacters+500 {
		t.Fatalf("fallback exceeded selection budget: %#v\n%s", pkg.Budget, rendered)
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

func TestSelectAdviceHonorsEmptySelectionAndRejectsInventedIDs(t *testing.T) {
	pkg := TaskPackage{
		CandidateAdvice: []AdviceItem{{ID: "known", Text: "App.tsx", Kind: "start_file", Confidence: 0.9}},
		Budget:          BudgetForTask(1, false),
	}

	empty := SelectAdvice(pkg, AdviceSelectionResponse{})
	if len(empty.SelectedAdvice) != 0 {
		t.Fatalf("empty selector response should remain empty: %#v", empty.SelectedAdvice)
	}

	invented := SelectAdvice(pkg, AdviceSelectionResponse{StartFiles: []string{"invented"}})
	if len(invented.SelectedAdvice) != 1 || invented.SelectedAdvice[0].ID != "known" {
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
