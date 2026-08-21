package broker

// Chain tracing across services (FS-0006 R24).
//
// Two ids, two different homes, for one reason: AMQP already has a native
// correlation-id property and no native causation-id.
//
//   - correlation_id — the WHOLE chain. Carried in Message.CorrelationId, the
//     native AMQP property, because that is what it is for.
//   - causation_id — the IMMEDIATE parent event. Carried in a header, because
//     AMQP has nowhere else to put it.
//
// Both are ENVELOPE concerns, not domain data, which is why they live here
// rather than as two more fields on every event proto.
//
// The helpers exist to stop the key drifting. Nothing prevents one service
// writing "causation_id" and another reading "causationId"; the resulting break
// is silent, because a missing header reads as an empty string rather than an
// error. One constant, one setter, one getter.
const CausationIDHeader = "x-causation-id"

// WithCausation returns a copy of the message carrying the id of the event that
// caused it. Safe on a zero-value Message: the Headers map is created on demand,
// so callers never have to remember to initialise it.
//
// The map is CLONED, not shared. A value receiver copies the struct but not the
// map it points at, so writing straight through would mutate the caller's
// headers — and every message derived from a common base would overwrite each
// other's causation id. That failure is silent, and what it corrupts is the
// trace you would use to find it.
func (m Message) WithCausation(eventID string) Message {
	headers := make(map[string]any, len(m.Headers)+1)
	for k, v := range m.Headers {
		headers[k] = v
	}
	headers[CausationIDHeader] = eventID
	m.Headers = headers
	return m
}

// CausationIDFrom reads the causation id out of a delivery's headers, returning
// "" when absent or when it is not a string. Absence is not an error — an event
// published before this convention existed, or by a producer that predates it,
// is still a valid event.
func CausationIDFrom(headers map[string]any) string {
	if headers == nil {
		return ""
	}
	id, _ := headers[CausationIDHeader].(string)
	return id
}
