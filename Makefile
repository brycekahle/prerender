.PHONY: all test build deps image

all: test build

build:
	go build

deps:
	@go get -u github.com/golang/lint/golint
	@go get -u github.com/Masterminds/glide && glide install

image:
	docker build .

lint:
	golint `go list ./... | grep -v /vendor/`

test:
	go test -v `go list ./... | grep -v /vendor/`
