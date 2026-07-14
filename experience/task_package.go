package experience

import (
	"crypto/sha256"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/repoguide/repoguide-core/contracts/v1"
	"github.com/repoguide/repoguide-core/model"
)

const maxSimilarSessions = 20

type FilePattern struct {
	Path     string
	Sessions int
}

type AdviceItem = contracts.AdviceItem
type TopicRoutingExample = contracts.TopicRoutingExample
type SelectionBudget = contracts.SelectionBudget
type AdviceSelectionResponse = contracts.AdviceSelectionResponse

// TaskPackage is a compact, deterministic view of repository behavior for one
// task. Topic prose is used only as a bounded fallback when session evidence is
// unavailable.
type TaskPackage struct {
	SimilarSessions int
	Files           []FilePattern
	Avoid           []FilePattern
	Workflow        []string
	Warnings        []string
	Boundary        string
	MedianFiles     int
	MedianToolCalls int
	Behavioral      bool
	CandidateAdvice []AdviceItem
	SelectedAdvice  []AdviceItem
	Budget          SelectionBudget
}

type preparedSession struct {
	log       model.RepoSessionEvents
	edits     []string
	toolCalls int
	score     float64
}

// BuildTaskPackage selects sessions that overlap both the task language and
// the selected topic's file boundary, then aggregates their observed edits.
func BuildTaskPackage(task, repoRoot string, topic model.TopicContext, sessions []model.RepoSessionEvents) TaskPackage {
	known := topicFiles(topic)
	taskTokens := tokenSet(task)
	prepared := make([]preparedSession, 0, len(sessions))
	for _, session := range sessions {
		p := prepareSession(repoRoot, session)
		if len(p.edits) == 0 {
			continue
		}
		overlap := topicFileOverlap(p.edits, known)
		if len(known) > 0 && overlap == 0 {
			continue
		}
		promptScore := tokenSimilarity(taskTokens, sessionPromptTokens(session))
		if promptScore == 0 {
			continue
		}
		p.score = promptScore + 0.25*overlap
		prepared = append(prepared, p)
	}
	sort.SliceStable(prepared, func(i, j int) bool {
		if prepared[i].score == prepared[j].score {
			return prepared[i].log.UpdatedAt.After(prepared[j].log.UpdatedAt)
		}
		return prepared[i].score > prepared[j].score
	})
	if len(prepared) > maxSimilarSessions {
		prepared = prepared[:maxSimilarSessions]
	}
	if len(prepared) == 0 {
		return fallbackTaskPackage(task, topic)
	}
	return behavioralTaskPackage(topic, prepared)
}

func behavioralTaskPackage(topic model.TopicContext, sessions []preparedSession) TaskPackage {
	counts := map[string]int{}
	positions := map[string][]int{}
	fileTotals := make([]int, 0, len(sessions))
	toolTotals := make([]int, 0, len(sessions))
	for _, session := range sessions {
		fileTotals = append(fileTotals, len(session.edits))
		toolTotals = append(toolTotals, session.toolCalls)
		for pos, path := range session.edits {
			counts[path]++
			positions[path] = append(positions[path], pos)
		}
	}

	files := make([]FilePattern, 0, len(counts))
	for path, count := range counts {
		files = append(files, FilePattern{Path: path, Sessions: count})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Sessions != files[j].Sessions {
			return files[i].Sessions > files[j].Sessions
		}
		pi, pj := averagePosition(positions[files[i].Path]), averagePosition(positions[files[j].Path])
		if pi != pj {
			return pi < pj
		}
		return files[i].Path < files[j].Path
	})
	typical := files[:0]
	for _, file := range files {
		if file.Sessions*2 >= len(sessions) {
			typical = append(typical, file)
		}
		if len(typical) == 5 {
			break
		}
	}
	files = typical

	workflow := append([]FilePattern(nil), files...)
	sort.SliceStable(workflow, func(i, j int) bool {
		pi, pj := averagePosition(positions[workflow[i].Path]), averagePosition(positions[workflow[j].Path])
		if pi == pj {
			return workflow[i].Sessions > workflow[j].Sessions
		}
		return pi < pj
	})
	workflowPaths := make([]string, 0, min(3, len(workflow)))
	for _, file := range workflow {
		if len(workflowPaths) == 3 {
			break
		}
		workflowPaths = append(workflowPaths, file.Path)
	}

	avoid := lowSupportTopicFiles(topic, counts, len(sessions), files)
	warnings := []string(nil)
	if len(sessions) < 3 {
		warnings = append(warnings, fmt.Sprintf("Only %d similar implementation %s found; treat the pattern as tentative.", len(sessions), plural(len(sessions), "session", "sessions")))
	}

	pkg := TaskPackage{
		SimilarSessions: len(sessions),
		Files:           files,
		Avoid:           avoid,
		Workflow:        workflowPaths,
		Warnings:        warnings,
		Boundary:        expectedScope(topic, sessions),
		MedianFiles:     median(fileTotals),
		MedianToolCalls: median(toolTotals),
		Behavioral:      true,
	}
	pkg.CandidateAdvice = buildCandidateAdvice(topic, pkg, sessions)
	pkg.Budget = BudgetForTask(1, isCrossCutting(topic, pkg))
	pkg.SelectedAdvice = defaultAdviceSelection(pkg.CandidateAdvice, pkg.Budget)
	return pkg
}

