package guided

import "testing"

func TestNormalizeText_IgnoresCaseAndCollapsesWhitespace(t *testing.T) {
	tests := []struct{ a, b string }{
		{"Write a CLI", "write  a cli"},
		{"  Read the tour\n", "read the TOUR"},
	}
	for _, tt := range tests {
		if NormalizeText(tt.a) != NormalizeText(tt.b) {
			t.Errorf("NormalizeText(%q) = %q, NormalizeText(%q) = %q; want equal", tt.a, NormalizeText(tt.a), tt.b, NormalizeText(tt.b))
		}
	}
	if NormalizeText("Write a CLI") == NormalizeText("Write a CLI tool") {
		t.Error("different texts share a duplicate key")
	}
}
