# do not compile with cgo, since version mismatches cause trouble
export CGO_ENABLED=0

dev:
	cd cmd/always-cache; gow -s -e go,mod,yml -w ../.. run . -db memory -origin https://www.acache.io -vv

testw:
	gow -s test ./...

testh: http-cache http-server http-test

http-test:
	sleep 5
	cd http-tests; ./http-test.sh $(id)

http-server:
	cd http-tests/cache-tests; npm run server

http-cache:
	cd cmd/always-cache; gow -w ../.. run . -db memory -legacy -origin http://localhost:8000 -vv

test: test-unit test-http

test-unit:
	go test ./...

test-http:
	cd cmd/always-cache; go run . -origin http://localhost:8000 -legacy -db memory &
	cd http-tests/cache-tests; npm run server &
	sleep 2
	# do not use deno runner, it malfunctions when running the whole suite
	# cd http-tests; rm results-temp.json; deno run -A cli.ts results-temp-deno.json
	cd http-tests/cache-tests; ./test-host.sh localhost:8080 > ../results-temp.json
	killall always-cache
	killall node
	cd http-tests; deno run -A results.ts results-temp.json results.json
	cp http-tests/results-temp.json http-tests/results.json

.PHONY: dev testw test test-unit test-http