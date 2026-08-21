package broker

import "testing"

func TestMessage_WithCausation_RoundTrips(t *testing.T) {
	// A Message built the ordinary way has a nil Headers map — setting causation
	// on it must not panic, which is the whole reason this is a helper and not a
	// map write at every call site.
	m := Message{MessageId: "m1", Body: []byte("x")}

	m = m.WithCausation("parent-event-id")

	if got := CausationIDFrom(m.Headers); got != "parent-event-id" {
		t.Fatalf("want causation id to round-trip, got %q", got)
	}
}

func TestToPublishing_CarriesTraceAcrossTheWire(t *testing.T) {
	m := Message{
		MessageId:     "evt-1",
		ContentType:   "application/protobuf",
		Body:          []byte("payload"),
		DeliveryMode:  Persistent,
		CorrelationId: "chain-1",
	}.WithCausation("parent-1")

	p := toPublishing(m)

	if p.CorrelationId != "chain-1" {
		t.Errorf("correlation id: want chain-1, got %q", p.CorrelationId)
	}
	if got := CausationIDFrom(p.Headers); got != "parent-1" {
		t.Errorf("causation id: want parent-1, got %q", got)
	}
	if p.MessageId != "evt-1" || string(p.Body) != "payload" || p.DeliveryMode != Persistent {
		t.Errorf("the rest of the envelope did not survive: %+v", p)
	}
}

// WithCausation advertises value semantics — value receiver in, value out, and a
// doc comment saying "returns a copy". Headers is a map, so without an explicit
// clone the write lands in the CALLER's map and every message derived from a
// shared base cross-contaminates. The failure is silent and it corrupts the one
// signal you would use to debug it.
func TestMessage_WithCausation_DoesNotMutateTheOriginal(t *testing.T) {
	base := Message{MessageId: "m", Headers: map[string]any{"pre": "existing"}}

	derived := base.WithCausation("child-1")

	if got := CausationIDFrom(base.Headers); got != "" {
		t.Errorf("base was mutated: want no causation id on the original, got %q", got)
	}
	if got := CausationIDFrom(derived.Headers); got != "child-1" {
		t.Errorf("derived lost its causation id: got %q", got)
	}
	if derived.Headers["pre"] != "existing" {
		t.Error("pre-existing headers did not survive the clone")
	}
}
