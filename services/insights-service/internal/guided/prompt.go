package guided

import "strings"

// Snippet block headers of a generation's user message. Prompt prose may say
// "milestone" (R6); no type or field here does.
const (
	workedExamplesHeader = "WORKED EXAMPLES (from a shared playbook of goal breakdowns):"
	personalHeader       = "THINGS THIS USER FINISHED ON OTHER PLANS:"
)

// WriteSnippets writes one labelled block per origin (R4), playbook first, each
// followed by a blank line; every generation prompt that carries snippets uses it.
// Nothing at all is written when there are no snippets (R5).
func WriteSnippets(b *strings.Builder, snippets []Snippet) {
	for _, block := range []struct {
		origin SnippetOrigin
		header string
	}{{OriginPlaybook, workedExamplesHeader}, {OriginPersonal, personalHeader}} {
		wrote := false
		for _, s := range snippets {
			if s.Origin != block.origin {
				continue
			}
			if !wrote {
				b.WriteString(block.header)
				b.WriteByte('\n')
				wrote = true
			}
			b.WriteString("- ")
			b.WriteString(strings.Join(strings.Fields(s.Text), " "))
			b.WriteByte('\n')
		}
		if wrote {
			b.WriteByte('\n')
		}
	}
}