func fallbackTaskPackage(task string, topic model.TopicContext) TaskPackage {
	files := rankedTopicFiles(task, topic)
	workflows := rankText(task, model.TopicGuidanceTexts(topic.KnownWorkflows), 1)
	warnings := append([]string(nil), model.TopicGuidanceTexts(topic.AvoidWastingTime)...)
	warnings = append(warnings, model.TopicGuidanceTexts(topic.RiskFlags)...)
	warnings = rankText(task, warnings, 2)
	boundaries := rankText(task, model.TopicGuidanceTexts(topic.ScopeBoundaries), 1)
	boundary := ""
	if len(boundaries) > 0 {
		boundary = boundaries[0]
	}
	pkg := TaskPackage{Files: files, Workflow: workflows, Warnings: warnings, Boundary: boundary}
	pkg.CandidateAdvice = buildCandidateAdvice(topic, pkg, nil)
	pkg.Budget = BudgetForTask(1, isCrossCutting(topic, pkg))
	pkg.SelectedAdvice = defaultAdviceSelection(pkg.CandidateAdvice, pkg.Budget)
	return pkg
}

func RenderTaskPackage(topic model.TopicContext, pkg TaskPackage) string {
	var sb strings.Builder
	if pkg.Behavioral {
		sb.WriteString("Repository experience\n")
		fmt.Fprintf(&sb, "Task-similar sessions: %d\n", pkg.SimilarSessions)
	} else {
		sb.WriteString("Task package\n")
		fmt.Fprintf(&sb, "Topic: %s\n", topic.Name)
		sb.WriteString("No similar implementation session found; showing a bounded topic fallback.\n")
	}
	if len(pkg.SelectedAdvice) > 0 {
		for _, group := range adviceGroups {
			items := adviceOfKind(pkg.SelectedAdvice, group.kind)
			if len(items) == 0 {
				continue
			}
			fmt.Fprintf(&sb, "\n%s\n", group.heading)
			for _, advice := range items {
				fmt.Fprintf(&sb, "- %s\n", advice.Text)
				for _, step := range advice.Steps {
					fmt.Fprintf(&sb, "  - %s\n", step)
				}
			}
		}
	}
	if pkg.Behavioral {
		sb.WriteString("\nMedian\n")
		fmt.Fprintf(&sb, "%d files · %d tool calls\n", pkg.MedianFiles, pkg.MedianToolCalls)
	}
	return strings.TrimSpace(sb.String())
}

// ApplyAdviceFeedback adjusts confidence using explicit item-level feedback.
// The underlying evidence text and support counts remain immutable.
func ApplyAdviceFeedback(pkg TaskPackage, feedback []model.MCPFeedback) TaskPackage {
	helpful := map[string]int{}
	unhelpful := map[string]int{}
	helpfulText := map[string]int{}
	unhelpfulText := map[string]int{}
	for _, item := range feedback {
		if item.AdviceEvaluation == nil {
			continue
		}
		for _, id := range item.AdviceEvaluation.HelpfulAdviceIDs {
			helpful[id]++
		}
		for _, id := range item.AdviceEvaluation.UnhelpfulAdviceIDs {
			unhelpful[id]++
		}
		for _, candidate := range pkg.CandidateAdvice {
			if !contains(item.AdviceEvaluation.HelpfulAdviceIDs, candidate.ID) && adviceEvaluationMatches(candidate, item.AdviceEvaluation.UsefulAdvice, item.AdviceEvaluation.UsefulFiles) {
				helpfulText[candidate.ID]++
			}
			negativeText := strings.TrimSpace(item.AdviceEvaluation.IncorrectAdvice + " " + item.AdviceEvaluation.UnnecessaryAdvice)
			if !contains(item.AdviceEvaluation.UnhelpfulAdviceIDs, candidate.ID) && adviceEvaluationMatches(candidate, negativeText, item.AdviceEvaluation.UnhelpfulFiles) {
				unhelpfulText[candidate.ID]++
			}
		}
	}
	for i := range pkg.CandidateAdvice {
		item := &pkg.CandidateAdvice[i]
		item.HelpfulTextMatches = helpfulText[item.ID]
		item.UnhelpfulTextMatches = unhelpfulText[item.ID]
		item.HelpfulFeedback = helpful[item.ID] + item.HelpfulTextMatches
		item.UnhelpfulFeedback = unhelpful[item.ID] + item.UnhelpfulTextMatches
		priorWeight := float64(max(item.Total, item.Support) + 2)
		priorConfidence := item.Confidence
		if priorConfidence <= 0 || priorConfidence >= 1 {
			priorConfidence = float64(item.Support+1) / priorWeight
		}
		positive := priorConfidence*priorWeight + float64(item.HelpfulFeedback)
		observations := priorWeight + float64(item.HelpfulFeedback+item.UnhelpfulFeedback)
		item.Confidence = positive / observations
	}
	pkg.SelectedAdvice = defaultAdviceSelection(pkg.CandidateAdvice, pkg.Budget)
	return pkg
}

