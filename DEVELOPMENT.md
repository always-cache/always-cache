# Developemnt

## HTTP testing

Makefile targets exist for testing for HTTP caching conformance. The tests are run using `http-tests/cache-tests`, and the logic exists in the `http-tests` directory. Here are the helper scripts (written for Deno) as well as a git submodule of the test suite.

### Development against individual tests

```
make -j testh id=[TEST-ID]
```

This starts the testing server (and always-cache, of course) and re-runs the test on file modification.

### Test results page

```
make http-server
```

This starts the testing server. Results are available at http://localhost:8000. The results for always-cached are symlinked from the `release` directory into the test server directory.

## Release

```
make release
```

This runs all tests, builds the binary into the `release` directory, and updates the conformance test results also located in `release`. The release fails on failing unit tests or regressions in HTTP caching tests.
