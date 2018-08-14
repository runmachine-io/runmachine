PROTO_DIR := $(shell pwd)/proto
PROTO_DEFS_DIR := $(shell pwd)/proto/defs
GO_BUILD_CMD := CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo
GO_BIN_DIR := $(GOPATH)/bin
GO_PROTOC_BIN := $(GO_BIN_DIR)/protoc-gen-go
DEP_BIN := $(shell which dep)

$(GO_PROTOC_BIN):
	@go get -u github.com/golang/protobuf/protoc-gen-go

# Generates protobuffer code
generated: $(GO_PROTOC_BIN)
	@echo -n "Generating protobuffer code from proto definitions ... "
	@protoc -I $(PROTO_DEFS_DIR) \
	       $(PROTO_DEFS_DIR)/*.proto \
	       --go_out=plugins=grpc:$(PROTO_DIR) && echo "ok."
