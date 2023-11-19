app = ps-http-sim
gomod := github.com/mattrobenolt/$(app)

PSDB_PROTO_OUT := types
PSDB_PROTO_ROOT := $(PSDB_PROTO_OUT)/psdb
PSDB_V1ALPHA1 := $(PSDB_PROTO_ROOT)/v1alpha1

BIN := bin

UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)

all: $(BIN)/$(app)

proto: \
	$(PSDB_V1ALPHA1)/database.pb.go

clean: clean-bin clean-dist

clean-proto:
	rm -rf $(PSDB_PROTO_OUT)

clean-bin:
	rm -rf $(BIN)

clean-dist:
	rm -rf dist

$(BIN):
	mkdir -p $(BIN)

$(PSDB_PROTO_OUT):
	mkdir -p $(PSDB_PROTO_OUT)

GO_INSTALL := env GOBIN=$(PWD)/$(BIN) go install -ldflags "-s -w" -trimpath

$(BIN)/buf: Makefile | $(BIN)
	$(GO_INSTALL) github.com/bufbuild/buf/cmd/buf@v1.28.0

$(BIN)/protoc-gen-go: Makefile | $(BIN)
	$(GO_INSTALL) google.golang.org/protobuf/cmd/protoc-gen-go

$(BIN)/protoc-gen-connect-go: Makefile | $(BIN)
	$(GO_INSTALL) connectrpc.com/connect/cmd/protoc-gen-connect-go

$(BIN)/protoc-gen-go-vtproto: Makefile | $(BIN)
	$(GO_INSTALL) github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v0.5.0

$(BIN)/goreleaser: Makefile | $(BIN)
	$(GO_INSTALL) github.com/goreleaser/goreleaser@v1.22.1

PROTO_TOOLS := $(BIN)/buf $(BIN)/protoc-gen-go $(BIN)/protoc-gen-connect-go $(BIN)/protoc-gen-go-vtproto
tools: $(PROTO_TOOLS)

$(PSDB_V1ALPHA1)/database.pb.go: $(PROTO_TOOLS) proto-src/planetscale/psdb/v1alpha1/database.proto | $(PSDB_PROTO_OUT)
	$(BIN)/buf generate -v proto-src/planetscale/psdb/v1alpha1/database.proto

$(BIN)/$(app): main.go go.mod go.sum | $(BIN)
	$(GO_INSTALL) $(gomod)

run: $(BIN)/$(app)
	$< \
		-listen-addr=127.0.0.1 \
		-listen-port=8080 \
		-mysql-addr=127.0.0.1 \
		-mysql-port=3306 \
		-mysql-idle-timeout=5s \
		-mysql-no-pass \
		-mysql-max-rows=1000 \
		-mysql-dbname=mysql

docker:
	docker buildx build --target=local --rm -t $(app) .

run-mysql:
	docker run -it --rm --name $(app)-mysqld -e MYSQL_ALLOW_EMPTY_PASSWORD="true" -e MYSQL_ROOT_PASSWORD="" -p 127.0.0.1:3306:3306 mysql:8.0.29

publish: clean $(BIN)/goreleaser
	$(BIN)/goreleaser release

.PHONY: all proto clean clean-proto clean-bin clean-dist tools run run-mysql publish
