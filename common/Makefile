.PHONY: gen
gen:
	@protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/proto/auth/auth.proto \
		api/proto/example/example.proto \
		api/proto/plan/plan.proto \
		api/proto/plan/checklist.proto \
		api/proto/events/auth.proto \
		api/proto/events/plan.proto
