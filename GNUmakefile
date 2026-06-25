default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run
	terraform fmt -check -recursive examples/

generate:
	go tool tfplugindocs generate -provider-name truenas

fmt:
	gofmt -s -w -e .
	terraform fmt -recursive examples/

unit-tests:
	go test -v -coverprofile=coverage.out -covermode=atomic -timeout=120s -parallel=10 ./...

acc-tests:
	TF_ACC=1 go test -v -coverprofile=coverage.out -covermode=atomic -timeout 120m ./...

.PHONY: fmt lint build install generate unit-tests acc-tests
