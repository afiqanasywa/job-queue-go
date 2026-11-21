# Go parameters
GOCMD=go
BINARY_NAME=jobqueue
BINARY_UNIX=$(BINARY_NAME)_unix

.PHONY: all build run test unittest lint clean build-linux

all: test build

build:
	$(GOCMD) build -o $(BINARY_NAME) -v .

run:
	$(GOCMD) run .

# -race needs cgo (gcc) to be available
test:
	$(GOCMD) test ./... -cover -race -count=1

unittest:
	$(GOCMD) test ./... -short -cover -count=1

lint:
	golangci-lint run ./...

clean:
	$(GOCMD) clean
	rm -f $(BINARY_NAME) $(BINARY_UNIX)

build-linux:
	CGO_ENABLED=0 GOOS=linux $(GOCMD) build -o $(BINARY_UNIX) -a -installsuffix cgo -v .
