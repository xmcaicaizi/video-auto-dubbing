package translate

import (
	"context"
)

// Translator is the interface for translation providers.
type Translator interface {
	// Translate translates a batch of texts from source language to target language.
	// Returns translated texts in the same order as input.
	Translate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error)
}
