.PHONY: build test vet clean

BINARY=forumgo

build:
	go build -o $(BINARY) ./cmd/forumgo

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
