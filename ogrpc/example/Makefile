.PHONY: tools
tools:
	go get -u github.com/golang/protobuf/protoc-gen-go

.PHONY: zonePB
zonePB:
	protoc \
	  --proto_path=zonePB \
	  --go_out=plugins=grpc:zonePB \
	  --go_opt=paths=source_relative zonePB/zone.proto \
	  zonePB/zone.proto

.PHONY: server
server:
	@ cd server && go run .

.PHONY: client
client:
	@ cd client && go run .
