package authgw

import "testing"

// FS-12ZMF R15: display name if set, otherwise account name.
func TestUserShownName(t *testing.T) {
	str := func(s string) *string { return &s }
	cases := map[string]struct {
		display *string
		want    string
	}{
		"display name set": {str("Nicky"), "Nicky"},
		"no display name":  {nil, "Nick"},
		"empty display":    {str(""), "Nick"},
		"blank display":    {str("   "), "Nick"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			u := &User{Name: "Nick", DisplayName: tc.display}
			if got := u.ShownName(); got != tc.want {
				t.Errorf("ShownName() = %q, want %q", got, tc.want)
			}
		})
	}
}