// FeedbackForTopic prevents prose about one topic from changing the quality
// score of a similarly worded advice item in another topic.
func FeedbackForTopic(topicID string, feedback []model.MCPFeedback) []model.MCPFeedback {
	filtered := make([]model.MCPFeedback, 0, len(feedback))
	for _, item := range feedback {
		if item.TopicID == topicID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func adviceEvaluationMatches(item AdviceItem, feedbackText string, feedbackFiles []string) bool {
	for _, file := range feedbackFiles {
		if file == item.Text || contains(item.Files, file) {
			return true
		}
	}
	feedbackText = strings.ToLower(strings.TrimSpace(feedbackText))
	if feedbackText == "" {
		return false
	}
	texts := append([]string{item.Text}, item.Steps...)
	texts = append(texts, item.Files...)
	for _, text := range texts {
		normalized := strings.ToLower(strings.TrimSpace(text))
		if len(normalized) >= 4 && strings.Contains(feedbackText, normalized) {
			return true
		}
	}
	feedbackTokens := tokenSet(feedbackText)
	itemTokens := tokenSet(strings.Join(texts, " "))
	if len(feedbackTokens) == 0 || len(itemTokens) == 0 {
		return false
	}
	overlap := 0
	for token := range itemTokens {
		if _, ok := feedbackTokens[token]; ok {
			overlap++
		}
	}
	return overlap >= 2 && float64(overlap)/float64(min(len(itemTokens), len(feedbackTokens))) >= 0.5
}

// SelectAdvice restricts the final package to IDs returned by the selector.
// Unknown IDs, category mismatches, duplicates, and over-budget entries are
// discarded. The selector never controls advice text.
func SelectAdvice(pkg TaskPackage, selection AdviceSelectionResponse) TaskPackage {
	requested := len(selectionIDs(selection))
	pkg.SelectedAdvice = validateAdviceSelection(pkg.CandidateAdvice, selection, pkg.Budget)
	if requested > 0 && len(pkg.SelectedAdvice) == 0 {
		pkg.SelectedAdvice = defaultAdviceSelection(pkg.CandidateAdvice, pkg.Budget)
	}
	return pkg
}

func SetSelectionBudget(pkg TaskPackage, matchCount int) TaskPackage {
	crossCutting := pkg.Budget.MaxTotal > 10 || strings.Contains(strings.ToLower(pkg.Boundary), "cross-layer")
	pkg.Budget = BudgetForTask(matchCount, crossCutting)
	pkg.SelectedAdvice = defaultAdviceSelection(pkg.CandidateAdvice, pkg.Budget)
	return pkg
}

func BudgetForTask(matchCount int, crossCutting bool) SelectionBudget {
	if crossCutting || matchCount > 1 {
		return SelectionBudget{MaxTotal: 15, MaxPerCategory: 4, MinDistinctKinds: 4, MaxCharacters: 6000}
	}
	return SelectionBudget{MaxTotal: 10, MaxPerCategory: 3, MinDistinctKinds: 2, MaxCharacters: 4000}
}

// BuildTopicRoutingExamples admits only feedback with a strong positive route
// signal. Wrong-topic feedback is kept separately as a negative example.
func BuildTopicRoutingExamples(topicID string, feedback []model.MCPFeedback) (positive, negative []TopicRoutingExample) {
	allPositive, negative := BuildRoutingExamples(feedback)
	for _, example := range allPositive {
		if example.TopicID == topicID {
			positive = append(positive, example)
		}
	}
	if len(positive) > 8 {
		positive = positive[:8]
	}
	return positive, negative
}

func BuildRoutingExamples(feedback []model.MCPFeedback) (positive, negative []TopicRoutingExample) {
	for _, item := range feedback {
		if strings.TrimSpace(item.Task) == "" || strings.TrimSpace(item.TopicID) == "" {
			continue
		}
		wrong := containsWrongTopicSignal(item)
		if wrong {
			negative = append(negative, TopicRoutingExample{Task: item.Task, TopicID: item.TopicID, Feedback: "rejected", Reason: firstNonEmpty(item.WhatWentWrong, item.MissingContext)})
			continue
		}
		if !positiveTopicFeedback(item) {
			continue
		}
		positive = append(positive, TopicRoutingExample{Task: item.Task, TopicID: item.TopicID, Feedback: "high"})
	}
	if len(positive) > 16 {
		positive = positive[:16]
	}
	if len(negative) > 4 {
		negative = negative[:4]
	}
	return positive, negative
}

func buildCandidateAdvice(topic model.TopicContext, pkg TaskPackage, sessions []preparedSession) []AdviceItem {
	model.EnsureTopicProvenance(&topic)
	items := make([]AdviceItem, 0, 32)
	// Topic-wide evidence is deliberately not used as a denominator for a
	// task candidate. It inflates support for claims that were never observed
	// in the task-similar session set.
	total := max(1, pkg.SimilarSessions)
	for _, file := range pkg.Files {
		items = append(items, newAdvice(topic.ID, "start_file", "history-file:"+file.Path,
			file.Path, file.Sessions, max(1, pkg.SimilarSessions), "session_history", ""))
	}
	if len(pkg.Workflow) >= 2 {
		support := coEditSupport(pkg.Workflow[:2], sessions)
		items = append(items, newAdvice(topic.ID, "workflow", "coedit:"+strings.Join(pkg.Workflow[:2], "|"),
			fmt.Sprintf("%s and %s are commonly edited together (%d/%d sessions).", pkg.Workflow[0], pkg.Workflow[1], support, max(1, pkg.SimilarSessions)), support, max(1, pkg.SimilarSessions), "session_history", ""))
	}
	for _, file := range pkg.Avoid {
		support := max(1, pkg.SimilarSessions) - file.Sessions
		items = append(items, newAdvice(topic.ID, "avoid", "avoid:"+file.Path,
			fmt.Sprintf("%s is rarely involved in similar work (edited in %d/%d sessions).", file.Path, file.Sessions, max(1, pkg.SimilarSessions)), support, max(1, pkg.SimilarSessions), "session_history", ""))
	}
	if pkg.Behavioral && pkg.Boundary != "" {
		behaviorTotal := max(1, pkg.SimilarSessions)
		support := scopeSupport(pkg.Boundary, behaviorTotal)
		items = append(items, newAdvice(topic.ID, "scope_boundary", "scope:"+pkg.Boundary, pkg.Boundary+".", support, behaviorTotal, "session_history", ""))
	}

	for _, file := range topic.StartHere {
		items = append(items, topicPathAdvice(topic, "start_file", "start_here", file.Path))
	}
	for _, path := range topic.ImportantFiles.EditTargets {
		items = append(items, topicPathAdvice(topic, "start_file", "important_files.edit_targets", path))
	}
	for _, path := range topic.ImportantFiles.ReferenceFiles {
		items = append(items, topicPathAdvice(topic, "start_file", "important_files.reference_files", path))
	}
	for _, path := range topic.ImportantFiles.CrossCuttingFiles {
		items = append(items, topicPathAdvice(topic, "start_file", "important_files.cross_cutting_files", path))
	}
	for _, path := range topic.ImportantFiles.TestFiles {
		items = append(items, topicPathAdvice(topic, "test", "important_files.test_files", path))
	}
	for _, path := range topic.Tests.StartWith {
		items = append(items, topicPathAdvice(topic, "test", "tests.start_with", path))
	}
	if strings.TrimSpace(topic.Tests.Signal) != "" {
		items = append(items, newAdvice(topic.ID, "test", "tests.signal", topic.Tests.Signal, topic.Evidence.Sessions, total, "tests.signal", topic.Evidence.LastActive))
	}
	for _, command := range topic.Tests.Commands {
		items = append(items, newAdvice(topic.ID, "test", "tests.command:"+command, command, topic.Evidence.Sessions, total, "tests.commands", topic.Evidence.LastActive))
	}
	// Topic guidance is retained in the topic record, but it is not emitted as
	// production advice until it has item-level observed provenance. This
	// quarantines legacy authored workflows and warnings.
	items = append(items, guidanceAdvice(topic.KnownWorkflows, "workflow", "known_workflows", total)...)
	items = append(items, guidanceAdvice(topic.AvoidWastingTime, "avoid", "avoid_wasting_time", total)...)
	items = append(items, guidanceAdvice(topic.Tests.Notes, "test", "tests.notes", total)...)
	items = append(items, guidanceAdvice(topic.ScopeBoundaries, "scope_boundary", "scope_boundaries", total)...)
	items = append(items, guidanceAdvice(topic.RiskFlags, "risk", "risk_flags", total)...)
	return dedupeAdvice(items)
}

func newAdvice(topicID, kind, key, text string, support, total int, source, lastObserved string) AdviceItem {
	sum := sha256.Sum256([]byte(topicID + "\x00" + kind + "\x00" + key))
	confidence := float64(support+1) / float64(total+2)
	return AdviceItem{ID: fmt.Sprintf("a_%x", sum[:6]), Text: text, Kind: kind, Support: support, Total: total, Confidence: confidence, Source: source, LastObservedAt: lastObserved}
}

func topicPathAdvice(topic model.TopicContext, kind, section, path string) AdviceItem {
	prov := topic.ItemProvenance[model.TopicItemProvenanceKey(section, path)]
	support := prov.SupportSessionCount
	return newAdvice(topic.ID, kind, section+":"+path, path, support, max(1, support), section, prov.LastSupportedAt)
}

func guidanceAdvice(guidance []model.TopicGuidanceItem, kind, section string, total int) []AdviceItem {
	items := make([]AdviceItem, 0, len(guidance))
	for _, item := range guidance {
		if item.ID == "" || strings.TrimSpace(item.Text) == "" || item.Provenance.SupportSessionCount == 0 {
			continue
		}
		lastObserved := ""
		if !item.LastObservedAt.IsZero() {
			lastObserved = item.LastObservedAt.Format("2006-01-02")
		}
		itemTotal := item.SuccessCount
		if itemTotal < item.SupportCount {
			itemTotal = item.SupportCount
		}
		if itemTotal == 0 {
			itemTotal = item.Provenance.SupportSessionCount
		}
		if itemTotal == 0 {
			continue
		}
		confidence := float64(item.SupportCount+1) / float64(itemTotal+2)
		items = append(items, AdviceItem{
			ID: item.ID, Text: item.Text, Steps: append([]string(nil), item.Steps...), Files: append([]string(nil), item.Files...), Severity: item.Severity,
			Kind: kind, Support: item.SupportCount,
			Total: itemTotal, Confidence: confidence, Source: section,
			LastObservedAt: lastObserved,
		})
	}
	return items
}

func defaultAdviceSelection(items []AdviceItem, budget SelectionBudget) []AdviceItem {
	ranked := append([]AdviceItem(nil), items...)
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Confidence == ranked[j].Confidence {
			if ranked[i].Support == ranked[j].Support {
				return ranked[i].ID < ranked[j].ID
			}
			return ranked[i].Support > ranked[j].Support
		}
		return ranked[i].Confidence > ranked[j].Confidence
	})
	target := min(budget.MaxTotal, 8)
	if budget.MaxTotal > 10 {
		target = min(budget.MaxTotal, 12)
	}
	selection := AdviceSelectionResponse{}
	perKind := map[string]int{}
	characters := 0
	for pass := 0; pass < 2; pass++ {
		for _, item := range ranked {
			if len(selectionIDs(selection)) >= target || perKind[item.Kind] >= budget.MaxPerCategory || contains(selectionIDs(selection), item.ID) {
				continue
			}
			if pass == 0 && perKind[item.Kind] > 0 {
				continue
			}
			if characters+adviceCharacters(item) > budget.MaxCharacters {
				continue
			}
			addSelectionID(&selection, item.Kind, item.ID)
			perKind[item.Kind]++
			characters += adviceCharacters(item)
		}
	}
	return validateAdviceSelection(items, selection, budget)
}

