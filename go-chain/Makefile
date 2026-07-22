BINARY=tender-node

.PHONY: build run test docker

build:
	go build -o $(BINARY) .

run:
	go run .

test:
	go test ./... -count=1

docker:
	docker compose up --build
