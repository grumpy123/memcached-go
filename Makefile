TEST_OPTS=--race

.PHONY: test
test:
	go test $(TEST_OPTS) ./...

.PHONY: stress-test
stress-test:
	TEST_CONCURRENT_WORKERS=50 TEST_CONCURRENT_ITERATIONS=50 go test $(TEST_OPTS) --count=50 ./...

.PHONY: stress-reconnect
stress-reconnect:
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

.PHONY: start-mmc
start-mmc:
	docker stop memcached || true
	docker rm memcached || true
	docker run -p 11211:11211 --name memcached --detach memcached:alpine

.PHONY: test-mmc
test-mmc: start-mmc
	MEMCACHED_ADDR=localhost:11211 go test $(TEST_OPTS) ./mmc/...
	MEMCACHED_ADDR=localhost:11211 go test $(TEST_OPTS) .

.PHONY: stress-mmc
stress-mmc: start-mmc
	MEMCACHED_ADDR=localhost:11211 TEST_MAX_CONCURRENT_CONNECTIONS=25 TEST_CONCURRENT_WORKERS=250 TEST_CONCURRENT_ITERATIONS=1000 go test $(TEST_OPTS) --count=10 .

.PHONY: test-all
test-all: test stress-test stress-reconnect test-mmc stress-mmc

.PHONY: bench-mmc
bench: start-mmc
	go test --bench=^Benchmark --benchtime=5s ./bench/...
