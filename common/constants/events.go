package commonconstants

/**
NOTE: naming conventions are kept consistent across the codebase.

Simple rule:

Constant      |    Pattern                             |    Example
Exchange         {domain}.events                            auth.events
Routing Key      {resource}.{action}                        user.created
Queue names —  {consumer-service}.{routing-key}
												owner       what it consumes
e.g. : 	InsightsPlanEventsQueue        = "insights-service.plan.created"
Reads as insights queue for plan.created event.

For the scaffolded MVP we use a single per-service "events" queue and split
per routing-key once real consumers come online.
**/

/**
* Exchanges — one per domain.
**/
const (
	AuthEventsExchange         = "auth.events"
	PlanEventsExchange         = "plan.events"
	InsightsEventsExchange     = "insights.events"
	ExampleEventsExchange      = "example.events"
	OrchestratorEventsExchange = "orchestrator.events"

	DlxEventsExchange = "dlx.exchange"
	RetryExchange     = "retry.exchange"
)

/**
* Routing keys, also acting as event names.
* {resource}.{action}
**/
const (
	// auth events (published by auth-service)
	UserCreated = "user.created"
	UserUpdated = "user.updated"
	UserDeleted = "user.deleted"

	// plan events (published by plan-service)
	PlanCreated = "plan.created"
	PlanUpdated = "plan.updated"
	PlanDeleted = "plan.deleted"

	// PlanItemsRequested is the SOLE trigger for initial-item generation —
	// emitted both at plan creation and on user-initiated retry (FS-0006 R15).
	// insights-service binds this and nothing else; PlanCreated stays a
	// general-purpose lifecycle fact, so a future subscriber is not told a plan
	// was created every time someone retries.
	PlanItemsRequested = "plan.items_requested"

	// insight events (published by insights-service). Both are FACTS, named for
	// what happened and never for what should happen next (ADR-0008) — which is
	// what lets a notification or embedding worker subscribe later without
	// insights-service changing at all.
	InsightGenerated        = "insight.generated"
	InsightGenerationFailed = "insight.generation_failed"

	ChecklistItemCreated     = "checklist_item.created"
	ChecklistItemCompleted   = "checklist_item.completed"
	ChecklistItemUncompleted = "checklist_item.uncompleted"
	ChecklistItemDeleted     = "checklist_item.deleted"
)

/*
*
* Queue names —  {consumer-service}.{routing-key}
*												owner       what it consumes
* e.g. : 	InsightsPlanEventsQueue        = "insights-service.plan.created"
* Reads as insights queue for plan.created event.
 */
const (
	AuthServiceEventsQueue         = "auth-service.events"
	PlanServiceEventsQueue         = "plan-service.events"
	ExampleServiceEventsQueue      = "example-service.events"
	OrchestratorServiceEventsQueue = "orchestrator-service.events"
	ApiGatewayEventsQueue          = "api-gateway.events"
	InsightsPlanEventsQueue        = "insights-service.plan.created"

	// InsightsPlanItemsRequestedQueue replaces InsightsPlanEventsQueue as the
	// generation trigger's queue: insights binds plan.items_requested, not
	// plan.created (FS-0006 R15).
	InsightsPlanItemsRequestedQueue = "insights-service.plan.items_requested"

	// PlanServiceInsightEventsQueue is plan-service's queue for the return hop —
	// it consumes insight.generated and materializes the items.
	PlanServiceInsightEventsQueue = "plan-service.insight.generated"
)
