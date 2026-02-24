package translator

import (
	"fmt"
	"strings"

	"github.com/renlulu/hearx/backend/internal/skillpack"
)

func BuildSystemPrompt(merged *skillpack.MergedPack) string {
	var sb strings.Builder

	// Combine all skill pack prompts
	for _, p := range merged.Prompts {
		sb.WriteString(p)
		sb.WriteString("\n\n")
	}

	// Add glossary
	if len(merged.Glossary) > 0 {
		sb.WriteString("## Glossary\n")
		sb.WriteString("Use the following translations consistently:\n\n")
		for en, zh := range merged.Glossary {
			fmt.Fprintf(&sb, "- %s → %s\n", en, zh)
		}
		sb.WriteString("\n")
	}

	// Add entity rules
	if len(merged.Entities) > 0 {
		sb.WriteString("## Entity Rules\n")
		for _, e := range merged.Entities {
			fmt.Fprintf(&sb, "- %s (%s) → %s\n", e.Pattern, e.Category, e.Translation)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
