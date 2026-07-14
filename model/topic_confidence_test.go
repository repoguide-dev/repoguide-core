package model

import "testing"

func TestApplyTopicFeedbackConfidenceMovesWithRatings(t *testing.T) {
	topic := TopicContext{ID: "topic", Confidence: 0.60}
	positive := ApplyTopicFeedbackConfidence(topic, []MCPFeedback{{TopicID: "topic", Stars: 4}})
	negative := ApplyTopicFeedbackConfidence(topic, []MCPFeedback{{TopicID: "topic", Stars: 3}})
	if positive.Confidence <= topic.Confidence || negative.Confidence >= topic.Confidence {
		t.Fatalf("confidence movement positive=%f negative=%f base=%f", positive.Confidence, negative.Confidence, topic.Confidence)
	}
}
