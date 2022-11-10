# do not compile with cgo, since version mismatches cause trouble
export CGO_ENABLED=0

dev:
	gow -s run .

build:
	GOOS=linux GOARCH=amd64 go build -o acache

testw:
	gow -s test ./...

test:
	go test ./...

.PHONY: dev build testw test
