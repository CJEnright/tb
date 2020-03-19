BINARY_NAME=tb

build:
	go build -v -o ./bin/$(BINARY_NAME) ./cmd/tb

install:
	go install -i ./cmd/tb
