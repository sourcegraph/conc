.DEFAULT_GOAL = test

build:
	go build -v ./...
.PHONY: build

lint:
	golangci-lint run ./...
.PHONY: lint

test:
	go test -race -v ./...
.PHONY: test
