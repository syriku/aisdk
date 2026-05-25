package request

import (
	"encoding/json"
	"fmt"
	"strings"
)

// GlossaryEntry represents a term mapping for consistency across the novel.
type GlossaryEntry struct {
	Source string `json:"source"` // Original term (e.g., character name in source language)
	Target string `json:"target"` // Translated term (e.g., character name in target language)
	Note   string `json:"note"`   // Optional context or description of the term
}

// ToJSON serializes the GlossaryEntry to a JSON byte slice.
func (g *GlossaryEntry) ToJSON() ([]byte, error) {
	return json.Marshal(g)
}

// ParseGlossaryEntryFromJSON deserializes a JSON byte slice into a GlossaryEntry.
func ParseGlossaryEntryFromJSON(data []byte) (*GlossaryEntry, error) {
	var g GlossaryEntry
	err := json.Unmarshal(data, &g)
	return &g, err
}

// TranslationContext maintains dynamic information during the translation process to ensure continuity.
type TranslationContext struct {
	PreviousSummary string          // Summary of previous plot points to help the AI understand the story
	RecentHistory   []string        // Recent translation history (last few paragraphs) for linguistic continuity
	Glossary        []GlossaryEntry // Active glossary mapping for the current translation request
}

// Language represents a language identifier defined in constant.go.
type Language int

// Translator holds the core configuration for translation style and few-shot learning.
type Translator struct {
	SourceLang  Language `json:"source_lang"`  // Source language (e.g., LAN_JP)
	TargetLang  Language `json:"target_lang"`  // Target language (e.g., LAN_ZH_CN)
	StylePrompt string   `json:"style_prompt"` // Instructions on writing style (e.g., "Modern Chinese light novel style")
	Template    string   `json:"template"`     // Short sample of high-quality human-translated text for few-shot learning
}

// ToJSON serializes the Translator to a JSON byte slice.
func (t *Translator) ToJSON() ([]byte, error) {
	return json.Marshal(t)
}

// ParseTranslatorFromJSON deserializes a JSON byte slice into a Translator.
func ParseTranslatorFromJSON(data []byte) (*Translator, error) {
	var t Translator
	err := json.Unmarshal(data, &t)
	return &t, err
}

// langNames maps language constants to their human-readable names.
var langNames = map[Language]string{
	LAN_ZH_CN: "Chinese (Simplified)",
	LAN_JP:    "Japanese",
	LAN_EN:    "English",
}

// GetLanguageName returns the string representation of a Language constant.
func GetLanguageName(lang Language) string {
	if name, ok := langNames[lang]; ok {
		return name
	}
	return "Unknown"
}

// GetLanguagesMap returns the map of Language constants to their human-readable names.
func GetLanguagesMap() map[Language]string {
	return langNames
}

// TranslateRequest contains all data needed for a single translation task.
type TranslateRequest struct {
	SourceText string             // The raw text to be translated
	Context    TranslationContext // The narrative context and terminology constraints
}

// GenerateSystemPrompt constructs the system instructions for the AI model.
// This typically includes the role, style, and persistent glossary rules.
func (t *Translator) GenerateSystemPrompt(glossary []GlossaryEntry) string {
	var sb strings.Builder

	// 1. Role and Style
	sb.WriteString("### Role and Style\n")
	fmt.Fprintf(&sb, "You are a professional novel translator. Translate from %s to %s.\n",
		GetLanguageName(t.SourceLang), GetLanguageName(t.TargetLang))
	sb.WriteString("The paragraph layout (line break positions) of the translated text must match the original text exactly, unless specified otherwise by the style guidelines below.\n")
	sb.WriteString("Follow this style and instruction:\n")
	sb.WriteString(t.StylePrompt)
	sb.WriteString("\n\n")

	// 2. Glossary/Terminology (Persistent part of the context)
	if len(glossary) > 0 {
		sb.WriteString("### Glossary and Terminology\n")
		sb.WriteString("Strictly use the following translations for these terms to maintain consistency:\n")
		for _, entry := range glossary {
			if entry.Note != "" {
				fmt.Fprintf(&sb, "- %s -> %s (%s)\n", entry.Source, entry.Target, entry.Note)
			} else {
				fmt.Fprintf(&sb, "- %s -> %s\n", entry.Source, entry.Target)
			}
		}
		sb.WriteString("\n")
	}

	// 3. Few-shot Template
	if t.Template != "" {
		sb.WriteString("### Reference Template\n")
		sb.WriteString("Use the following human-translated example as a reference for quality and tone:\n")
		sb.WriteString(t.Template)
		sb.WriteString("\n")
	}

	return sb.String()
}

// GenerateUserPrompt constructs the specific translation task for the user message.
func (t *Translator) GenerateUserPrompt(req TranslateRequest) string {
	var sb strings.Builder

	// 1. Context (Summary and History)
	if req.Context.PreviousSummary != "" || len(req.Context.RecentHistory) > 0 {
		sb.WriteString("### Narrative Context\n")
		if req.Context.PreviousSummary != "" {
			sb.WriteString("Plot summary so far:\n")
			sb.WriteString(req.Context.PreviousSummary)
			sb.WriteString("\n")
		}
		if len(req.Context.RecentHistory) > 0 {
			sb.WriteString("Recent translation history:\n")
			for _, h := range req.Context.RecentHistory {
				fmt.Fprintf(&sb, "> %s\n", h)
			}
		}
		sb.WriteString("\n")
	}

	// 2. The Task
	sb.WriteString("### Translation Task\n")
	fmt.Fprintf(&sb, "Translate the following %s text into %s, maintaining the style and consistency described in the system instructions:\n\n",
		GetLanguageName(t.SourceLang), GetLanguageName(t.TargetLang))
	sb.WriteString(req.SourceText)

	return sb.String()
}
