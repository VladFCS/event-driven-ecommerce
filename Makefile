.PHONY: proto

proto:
	PATH="$(PATH):$(HOME)/go/bin" find api/proto -name '*.proto' -print0 | xargs -0 protoc \
		-I api/proto \
		--go_out=. \
		--go_opt=module=github.com/vladfc/event-driven-ecommerce-app \
		--go-grpc_out=. \
		--go-grpc_opt=module=github.com/vladfc/event-driven-ecommerce-app
