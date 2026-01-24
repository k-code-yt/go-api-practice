.PHONY: proto mgr

#### PAYMENT ####
build-payment-server:
	@go build -o ./bin/payment-server ./cmd/payment-server/.
	@chmod +x ./bin/payment-server

build-payment-server-race:
	@go build -race -o ./bin/payment-server ./cmd/payment-server/.
	@chmod +x ./bin/payment-server

payment: build-payment-server-race
	@CG_ID=api-cg ./bin/payment-server

migrate-payment:
	go run migrations/main.go -service payment -action up

#### INVENTORY ####
build-inventory-server:
	@go build -o ./bin/inventory-server ./cmd/inventory-server/.
	@chmod +x ./bin/inventory-server

build-inventory-server-race:
	@go build -race -o ./bin/inventory-server ./cmd/inventory-server/.
	@chmod +x ./bin/inventory-server

inventory: build-inventory-server-race
	@CG_ID=audit-cg ./bin/inventory-server

migrate-inventory:
	go run migrations/main.go -service inventory -action up

#### GLOBAL ####
mgr: migrate-payment migrate-inventory
	
proto:
	protoc --go_out=pkg/types --go_opt=paths=source_relative \
		--proto_path=pkg/types/proto \
		pkg/types/proto/*.proto