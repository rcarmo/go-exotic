TMPDIR ?= /workspace/tmp
GOTMPDIR ?= /workspace/tmp
export TMPDIR GOTMPDIR

.PHONY: all fmt test vet check run clean

all: check

fmt:
	gofmt -w ./cmd ./internal

test:
	go test ./...

vet:
	go vet ./...

check: fmt test vet
	git diff --check

run:
	go run ./cmd/go-exotic -layers $${LAYERS:-4}

clean:
	rm -f go-exotic
