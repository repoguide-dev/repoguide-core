package model

// ApplyTopicFeedbackConfidence updates a topic's routing confidence from
// explicit user ratings. Four- and five-star feedback is positive; one to
// three stars is corrective. Each signal has diminishing impact so a single
// rating cannot overturn accumulated session evidence.
func ApplyTopicFeedbackConfidence(topic TopicContext, feedback []MCPFeedback) TopicContext {
	for _, item := range feedback {
		if item.TopicID != "" && item.TopicID != topic.ID || item.Stars == 0 {
			continue
		}
		if item.Stars > 3 {
			topic.Confidence += (0.95 - topic.Confidence) * 0.08
		} else {
			topic.Confidence -= (topic.Confidence - 0.05) * 0.10
		}
	}
	return topic
}
