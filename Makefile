GOCMD=go
GOBUILD=$(GOCMD) build
BINARY_NAME=tb

build:
	$(GOBUILD) -o $(BINARY_NAME) -v
