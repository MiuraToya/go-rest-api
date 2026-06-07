DB_PATH ?= var/app.db

.PHONY: run fmt test test-unit test-integration

run:
	go run ./cmd/api

fmt:
	gofmt -w ./cmd ./internal ./integration

test: test-unit test-integration

test-unit:
	go test ./...

test-integration:
	go test -tags=integration ./...