func validateAdviceSelection(items []AdviceItem, selection AdviceSelectionResponse, budget SelectionBudget) []AdviceItem {
	byID := make(map[string]AdviceItem, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	selected := make([]AdviceItem, 0, min(budget.MaxTotal, len(items)))
	seen := map[string]bool{}
	perKind := map[string]int{}
	characters := 0
	for _, group := range selectionGroups(selection) {
		for _, id := range group.ids {
			if len(selected) >= budget.MaxTotal || seen[id] || perKind[group.kind] >= budget.MaxPerCategory {
				continue
			}
			item, ok := byID[id]
			if !ok || item.Kind != group.kind || characters+adviceCharacters(item) > budget.MaxCharacters {
				continue
			}
			seen[id] = true
			perKind[group.kind]++
			characters += adviceCharacters(item)
			selected = append(selected, item)
		}
	}
	return selected
}

func adviceCharacters(item AdviceItem) int {
	total := len([]rune(item.Text))
	for _, step := range item.Steps {
		total += len([]rune(step))
	}
	for _, file := range item.Files {
		total += len([]rune(file))
	}
	return total
}

var adviceGroups = []struct {
	kind    string
	heading string
}{
	{kind: "start_file", heading: "Start files"},
	{kind: "workflow", heading: "Workflows"},
	{kind: "avoid", heading: "Avoid"},
	{kind: "test", heading: "Tests"},
	{kind: "scope_boundary", heading: "Scope boundaries"},
	{kind: "risk", heading: "Risks"},
}

type selectionGroup struct {
	kind string
	ids  []string
}

func selectionGroups(selection AdviceSelectionResponse) []selectionGroup {
	return []selectionGroup{
		{kind: "start_file", ids: selection.StartFiles},
		{kind: "workflow", ids: selection.Workflows},
		{kind: "avoid", ids: selection.Avoid},
		{kind: "test", ids: selection.Tests},
		{kind: "scope_boundary", ids: selection.ScopeBoundaries},
		{kind: "risk", ids: selection.Risks},
	}
}

func selectionIDs(selection AdviceSelectionResponse) []string {
	var ids []string
	for _, group := range selectionGroups(selection) {
		ids = append(ids, group.ids...)
	}
	return ids
}

func addSelectionID(selection *AdviceSelectionResponse, kind, id string) {
	switch kind {
	case "start_file":
		selection.StartFiles = append(selection.StartFiles, id)
	case "workflow":
		selection.Workflows = append(selection.Workflows, id)
	case "avoid":
		selection.Avoid = append(selection.Avoid, id)
	case "test":
		selection.Tests = append(selection.Tests, id)
	case "scope_boundary":
		selection.ScopeBoundaries = append(selection.ScopeBoundaries, id)
	case "risk":
		selection.Risks = append(selection.Risks, id)
	}
}

func adviceOfKind(items []AdviceItem, kind string) []AdviceItem {
	out := make([]AdviceItem, 0)
	for _, item := range items {
		if item.Kind == kind {
			out = append(out, item)
		}
	}
	return out
}

func dedupeAdvice(items []AdviceItem) []AdviceItem {
	byIdentity := map[string]int{}
	out := make([]AdviceItem, 0, len(items))
	for _, item := range items {
		identity := item.Kind + "\x00" + strings.TrimSpace(item.Text) + "\x00" + strings.Join(item.Steps, "\x00") + "\x00" + strings.Join(item.Files, "\x00")
		if existing, ok := byIdentity[identity]; ok {
			if item.Confidence > out[existing].Confidence {
				out[existing] = item
			}
			continue
		}
		byIdentity[identity] = len(out)
		out = append(out, item)
	}
	return out
}

func isCrossCutting(topic model.TopicContext, pkg TaskPackage) bool {
	if strings.Contains(strings.ToLower(pkg.Boundary), "cross-layer") || len(topic.ImportantFiles.CrossCuttingFiles) > 0 {
		return true
	}
	layers := map[string]bool{}
	for _, file := range pkg.Files {
		layers[fileLayer(topic.Name, file.Path)] = true
	}
	return len(layers) > 1
}

func coEditSupport(paths []string, sessions []preparedSession) int {
	support := 0
	for _, session := range sessions {
		matched := 0
		for _, expected := range paths {
			for _, actual := range session.edits {
				if samePath(actual, expected) {
					matched++
					break
				}
			}
		}
		if matched == len(paths) {
			support++
		}
	}
	return support
}

func scopeSupport(boundary string, total int) int {
	var support, denominator int
	if start := strings.LastIndex(boundary, "("); start >= 0 {
		if _, err := fmt.Sscanf(boundary[start:], "(%d/%d sessions)", &support, &denominator); err == nil && denominator == total {
			return support
		}
	}
	return max(1, total/2)
}

func positiveTopicFeedback(item model.MCPFeedback) bool {
	if item.Helpfulness == "none" || item.Helpfulness == "low" || item.Stars > 0 && item.Stars < 4 {
		return false
	}
	if contains(item.HelpedWith, "topic_routing") && (item.Helpfulness == "medium" || item.Helpfulness == "high") {
		return true
	}
	return item.Stars >= 4
}

func containsWrongTopicSignal(item model.MCPFeedback) bool {
	text := strings.ToLower(strings.Join([]string{item.WhatWentWrong, item.MissingContext, item.WhatCouldBeImproved}, " "))
	for _, signal := range []string{"wrong topic", "topic was wrong", "wrong route", "misrouted", "another topic", "different topic"} {
		if strings.Contains(text, signal) {
			return true
		}
	}
	return item.Helpfulness == "none" && contains(item.HelpedWith, "topic_routing")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return "Topic routing was not useful."
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func prepareSession(repoRoot string, session model.RepoSessionEvents) preparedSession {
	seen := map[string]struct{}{}
	edits := make([]string, 0)
	toolCalls := 0
	for _, event := range session.Events {
		if event.Kind == "tool_call" {
			toolCalls++
		}
		for _, raw := range event.WritePaths {
			path := displayPath(repoRoot, raw)
			if path == "" {
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			edits = append(edits, path)
		}
	}
	return preparedSession{log: session, edits: edits, toolCalls: toolCalls}
}

func displayPath(repoRoot, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	clean := filepath.Clean(raw)
	if repoRoot != "" {
		if rel, err := filepath.Rel(filepath.Clean(repoRoot), clean); err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			clean = rel
		}
	}
	return strings.TrimPrefix(filepath.ToSlash(clean), "./")
}

func topicFiles(topic model.TopicContext) []string {
	paths := make([]string, 0)
	for _, file := range topic.StartHere {
		paths = append(paths, file.Path)
	}
	paths = append(paths, topic.ImportantFiles.EditTargets...)
	paths = append(paths, topic.ImportantFiles.ReferenceFiles...)
	paths = append(paths, topic.ImportantFiles.CrossCuttingFiles...)
	return dedupePaths(paths)
}

func topicFileOverlap(edits, known []string) float64 {
	if len(known) == 0 || len(edits) == 0 {
		return 0
	}
	matches := 0
	for _, edit := range edits {
		for _, candidate := range known {
			if samePath(edit, candidate) {
				matches++
				break
			}
		}
	}
	return float64(matches) / float64(len(edits))
}

func samePath(a, b string) bool {
	a = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(a)), "./")
	b = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(b)), "./")
	return a == b || strings.HasSuffix(a, "/"+b) || strings.HasSuffix(b, "/"+a)
}

