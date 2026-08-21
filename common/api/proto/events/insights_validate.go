package events

import "fmt"

// Validate checks an InsightGeneratedEvent's item graph before anything is
// written. It lives here, beside the generated code rather than inside either
// service, so the producer and the consumer cannot disagree about what a legal
// payload is.
//
// Generated .pb.go files are never hand-edited; this is a separate file in the
// same package for exactly that reason.
func (e *InsightGeneratedEvent) Validate() error {
	for i, it := range e.Items {
		if it.ParentIndex == nil {
			continue
		}
		p := *it.ParentIndex
		if p < 0 || int(p) >= len(e.Items) {
			return fmt.Errorf("events: item %d has parent_index %d, out of range for %d items", i, p, len(e.Items))
		}
		if e.Items[p].ParentIndex != nil {
			return fmt.Errorf("events: item %d nests under item %d, which is itself nested: two-tier maximum", i, p)
		}
	}
	return nil
}
