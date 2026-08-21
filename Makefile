.PHONY: build run test tidy lint clean

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server configs/config.yaml

test:
	go test ./...

tidy:
	go mod tidy

lint:
	go vet ./...

clean:
	rm -rf bin
