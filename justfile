generate:
    mkdir -p generated/protocol
    go generate ./...
run:
    river -c "go run ./..."
