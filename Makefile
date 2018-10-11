PROTO := proto
VENDOR := vendor
VERSION := $(shell git describe --tags --always --dirty)
PROTO_DIR := $(shell pwd)/$(PROTO)
PROTO_DEFS_DIR := $(shell pwd)/proto/defs
GO_BIN_DIR := $(GOPATH)/bin
GO_PROTOC_BIN := $(GO_BIN_DIR)/protoc-gen-go
PKGS := $(shell go list ./... | grep -v /$(VENDOR)/ | grep -v /$(PROTO)/)
SRC = $(shell find . -type f -name '*.go' -not -path "*/$(VENDOR)/*" -not -path "*/$(PROTO_DIR)/*")

.PHONY: test
test: generated fmtcheck vet
	go test $(PKGS)

$(GO_PROTOC_BIN):
	@go get -u github.com/golang/protobuf/protoc-gen-go

.PHONY: generated
# Generates protobuffer code
generated: $(GO_PROTOC_BIN)
	@echo -n "Generating protobuffer code from proto definitions ... "
	@protoc -I $(PROTO_DEFS_DIR) \
	       $(PROTO_DEFS_DIR)/*.proto \
	       --go_out=plugins=grpc:$(PROTO_DIR) && echo "ok."

$(GOMETALINTER):
	go get -u github.com/alecthomas/gometalinter
	$(GOMETALINTER) --install &> /dev/null

.PHONY: lint
lint: $(GOMETALINTER)
	$(GOMETALINTER) ./... --vendor

.PHONY: fmt
fmt:
	@echo "Running gofmt on all sources ..."
	@gofmt -s -l -w $(SRC)

.PHONY: fmtcheck
fmtcheck:
	@bash -c "diff -u <(echo -n) <(gofmt -d $(SRC))"

.PHONY: vet
vet:
	go vet $(PKGS)

.PHONY: cover
cover:
	$(shell [ -e coverage.out ] && rm coverage.out)
	@echo "mode: count" > coverage-all.out
	$(foreach pkg,$(PKGS),\
		go test -coverprofile=coverage.out -covermode=count $(pkg);\
		tail -n +2 coverage.out >> coverage-all.out;)
	go tool cover -html=coverage-all.out -o=coverage-all.html

build: test
	@echo "building all binaries as Docker images ..."
	docker build -t runm-metadata:$(VERSION) . -f cmd/metadata/Dockerfile
