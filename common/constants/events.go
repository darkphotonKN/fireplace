package commonconstants

/**
NOTE: naming conventions are kept consistent across the codebase.

Simple rule:

Constant      |    Pattern                             |    Example
Exchange         {domain}.events                            auth.events
Routing Key      {resource}.{action}                        user.created
Queue            {service}.{domain}.{resource}.{action}     plan-service.auth.user.deleted

For the scaffolded MVP we use a single per-service "events" queue and split
per routing-key once real consumers come online.
**/

/**
* Exchanges — one per domain.
**/
const (
	AuthEventsExchange         = "auth.events"
	PlanEventsExchange         = "plan.events"
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

	ChecklistItemCreated     = "checklist_item.created"
	ChecklistItemCompleted   = "checklist_item.completed"
	ChecklistItemUncompleted = "checklist_item.uncompleted"
	ChecklistItemDeleted     = "checklist_item.deleted"
)

/**
* Queue names — {service}.events
**/
const (
	AuthServiceEventsQueue         = "auth-service.events"
	PlanServiceEventsQueue         = "plan-service.events"
	ExampleServiceEventsQueue      = "example-service.events"
	OrchestratorServiceEventsQueue = "orchestrator-service.events"
	ApiGatewayEventsQueue          = "api-gateway.events"
)
