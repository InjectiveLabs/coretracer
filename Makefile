all:

test:
	go test -race -cover -v ./...

testv:
	go test -race -cover -v ./...

bench:
	go test -race -bench=. -run=^$ ./...

coverprofile:
	go test -race -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

lint:
	golangci-lint run
# % golangci-lint --version
# golangci-lint has version 1.63.4 built with go1.23.4 from c1149695 on 2025-01-03T19:49:42Z

.PHONY: test lint