func sessionPromptTokens(session model.RepoSessionEvents) map[string]struct{} {
	var prompts []string
	for _, event := range session.Events {
		if (event.Kind == "prompt" || event.Role == "user") && strings.TrimSpace(event.Text) != "" {
			prompts = append(prompts, event.Text)
		}
	}
	return tokenSet(strings.Join(prompts, " "))
}

func tokenSimilarity(task, candidate map[string]struct{}) float64 {
	if len(task) == 0 || len(candidate) == 0 {
		return 0
	}
	overlap := 0
	for token := range task {
		if _, ok := candidate[token]; ok {
			overlap++
		}
	}
	if overlap == 0 {
		return 0
	}
	return float64(overlap) / math.Sqrt(float64(len(task)*len(candidate)))
}

func tokenSet(value string) map[string]struct{} {
	parts := strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	out := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		token := normalizeToken(part)
		if token == "" || stopWords[token] {
			continue
		}
		out[token] = struct{}{}
	}
	return out
}

func normalizeToken(token string) string {
	switch token {
	case "authenticated", "authentication", "authorize", "authorization", "login", "logged", "signin", "session":
		return "auth"
	case "frontend", "ui", "clientside":
		return "frontend"
	case "backend", "serverside":
		return "backend"
	case "profiles":
		return "profile"
	case "users":
		return "user"
	}
	if len(token) > 5 && strings.HasSuffix(token, "ing") {
		token = strings.TrimSuffix(token, "ing")
	} else if len(token) > 4 && strings.HasSuffix(token, "ed") {
		token = strings.TrimSuffix(token, "ed")
	} else if len(token) > 4 && strings.HasSuffix(token, "s") {
		token = strings.TrimSuffix(token, "s")
	}
	return token
}

