BINARY  := pdc
PKG     := github.com/rajmohanram/pdc
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(PKG)/cmd.version=$(VERSION) \
	-X $(PKG)/cmd.commit=$(COMMIT) \
	-X $(PKG)/cmd.date=$(DATE)

# Pinned googleapis ref for the bundled google/api protos (see protos/PROTO_VERSION).
GAPIS_REF ?= 1526e545e9d26f23b9c5d0f04af17297def8d045
GAPIS_RAW := https://raw.githubusercontent.com/googleapis/googleapis/$(GAPIS_REF)/google/api

.PHONY: build cross test tidy vet vendor-protos clean

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

cross:
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64 .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe .

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

vendor-protos:
	@mkdir -p protos/google/api
	@for f in http.proto annotations.proto; do \
		echo "fetching google/api/$$f @ $(GAPIS_REF)"; \
		curl -fsSL "$(GAPIS_RAW)/$$f" -o "protos/google/api/$$f"; \
	done
	@echo "googleapis ref: $(GAPIS_REF)" > protos/PROTO_VERSION
	@echo "wrote protos/PROTO_VERSION"

clean:
	rm -rf bin dist
