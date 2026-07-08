package model

import (
	"errors"
	"strings"
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
	for _, item := range topic.KnownWorkflows {
		ensureSection("known_workflows")
		ensureItem("known_workflows", item)
	}
	for _, item := range topic.AvoidWastingTime {
		ensureSection("avoid_wasting_time")
		ensureItem("avoid_wasting_time", item)
	}
	for _, item := range topic.RiskFlags {
		ensureSection("risk_flags")
		ensureItem("risk_flags", item)
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

func editableTopicStringSlice(topic *TopicContext, section string) *[]string {
	switch section {
	case "known_workflows":
		return &topic.KnownWorkflows
	case "avoid_wasting_time":
		return &topic.AvoidWastingTime
	case "risk_flags":
		return &topic.RiskFlags
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

// EditTopicEntry overwrites the text of a topic's summary or an item within an
// editable section, re-keying provenance for string-array items since their
// item key is derived from their own text.
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
		slice := editableTopicStringSlice(topic, section)
		if slice == nil {
			return ErrTopicEntryNotEditable
		}
		for i, v := range *slice {
			if v != itemKey {
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
