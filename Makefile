all: ready testsuite install clean

testsuite: test race benchmark

install: deps
	go install github.com/universonic/cmdb/cmd/...

deps:
	dep ensure -v

test: deps
	go test -cpu 1,4 -timeout 5m github.com/universonic/cmdb/...

race: deps
	go test -race -cpu 1,4 -timeout 7m github.com/universonic/cmdb/...

benchmark: deps
	go test -bench . -cpu 1,4 -timeout 10m github.com/universonic/cmdb/...

build: deps
	go build github.com/universonic/cmdb/cmd/...

clean:
	go clean -i github.com/universonic/cmdb/...

ready:
	go version
	go get -u github.com/golang/dep/cmd/dep

.PHONY: \
	testsuite \
	install \
	deps \
	test \
	race \
	benchmark \
	build \
	clean \