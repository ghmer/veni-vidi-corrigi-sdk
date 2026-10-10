.PHONY: test lint fmt tidy

test:
	go test -timeout=180s ./...

lint:
	golangci-lint run

fmt:
	gofmt -l -w .

tidy:
	go mod tidy
