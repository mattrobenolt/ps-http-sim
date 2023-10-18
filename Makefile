gomod := github.com/mattrobenolt/ps-http-sim

PSDB_PROTO_OUT := types
PSDB_PROTO_ROOT := $(PSDB_PROTO_OUT)/psdb
PSDB_V1ALPHA1 := $(PSDB_PROTO_ROOT)/v1alpha1

BIN := bin

UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)

proto: \
	$(PSDB_V1ALPHA1)/database.pb.go

clean: clean-proto clean-bin

clean-proto:
	rm -rf $(PSDB_PROTO_OUT)

clean-bin:
	rm -rf $(BIN)

$(BIN):
	mkdir -p $(BIN)

$(PSDB_PROTO_OUT):
	mkdir -p $(PSDB_PROTO_OUT)

GO_INSTALL := env GOBIN=$(PWD)/$(BIN) go install

$(BIN)/buf: Makefile | $(BIN)
	$(GO_INSTALL) github.com/bufbuild/buf/cmd/buf@v1.27.0

$(BIN)/protoc-gen-go: Makefile | $(BIN)
	$(GO_INSTALL) google.golang.org/protobuf/cmd/protoc-gen-go

$(BIN)/protoc-gen-connect-go: Makefile | $(BIN)
	$(GO_INSTALL) connectrpc.com/connect/cmd/protoc-gen-connect-go

$(BIN)/protoc-gen-go-vtproto: Makefile | $(BIN)
	$(GO_INSTALL) github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v0.5.0

PROTO_TOOLS := $(BIN)/buf $(BIN)/protoc-gen-go $(BIN)/protoc-gen-connect-go $(BIN)/protoc-gen-go-vtproto
tools: $(PROTO_TOOLS)

$(PSDB_V1ALPHA1)/database.pb.go: $(PROTO_TOOLS) proto-src/planetscale/psdb/v1alpha1/database.proto | $(PSDB_PROTO_OUT)
	$(BIN)/buf generate -v proto-src/planetscale/psdb/v1alpha1/database.proto

$(BIN)/ps-http-sim: main.go go.mod go.sum
	GOBIN=$(abspath $(BIN)) go install $(gomod)

run: $(BIN)/ps-http-sim proto
	$(BIN)/ps-http-sim \
		-http-addr=127.0.0.1 \
		-http-port=8080 \
		-mysql-addr=127.0.0.1 \
		-mysql-port=3306 \
		-mysql-idle-timeout=5s \
		-mysql-no-pass \
		-mysql-max-rows=1000 \
		-mysql-dbname=mysql

docker:
	docker build --rm -t ps-http-sim .
