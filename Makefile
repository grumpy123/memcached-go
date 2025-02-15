TEST_OPTS=--race

.PHONY: test
test:
	go test $(TEST_OPTS) ./...

.PHONY: stress-test
stress-test:
	TEST_CONCURRENT_WORKERS=50 TEST_CONCURRENT_ITERATIONS=50 go test $(TEST_OPTS) --count=50 ./...

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
