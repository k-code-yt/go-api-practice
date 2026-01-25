[x] add db -> migrations?
[x] add deb-m configs
[x] consume only payment created && remove events
[x] generic msg for deb -> add switch/case into consumer
[x] send more data via MsgCH -> offset etc
[x] add mapping for kafka msg to domain
[x] udpate inbox logic
[x] add one context for handler and repo
[x] respond back to producer(basic saga??)
[x] add inventory service
[x] add sep-te db for each -> pg && deb configs -> read from .env???
[x] adjust folder structure
[x] kafka commit loop
[x] add DB migrations
[x] refactor handler for PROTO
[] how to manager schema migration with PROTO
[] add AVRO handlers
[] how to manager schema migration with AVRO

[] add GRPC handler && kafka consumer together in the same service -> both will consume PROTO
[] saga -> one compensating aciton???
[] debezium migration -> add like in DB. Store entire JSON with version -> read file -> compare -> update DB
