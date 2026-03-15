.PHONY: test bench vet lint fmt check cover

test:
	go test -race -count=1 ./...

bench:
	go test -bench=. -benchmem -count=3 -run=^$$ ./...

vet:
	go vet ./...

lint:
	@which golangci-lint > /dev/null 2>&1 || { echo "Install: https://golangci-lint.run/welcome/install/"; exit 1; }
	golangci-lint run ./...

fmt:
	gofmt -s -w .

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

check: fmt vet test
