all: binaries

binaries:
	env CGO_ENABLED=0 go build \
		-a \
		-trimpath \
		-ldflags "-s -w -extldflags '-static'" \
		-installsuffix cgo \
		-tags netgo \
		-o ./bin/ \
		./cmd/...

test: lint vet
	go test -race -cover ./...

lint:
	# if staticcheck is missing, `go install honnef.co/go/tools/cmd/staticcheck@latest`
	staticcheck ./...

vet:
	go vet ./...

.PHONY: all binaries test lint vet
