.PHONY: test fmt run

test:
	go test ./... -race

fmt:
	gofmt -w .

run:
	go run cmd/bot/main.go
