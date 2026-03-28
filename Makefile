.PHONY: build tidy run-sample test-hier clean

# Binary output directory
BIN := ./bin/logr

# Version (can be overridden: make build VERSION=1.0.0)
VERSION ?= dev

build: tidy
	go build -ldflags "-s -w -X github.com/saurabh/logr/cmd.Version=$(VERSION)" -o $(BIN) .

tidy:
	go mod tidy

# Quick smoke test using the bundled sample log (requires LOGR_DEV=1)
run-sample:
	LOGR_DEV=1 cat testdata/sample.log | go run . --level info

# Test hier path filter
test-hier:
	LOGR_DEV=1 cat testdata/sample.log | go run . --hier "payment.**"

# Test suppression (two identical-pattern lines should produce only one)
test-suppress:
	LOGR_DEV=1 cat testdata/sample.log | go run . --suppress-ttl 1h

# Test service filter
test-service:
	LOGR_DEV=1 cat testdata/sample.log | go run . --service payment-service

# Test profile round-trip
test-profile:
	LOGR_DEV=1 go run . profile save demo --level warn --service payment-service
	LOGR_DEV=1 go run . profile load demo
	LOGR_DEV=1 go run . profile list
	LOGR_DEV=1 go run . profile delete demo

# Test JSON output
test-json:
	LOGR_DEV=1 cat testdata/sample.log | go run . --json --level error

clean:
	rm -rf bin/
