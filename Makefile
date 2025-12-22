TEST_OPTS=--race

.PHONY: test
test:
	go test $(TEST_OPTS) ./...

.PHONY: stress-test
stress-test:
	TEST_CONCURRENT_WORKERS=50 TEST_CONCURRENT_ITERATIONS=50 go test $(TEST_OPTS) --count=50 ./...

.PHONY: stress-reconnect
stress-recconect:
	TEST_MAX_CONCURRENT_CONNECTIONS=50 TEST_CONCURRENT_WORKERS=250 TEST_CONCURRENT_ITERATIONS=100 go test --race --count 5 --run ClientSuite/TestClientWithErrors ./gonet

.PHONY: fmt
fmt:
	# Fixup modules
	go mod tidy
	# Format the Go sources:
	go fmt ./...

.PHONY: lint
lint:
	# Lint the Go source:
	go vet ./...

.PHONY: pr
pr: stress-test lint fmt
