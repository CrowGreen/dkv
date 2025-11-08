.PHONY: proto build run clean certs

proto:
	protoc --go_out=. --go-grpc_out=. proto/kv.proto

build:
	go build -o bin/dkv-server ./cmd/server
	go build -o bin/dkv-client ./cmd/client

run: build
	./bin/dkv-server

clean:
	rm -rf bin/ data/

certs:
	mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -keyout certs/server.key \
		-out certs/server.crt -days 365 -nodes \
		-subj "/CN=localhost"
	openssl req -x509 -newkey rsa:4096 -keyout certs/ca.key \
		-out certs/ca.crt -days 365 -nodes \
		-subj "/CN=dkv-ca"

test:
	go test ./... -v -race
