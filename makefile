build_index:
	go build -o bin/index cmd/index/main.go

build_server:
	go build -o bin/server cmd/server/main.go

test:
	go test -v ./...