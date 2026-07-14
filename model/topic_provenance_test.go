package model

import "testing"

func TestGuidanceIDIncludesStructuredWorkflowContent(t *testing.T) {
	first := TopicGuidanceItem{Text: "Propagate the shared model", Steps: []string{"Edit Go model", "Update TypeScript mirror"}, Files: []string{"model.go", "api.ts"}, Confidence: 0.8}
	EnsureTopicGuidanceItem("shared-models", "known_workflows", &first, TopicEvidence{Sessions: 4})
	if first.ID == "" {
		t.Fatal("expected stable guidance ID")
	}

	same := TopicGuidanceItem{Text: first.Text, Steps: append([]string(nil), first.Steps...), Files: append([]string(nil), first.Files...), Confidence: 0.8}
	EnsureTopicGuidanceItem("shared-models", "known_workflows", &same, TopicEvidence{Sessions: 4})
	if same.ID != first.ID {
		t.Fatalf("same workflow ID = %q, want %q", same.ID, first.ID)
	}

	changed := TopicGuidanceItem{Text: first.Text, Steps: []string{"Edit canonical schema first"}, Files: append([]string(nil), first.Files...), Confidence: 0.8}
	EnsureTopicGuidanceItem("shared-models", "known_workflows", &changed, TopicEvidence{Sessions: 4})
	if changed.ID == first.ID {
		t.Fatal("workflow steps changed without changing the stable content ID")
	}
}