var stopWords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true, "be": true, "by": true,
	"change": true, "do": true, "for": true, "from": true, "in": true, "it": true, "make": true,
	"of": true, "on": true, "or": true, "please": true, "should": true, "that": true, "the": true,
	"this": true, "to": true, "when": true, "with": true, "would": true,
}

func lowSupportTopicFiles(topic model.TopicContext, counts map[string]int, sessionCount int, selected []FilePattern) []FilePattern {
	if sessionCount < 2 {
		return nil
	}
	selectedSet := map[string]struct{}{}
	for _, file := range selected {
		selectedSet[file.Path] = struct{}{}
	}
	testSet := map[string]struct{}{}
	for _, path := range topic.ImportantFiles.TestFiles {
		testSet[path] = struct{}{}
	}
	threshold := max(1, sessionCount/4)
	avoid := make([]FilePattern, 0)
	for _, path := range topicFiles(topic) {
		if _, ok := selectedSet[path]; ok {
			continue
		}
		if _, ok := testSet[path]; ok || looksLikeTest(path) {
			continue
		}
		count := countMatchingPath(counts, path)
		if count <= threshold {
			avoid = append(avoid, FilePattern{Path: path, Sessions: count})
		}
	}
	sort.SliceStable(avoid, func(i, j int) bool {
		if avoid[i].Sessions == avoid[j].Sessions {
			return avoid[i].Path < avoid[j].Path
		}
		return avoid[i].Sessions > avoid[j].Sessions
	})
	if len(avoid) > 2 {
		avoid = avoid[:2]
	}
	return avoid
}

