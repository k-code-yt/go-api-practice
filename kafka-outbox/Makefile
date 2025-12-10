build-server:
	@go build -o ./bin/server ./cmd/server/.
	@chmod +x ./bin/server

kafka-app-chans: build-server
	@CG_ID=$(CG_ID) SHOULD_PRODUCE=$(SHOULD_PRODUCE) ./bin/server/server/.

build-server-race:
	@go build -race -o ./bin/server ./cmd/server/.
	@chmod +x ./bin/server

kafka-c-chans-1: build-server-race
	@CG_ID=consumer-group-1 SHOULD_PRODUCE=false ./bin/server

kafka-p-chans: build-server
	@CG_ID=producer-group SHOULD_PRODUCE=true ./bin/server
