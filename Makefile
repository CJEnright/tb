BINARY_NAME=tb

build:
	go build -o $(BINARY_NAME) -v

install:
	go install
