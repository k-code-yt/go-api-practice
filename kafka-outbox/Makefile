build-http-server:
	@go build -o ./bin/http-server ./cmd/http-server/.
	@chmod +x ./bin/http-server

build-msg-server:
	@go build -o ./bin/msg-server ./cmd/msg-server/.
	@chmod +x ./bin/msg-server

build-http-server-race:
	@go build -race -o ./bin/http-server ./cmd/http-server/.
	@chmod +x ./bin/http-server

build-msg-server-race:
	@go build -race -o ./bin/msg-server ./cmd/msg-server/.
	@chmod +x ./bin/msg-server

api: build-http-server-race
	@CG_ID=api-cg ./bin/http-server

audit: build-msg-server-race
	@CG_ID=audit-cg ./bin/msg-server
