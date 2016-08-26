all: fmt http-auth

fmt:
	go fmt ./...

http-auth:
	go build -o ./bin/http-auth ./cmd/http-auth
