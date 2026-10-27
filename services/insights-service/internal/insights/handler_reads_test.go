package insights

import (
	"context"
	"testing"

	pb "github.com/darkphotonKN/fireplace/common/api/proto/insights"
	"github.com/google/uuid"
)

// RPC classes under ADR-0019: only an explicit generation may reach the model;
// a read never does (clause 2), and a write changes stored state without one.
const (
	rpcRead     = "read"
	rpcGenerate = "generate"
	rpcWrite    = "write"
)

// rpcClasses classifies every InsightsService RPC. A new RPC must be added
// here, and a new read RPC must also get a case in
// TestHandler_ReadRPCsNeverReachAGenerator.
var rpcClasses = map[string]string{
	"GenerateSuggestion": rpcGenerate,
	"SuggestVideos":      rpcGenerate,
	"PlanMyDay":          rpcGenerate,
	"GetDayPlan":         rpcRead,
	"TickDayPlanEntry":   rpcWrite,
	"UntickDayPlanEntry": rpcWrite,
	"SetItemRevisions":   rpcWrite,
}

// FS-02JBZ R92, A2: the daily brief's RPCs are gone; every RPC left is
// classified, so no read slips in unasserted.
func TestInsightsService_EveryRPCIsClassifiedAndTheBriefIsGone(t *testing.T) {
	served := map[string]bool{}
	for _, m := range pb.InsightsService_ServiceDesc.Methods {
		served[m.MethodName] = true
		if _, ok := rpcClasses[m.MethodName]; !ok {
			t.Errorf("RPC %s is not classified as read, generate or write", m.MethodName)
		}
	}
	for name := range rpcClasses {
		if !served[name] {
			t.Errorf("classified RPC %s is not served", name)
		}
	}
	for _, gone := range []string{"GetDailyBrief", "RefreshDailyBrief"} {
		if served[gone] {
			t.Errorf("RPC %s is still served (FS-02JBZ R92)", gone)
		}
	}
}

// generatorTrap is an InsightsService whose every method is a generation; a
// read that reaches it has called the model.
type generatorTrap struct{ calls int }

func (g *generatorTrap) GenerateSuggestion(context.Context, uuid.UUID, uuid.UUID) (string, error) {
	g.calls++
	return "", nil
}

func (g *generatorTrap) SuggestVideos(context.Context, uuid.UUID, uuid.UUID) ([]Video, error) {
	g.calls++
	return nil, nil
}

// ADR-0019 clause 2, FS-02JBZ A1: a read RPC reaches only the read side of the
// services behind it, never a generator. dayplan.Service.GetDayPlan itself
// never generates (dayplan's TestGetDayPlan_NothingStored_IsNotPlannedAndNeverGenerates).
func TestHandler_ReadRPCsNeverReachAGenerator(t *testing.T) {
	reads := map[string]func(h *Handler) error{
		"GetDayPlan": func(h *Handler) error {
			_, err := h.GetDayPlan(context.Background(), &pb.GetDayPlanRequest{
				PlanId: uuid.NewString(), UserId: uuid.NewString(), Date: "2026-10-25",
			})
			return err
		},
	}
	for name, class := range rpcClasses {
		if class == rpcRead && reads[name] == nil {
			t.Errorf("read RPC %s has no case here", name)
		}
	}
	for name, call := range reads {
		t.Run(name, func(t *testing.T) {
			gens, dayPlans := &generatorTrap{}, &fakeDayPlans{}

			if err := call(NewHandler(gens, dayPlans)); err != nil {
				t.Fatalf("%s: %v", name, err)
			}

			if gens.calls != 0 || dayPlans.drew {
				t.Fatalf("%s reached a generator (insights calls %d, PlanMyDay %v)", name, gens.calls, dayPlans.drew)
			}
		})
	}
}