func countMatchingPath(counts map[string]int, candidate string) int {
	for path, count := range counts {
		if samePath(path, candidate) {
			return count
		}
	}
	return 0
}

func looksLikeTest(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "test") || strings.Contains(lower, "spec")
}

func expectedScope(topic model.TopicContext, sessions []preparedSession) string {
	counts := map[string]int{}
	for _, session := range sessions {
		layers := map[string]struct{}{}
		for _, path := range session.edits {
			layers[fileLayer(topic.Name, path)] = struct{}{}
		}
		if len(layers) == 1 {
			for layer := range layers {
				counts[layer]++
			}
		}
	}
	best, bestCount := "", 0
	for layer, count := range counts {
		if count > bestCount || (count == bestCount && layer < best) {
			best, bestCount = layer, count
		}
	}
	if best != "" && bestCount*4 >= len(sessions)*3 {
		prefix := best + " only"
		if bestCount != len(sessions) {
			prefix = "Usually " + strings.ToLower(best) + " only"
		}
		return fmt.Sprintf("%s (%d/%d sessions)", prefix, bestCount, len(sessions))
	}
	return "Cross-layer changes were typical in similar sessions"
}

func fileLayer(topicName, path string) string {
	lower := strings.ToLower(filepath.ToSlash(path))
	ext := strings.ToLower(filepath.Ext(lower))
	switch {
	case strings.Contains(lower, "/web/") || strings.HasPrefix(lower, "web/") || strings.Contains(lower, "/frontend/") || strings.HasPrefix(lower, "frontend/") || ext == ".jsx" || ext == ".tsx" || ext == ".vue" || ext == ".css":
		return "Frontend"
	case strings.Contains(lower, "/backend/") || strings.HasPrefix(lower, "backend/") || strings.Contains(lower, "/server/") || strings.HasPrefix(lower, "server/"):
		return "Backend"
	case strings.Contains(lower, "/k8s/") || strings.HasPrefix(lower, "k8s/") || strings.Contains(lower, "deploy"):
		return "Infrastructure"
	case strings.Contains(strings.ToLower(topicName), "cli") || strings.Contains(lower, "/cli/") || strings.HasPrefix(lower, "cli/") || strings.HasPrefix(lower, "cmd/"):
		return "CLI"
	default:
		return "Single subsystem"
	}
}

