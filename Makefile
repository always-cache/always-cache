# do not compile with cgo, since version mismatches cause trouble
export CGO_ENABLED=0
provider ?= memory

dev:
	gow -s -e go,mod,yml run . -provider memory -config config.yml -origin https://acache.statichost.eu -vv

testw:
	gow -s test ./...

testh: http-cache http-server http-test

http-test:
	sleep 5
	cd http-tests; ./http-test.sh $(id)

http-server:
	cd http-tests/cache-tests; npm run server

http-cache:
	gow run . -provider memory -legacy -origin http://localhost:8000 -vv

release: test build
	cp http-tests/results-temp.json release/results.json
	git add .

build: test
	GOOS=linux GOARCH=amd64 go build -o release/always-cache

test: test-unit test-http

test-unit:
	go test ./...

test-http:
	go run . -origin http://localhost:8000 -legacy -provider $(provider) &
	cd http-tests/cache-tests; npm run server &
	sleep 2
	cd http-tests; deno run -A cli.ts results-temp.json
	rm -f cache.db*
	killall always-cache
	killall node
	cd http-tests; deno run -A results.ts results-temp.json ../release/results.json

.PHONY: dev build testw test test-unit test-http release