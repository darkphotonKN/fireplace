package guided

import (
	"strings"
	"testing"
)

func TestWriteSnippets_PlaybookBlockThenPersonalBlock(t *testing.T) {
	var b strings.Builder

	WriteSnippets(&b, []Snippet{
		{Origin: OriginPersonal, Text: "Set up CI\n  for the blog"},
		{Origin: OriginPlaybook, Text: "Break a launch into deploy, docs, announce"},
	})

	want := workedExamplesHeader + "\n- Break a launch into deploy, docs, announce\n\n" +
		personalHeader + "\n- Set up CI for the blog\n\n"
	if b.String() != want {
		t.Fatalf("got:\n%q\nwant:\n%q", b.String(), want)
	}
}

func TestWriteSnippets_NoSnippetsWritesNothing(t *testing.T) {
	var b strings.Builder

	WriteSnippets(&b, nil)

	if b.Len() != 0 {
		t.Fatalf("wrote %q with no snippets", b.String())
	}
}