func rankedTopicFiles(task string, topic model.TopicContext) []FilePattern {
	type candidate struct {
		path  string
		text  string
		order int
		score int
	}
	items := make([]candidate, 0)
	for _, file := range topic.StartHere {
		items = append(items, candidate{path: file.Path, text: file.Path + " " + file.Role + " " + file.Why, order: len(items)})
	}
	for _, path := range append(append(append([]string{}, topic.ImportantFiles.EditTargets...), topic.ImportantFiles.ReferenceFiles...), topic.ImportantFiles.CrossCuttingFiles...) {
		items = append(items, candidate{path: path, text: path, order: len(items)})
	}
	taskTokens := tokenSet(task)
	seen := map[string]struct{}{}
	deduped := make([]candidate, 0, len(items))
	for _, item := range items {
		if item.path == "" {
			continue
		}
		if _, ok := seen[item.path]; ok {
			continue
		}
		seen[item.path] = struct{}{}
		for token := range tokenSet(item.text) {
			if _, ok := taskTokens[token]; ok {
				item.score++
			}
		}
		deduped = append(deduped, item)
	}
	sort.SliceStable(deduped, func(i, j int) bool {
		if deduped[i].score == deduped[j].score {
			return deduped[i].order < deduped[j].order
		}
		return deduped[i].score > deduped[j].score
	})
	if len(deduped) > 5 {
		deduped = deduped[:5]
	}
	out := make([]FilePattern, 0, len(deduped))
	for _, item := range deduped {
		out = append(out, FilePattern{Path: item.path})
	}
	return out
}

func rankText(task string, values []string, limit int) []string {
	type ranked struct {
		value string
		order int
		score int
	}
	taskTokens := tokenSet(task)
	items := make([]ranked, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		item := ranked{value: value, order: len(items)}
		for token := range tokenSet(value) {
			if _, ok := taskTokens[token]; ok {
				item.score++
			}
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score == items[j].score {
			return items[i].order < items[j].order
		}
		return items[i].score > items[j].score
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.value)
	}
	return out
}

func dedupePaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./")
		if path == "" || path == "." {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}

func averagePosition(values []int) float64 {
	if len(values) == 0 {
		return math.MaxFloat64
	}
	total := 0
	for _, value := range values {
		total += value
	}
	return float64(total) / float64(len(values))
}

func median(values []int) int {
	if len(values) == 0 {
		return 0
	}
	values = append([]int(nil), values...)
	sort.Ints(values)
	mid := len(values) / 2
	if len(values)%2 == 1 {
		return values[mid]
	}
	return int(math.Round(float64(values[mid-1]+values[mid]) / 2))
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
