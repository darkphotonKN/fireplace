package plangw

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

// schemaFor registers a body type and returns the generated schema for one of
// its properties.
//
// It asserts on the DOCUMENT huma produces, not on the Go struct, because the
// document is what clients are generated from. A three-state type can look
// perfectly reasonable in Go and still publish something no client can send.
func schemaFor(t *testing.T, register func(huma.API), schemaName, property string) *huma.Schema {
	t.Helper()
	_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
	register(api)
	s, ok := api.OpenAPI().Components.Schemas.Map()[schemaName]
	if !ok {
		t.Fatalf("schema %q was not generated; got %v", schemaName, keysOfSchemas(api))
	}
	prop, ok := s.Properties[property]
	if !ok {
		t.Fatalf("schema %q has no property %q", schemaName, property)
	}
	return prop
}

func keysOfSchemas(api huma.API) []string {
	var out []string
	for k := range api.OpenAPI().Components.Schemas.Map() {
		out = append(out, k)
	}
	return out
}

func registerUpdateChecklist(api huma.API) {
	type in struct{ Body UpdateChecklistReq }
	huma.Register(api, huma.Operation{
		OperationID: "t", Method: http.MethodPatch, Path: "/t",
		Summary: "t", Description: "t",
	}, func(ctx context.Context, i *in) (*struct{}, error) { return nil, nil })
}

func registerUpdateDates(api huma.API) {
	type in struct{ Body UpdateDatesReq }
	huma.Register(api, huma.Operation{
		OperationID: "t", Method: http.MethodPatch, Path: "/t",
		Summary: "t", Description: "t",
	}, func(ctx context.Context, i *in) (*struct{}, error) { return nil, nil })
}

// OptUUID is three-state in Go — {Present, Valid, Value} — but that is an
// IMPLEMENTATION detail. On the wire it has always been a uuid string or null.
//
// Without a schema hook huma derives the schema from the struct SHAPE and
// publishes an object requiring Present/Valid/Value, so a client generated from
// the document would send {"parentId":{"Present":true,...}} and every real
// request would fail strict validation. Registration also panics outright on
// the `example` tag, because huma parses example as JSON for non-string kinds.
func TestOptUUID_PublishesAsNullableUUIDString(t *testing.T) {
	prop := schemaFor(t, registerUpdateChecklist, "UpdateChecklistReq", "parentId")

	if prop.Type == "object" || len(prop.Properties) > 0 {
		t.Fatalf("parentId published as an object %+v; the wire shape is a uuid string or null", prop.Properties)
	}
	if prop.Type != "string" {
		t.Errorf("parentId type = %q, want %q", prop.Type, "string")
	}
	if prop.Format != "uuid" {
		t.Errorf("parentId format = %q, want %q", prop.Format, "uuid")
	}
	if !prop.Nullable {
		t.Error("parentId must be nullable: null is how a client CLEARS the parent")
	}
}

func TestOptDate_PublishesAsNullableDateString(t *testing.T) {
	for _, field := range []string{"startDate", "dueDate"} {
		t.Run(field, func(t *testing.T) {
			prop := schemaFor(t, registerUpdateDates, "UpdateDatesReq", field)

			if prop.Type == "object" || len(prop.Properties) > 0 {
				t.Fatalf("%s published as an object %+v; the wire shape is a date string or null", field, prop.Properties)
			}
			if prop.Type != "string" {
				t.Errorf("%s type = %q, want %q", field, prop.Type, "string")
			}
			if prop.Format != "date" {
				t.Errorf("%s format = %q, want %q", field, prop.Format, "date")
			}
			if !prop.Nullable {
				t.Error("must be nullable: null is how a client CLEARS the date")
			}
		})
	}
}

// The three-state contract is omit / null / value. A field listed as REQUIRED
// makes "omit" illegal, which would turn every partial update into a request
// that must restate fields it does not intend to change.
func TestThreeStateFields_AreNeverRequired(t *testing.T) {
	cases := []struct {
		name       string
		register   func(huma.API)
		schemaName string
		fields     []string
	}{
		{"UpdateChecklistReq", registerUpdateChecklist, "UpdateChecklistReq", []string{"parentId"}},
		{"UpdateDatesReq", registerUpdateDates, "UpdateDatesReq", []string{"startDate", "dueDate"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, api := humatest.New(t, huma.DefaultConfig("test", "1"))
			tc.register(api)
			s := api.OpenAPI().Components.Schemas.Map()[tc.schemaName]
			if s == nil {
				t.Fatalf("schema %q not generated", tc.schemaName)
			}
			for _, f := range tc.fields {
				for _, req := range s.Required {
					if req == f {
						t.Errorf("%s.%s is required; omitting it must remain legal on a partial update", tc.schemaName, f)
					}
				}
			}
		})
	}
}
