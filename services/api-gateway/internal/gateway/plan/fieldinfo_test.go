package plangw

import (
	"reflect"
	"strings"
	"testing"
)

// jsonField describes one published field: the name a client sees, and whether
// the json tag carries omitempty.
//
// omitempty is what huma reads to decide REQUIRED vs optional, which makes it
// contract rather than a marshalling nicety. Reading it here lets the tests
// assert on the published shape without booting a huma API.
type jsonField struct {
	name      string
	omitempty bool
}

func jsonFieldsOf(t *testing.T, v any) []jsonField {
	t.Helper()
	rt := reflect.TypeOf(v)
	if rt.Kind() != reflect.Struct {
		t.Fatalf("jsonFieldsOf wants a struct, got %s", rt.Kind())
	}

	var out []jsonField
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		parts := strings.Split(tag, ",")
		name := parts[0]
		if name == "" {
			name = f.Name
		}
		out = append(out, jsonField{
			name:      name,
			omitempty: contains(parts[1:], "omitempty"),
		})
	}
	if len(out) == 0 {
		t.Fatalf("%s published no json fields — the assertion below would pass vacuously", rt.Name())
	}
	return out
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
