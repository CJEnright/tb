BINARY_NAME=tb

build:
	go build -v -o $(BINARY_NAME) ./cmd/tb

install:
	go install
