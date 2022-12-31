.PHONY: test coverage lint vet

build:
	go build
lint:
	go fmt $(go list ./... | grep -v /vendor/)
vet:
	go vet $(go list ./... | grep -v /vendor/)
test:
	go test -v -cover ./...
coverage:
	go test -v -cover -coverprofile=coverage.out ./... &&\
	go tool cover -html=coverage.out -o coverage.html
