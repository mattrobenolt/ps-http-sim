app = ps-http-sim
gomod := github.com/mattrobenolt/$(app)

BIN := bin

src := main.go internal/session/session.go internal/vitess/vitess.go

ifdef DRYRUN
GORELEASERFLAGS += --snapshot
endif

all: $(BIN)/$(app)

clean: clean-bin clean-dist

clean-bin:
	rm -rf $(BIN)

clean-dist:
	rm -rf dist

$(BIN):
	mkdir -p $(BIN)

GO_INSTALL := env GOBIN=$(PWD)/$(BIN) go install -ldflags "-s -w" -trimpath

$(BIN)/goreleaser: Makefile | $(BIN)
	$(GO_INSTALL) github.com/goreleaser/goreleaser@v1.24.0

$(BIN)/$(app): main.go go.mod go.sum $(src) | $(BIN)
	$(GO_INSTALL) $(gomod)

run: $(BIN)/$(app)
	$< \
		-listen-addr=127.0.0.1 \
		-listen-port=8080 \
		-log-level=debug \
		-mysql-addr=127.0.0.1 \
		-mysql-port=3306 \
		-mysql-idle-timeout=5s \
		-mysql-no-pass \
		-mysql-max-rows=1000 \
		-mysql-dbname=mysql \
		-mysql-listen-port=3309

docker:
	docker buildx build --target=local --rm -t $(app) .

run-mysql:
	docker run -it --rm --name $(app)-mysqld -e MYSQL_ALLOW_EMPTY_PASSWORD="true" -e MYSQL_ROOT_PASSWORD="" -p 127.0.0.1:3306:3306 mysql:8.0.38

publish: $(BIN)/goreleaser
	$(BIN)/goreleaser release --clean $(GORELEASERFLAGS)
	$(MAKE) bump-godoc

bump-godoc:
	curl -XPOST https://pkg.go.dev/fetch/$(gomod)@$(shell jq -r .tag dist/metadata.json)

.PHONY: all clean clean-bin clean-dist run docker run-mysql publish bump-godoc
