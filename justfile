generate:
    go generate ./...
run:
    river -c "go run ./..."
test:
    go test ./...
lint:
    golangci-lint run
format:
    gofumpt -w .
