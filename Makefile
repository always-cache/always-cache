# do not compile with cgo, since version mismatches cause trouble
export CGO_ENABLED=0

dev:
	gow -s -e go,mod,yml run .

build:
	GOOS=linux GOARCH=amd64 go build -o acache

testw:
	gow -s test ./...

test:
	go test ./...

testc:
	gow -s -e go,mod,yml run . -config cache-tests.yml -legacy

.PHONY: dev build testw test testc
