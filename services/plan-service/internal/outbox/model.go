package outbox

type CreateOutboxParams struct {
	RoutingKey string `db:"routing_key"`
	Exchange   string `db:"exchange"`
	Payload    []byte `db:"payload"`
}
