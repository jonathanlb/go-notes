build: build_index build_server

build_index:
	go build -o bin/index cmd/index/main.go

build_server:
	CGO_CFLAGS="-DSQLITE_ENABLE_RTREE -DSQLITE_THREADSAFE=1" go build -o bin/server cmd/server/main.go

test:
	go test -v ./...

test_index:
	go test ./pkg/index

test_notes:
	go test ./pkg/notes

cover:
	rm -rf coverage
	mkdir coverage
	go list -f '{{if gt (len .TestGoFiles) 0}}"go test -covermode count -coverprofile {{.Name}}.coverprofile -coverpkg ./... {{.ImportPath}}"{{end}}' ./... | xargs -I {} bash -c {}
	echo "mode: count" > coverage/cover.out
	grep -h -v "^mode:" *.coverprofile >> "coverage/cover.out"
	rm *.coverprofile
	go tool cover -html=coverage/cover.out -o=coverage/cover.html

lint:
	go fmt ./pkg/*

clean:
	rm -rf coverage
	rm bin/server bin/index
