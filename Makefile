BINARY := myconfig
PKG    := ./...

.PHONY: build lint test install

build:
	go build -o $(BINARY) .

lint:
	go vet $(PKG)

test:
	go test $(PKG)

install:
	go install .
