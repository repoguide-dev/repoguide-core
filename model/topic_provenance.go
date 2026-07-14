package model

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	TopicProvenanceSourceSession      = "session"
	TopicProvenanceSourceUserFeedback = "user_feedback"

	TopicStatusActive    = "active"
	TopicStatusConfirmed = "confirmed"
	TopicStatusObsolete  = "obsolete"
)

func DefaultTopicProvenance(evidence TopicEvidence) TopicProvenance {
	return TopicProvenance{
		SourceType:          TopicProvenanceSourceSession,
		SupportSessionCount: evidence.Sessions,
		LastSupportedAt:     evidence.LastActive,
		Status:              TopicStatusActive,
	}
}

func EnsureTopicProvenance(topic *TopicContext) {
	if topic == nil {
		return
	}
	if topic.SectionProvenance == nil {
		topic.SectionProvenance = map[string]TopicProvenance{}
	}
	if topic.ItemProvenance == nil {
		topic.ItemProvenance = map[string]TopicProvenance{}
	}

	defaultProv := DefaultTopicProvenance(topic.Evidence)
	ensureSection := func(section string) {
		if _, ok := topic.SectionProvenance[section]; !ok {
			topic.SectionProvenance[section] = defaultProv
		}
	}
	ensureItem := func(section, itemKey string) {
		if itemKey == "" {
			return
		}
		key := TopicItemProvenanceKey(section, itemKey)
		if _, ok := topic.ItemProvenance[key]; !ok {
			topic.ItemProvenance[key] = defaultProv
		}
	}

	ensureSection("summary")
	for _, item := range topic.WhenToUse {
		ensureSection("when_to_use")
		ensureItem("when_to_use", item)
	}
	for _, item := range topic.PromptKeywords {
		ensureSection("prompt_keywords")
		ensureItem("prompt_keywords", item)
	}
	for i := range topic.ScopeBoundaries {
		item := &topic.ScopeBoundaries[i]
		EnsureTopicGuidanceItem(topic.ID, "scope_boundaries", item, topic.Evidence)
		ensureSection("scope_boundaries")
		ensureItem("scope_boundaries", item.ID)
	}
	for i := range topic.KnownWorkflows {
		item := &topic.KnownWorkflows[i]
		EnsureTopicGuidanceItem(topic.ID, "known_workflows", item, topic.Evidence)
		ensureSection("known_workflows")
		ensureItem("known_workflows", item.ID)
	}
	for i := range topic.AvoidWastingTime {
		item := &topic.AvoidWastingTime[i]
		EnsureTopicGuidanceItem(topic.ID, "avoid_wasting_time", item, topic.Evidence)
		ensureSection("avoid_wasting_time")
		ensureItem("avoid_wasting_time", item.ID)
	}
	for i := range topic.RiskFlags {
		item := &topic.RiskFlags[i]
		EnsureTopicGuidanceItem(topic.ID, "risk_flags", item, topic.Evidence)
		ensureSection("risk_flags")
		ensureItem("risk_flags", item.ID)
	}
	for i := range topic.Tests.Notes {
		item := &topic.Tests.Notes[i]
		EnsureTopicGuidanceItem(topic.ID, "tests.notes", item, topic.Evidence)
		ensureSection("tests.notes")
		ensureItem("tests.notes", item.ID)
	}
	for _, file := range topic.StartHere {
		ensureSection("start_here")
		ensureItem("start_here", file.Path)
	}

	fileSections := map[string][]string{
		"important_files.edit_targets":        topic.ImportantFiles.EditTargets,
		"important_files.reference_files":     topic.ImportantFiles.ReferenceFiles,
		"important_files.test_files":          topic.ImportantFiles.TestFiles,
		"important_files.cross_cutting_files": topic.ImportantFiles.CrossCuttingFiles,
	}
	for section, items := range fileSections {
		for _, item := range items {
			ensureSection(section)
			ensureItem(section, item)
		}
	}
}

func TopicItemProvenanceKey(section, itemKey string) string {
	if section == "" {
		return itemKey
	}
	return section + "::" + itemKey
}

func TopicStatusDisablesEntry(status string) bool {
	return status == TopicStatusObsolete
}

var (
	ErrTopicEntryNotEditable = errors.New("section is not editable")
	ErrTopicEntryNotFound    = errors.New("item not found")
)

