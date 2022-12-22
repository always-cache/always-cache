# do not compile with cgo, since version mismatches cause trouble
export CGO_ENABLED=0

dev:
	gow -s -e go,mod,yml run . -provider memory -config config.yml -origin https://www.arktis3d.com -vv

testw:
	gow -s test ./...

testh: http-cache http-server http-test

http-test:
	cd cache-tests-runner; ./http-test.sh $(id)

http-server:
	cd cache-tests-runner; npm run server

http-cache:
	gow -s run . -provider memory -legacy -origin http://localhost:8000 -vv

release: repo-is-clean test build
	cp cache-tests-runner/results-temp.json release/results.json
	git add .

build: test
	GOOS=linux GOARCH=amd64 go build -o release/always-cache

test: test-unit test-http

test-unit:
	go test ./...

test-http:
	go run . -origin http://localhost:8000 -legacy -provider memory &
	cd cache-tests-runner; npm run server &
	sleep 2
	cd cache-tests-runner; deno run -A cli.ts results-temp.json
	killall always-cache
	killall node
	cd cache-tests-runner; deno run -A results.ts results-temp.json ../release/results.json

repo-is-clean:
	git diff --exit-code

.PHONY: dev build testw test test-unit test-http release repo-is-clean
