BINARY := zone
MODULE := github.com/peasant-labs/zone

.PHONY: build test lint fmt vet clean install

build:
	go build -o bin/$(BINARY) .

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

clean:
	rm -rf bin/ dist/

install:
	go install .