func editableTopicGuidanceSlice(topic *TopicContext, section string) *[]TopicGuidanceItem {
	switch section {
	case "known_workflows":
		return &topic.KnownWorkflows
	case "avoid_wasting_time":
		return &topic.AvoidWastingTime
	case "risk_flags":
		return &topic.RiskFlags
	case "scope_boundaries":
		return &topic.ScopeBoundaries
	case "tests.notes":
		return &topic.Tests.Notes
	default:
		return nil
	}
}

func editableTopicPathSlice(topic *TopicContext, section string) *[]string {
	switch section {
	case "important_files.edit_targets":
		return &topic.ImportantFiles.EditTargets
	case "important_files.reference_files":
		return &topic.ImportantFiles.ReferenceFiles
	case "important_files.test_files":
		return &topic.ImportantFiles.TestFiles
	case "important_files.cross_cutting_files":
		return &topic.ImportantFiles.CrossCuttingFiles
	default:
		return nil
	}
}

func NewTopicGuidanceItem(topicID, section, text string, confidence float64, evidence TopicEvidence) TopicGuidanceItem {
	item := TopicGuidanceItem{Text: strings.TrimSpace(text), Confidence: confidence}
	EnsureTopicGuidanceItem(topicID, section, &item, evidence)
	return item
}

func EnsureTopicGuidanceItem(topicID, section string, item *TopicGuidanceItem, evidence TopicEvidence) {
	if item == nil {
		return
	}
	item.Text = strings.TrimSpace(item.Text)
	if item.ID == "" {
		sum := sha256.Sum256([]byte(topicID + "\x00" + section + "\x00" + item.Text + "\x00" + strings.Join(item.Steps, "\x00") + "\x00" + strings.Join(item.Files, "\x00")))
		item.ID = fmt.Sprintf("g_%x", sum[:6])
	}
	if item.Confidence <= 0 {
		item.Confidence = 0.5
	}
	if item.SupportCount == 0 {
		item.SupportCount = evidence.Sessions
	}
	if item.LastObservedAt.IsZero() && evidence.LastActive != "" {
		if parsed, err := time.Parse("2006-01-02", evidence.LastActive); err == nil {
			item.LastObservedAt = parsed
		}
	}
	if item.Provenance.SourceType == "" {
		item.Provenance = DefaultTopicProvenance(evidence)
	}
}

func TopicGuidanceTexts(items []TopicGuidanceItem) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item.Text != "" {
			out = append(out, item.Text)
		}
	}
	return out
}

// EditTopicEntry overwrites the text of a topic's summary or an item within an
// editable section. Structured guidance retains its stable item ID.
func EditTopicEntry(topic *TopicContext, section, itemKey, text, editor string) error {
	if topic == nil {
		return ErrTopicEntryNotFound
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("text is required")
	}
	EnsureTopicProvenance(topic)

	touchProvenance := func(key string) {
		prov := topic.ItemProvenance[key]
		if prov.SourceType == "" {
			prov = DefaultTopicProvenance(topic.Evidence)
		}
		prov.AddedBy = editor
		topic.ItemProvenance[key] = prov
	}

	switch section {
	case "summary":
		topic.Summary = text
		prov := topic.SectionProvenance["summary"]
		if prov.SourceType == "" {
			prov = DefaultTopicProvenance(topic.Evidence)
		}
		prov.AddedBy = editor
		topic.SectionProvenance["summary"] = prov
		return nil
	case "start_here":
		for i := range topic.StartHere {
			if topic.StartHere[i].Path != itemKey {
				continue
			}
			topic.StartHere[i].Why = text
			touchProvenance(TopicItemProvenanceKey(section, itemKey))
			return nil
		}
		return ErrTopicEntryNotFound
	default:
		if slice := editableTopicGuidanceSlice(topic, section); slice != nil {
			for i := range *slice {
				if (*slice)[i].ID != itemKey {
					continue
				}
				(*slice)[i].Text = text
				(*slice)[i].Provenance.AddedBy = editor
				touchProvenance(TopicItemProvenanceKey(section, itemKey))
				return nil
			}
			return ErrTopicEntryNotFound
		}
		slice := editableTopicPathSlice(topic, section)
		if slice == nil {
			return ErrTopicEntryNotEditable
		}
		for i, value := range *slice {
			if value != itemKey {
				continue
			}
			(*slice)[i] = text
			delete(topic.ItemProvenance, TopicItemProvenanceKey(section, itemKey))
			touchProvenance(TopicItemProvenanceKey(section, text))
			return nil
		}
		return ErrTopicEntryNotFound
	}
}
