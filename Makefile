TESTS ?= ./...

bin     	  := payment-reconciliation-consumer
commit        := $(shell git rev-parse --short HEAD)
tag           := $(shell git tag -l 'v*-rc*' --points-at HEAD)
version       := $(shell if [[ -n "$(tag)" ]]; then echo $(tag) | sed 's/^v//'; else echo $(commit); fi)
lint_output   := lint.txt

.EXPORT_ALL_VARIABLES:
GO111MODULE = on

.PHONY: all
all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: build
build: fmt
	CGO_ENABLED=0 go build

.PHONY: test
test: test-unit

.PHONY: test-unit
test-unit:
	go test $(TESTS)

.PHONY: clean
clean:
	rm -f ./$(bin) ./$(bin)-*.zip build.log

.PHONY: package
package:
ifndef version
	echo "no version defined, using default of unversioned"
	$(eval version:=unversioned)
endif
	$(eval tmpdir:=$(shell mktemp -d build-XXXXXXXXXX))
	cp ./$(bin) $(tmpdir)/$(bin)
	cp ./start.sh $(tmpdir)/start.sh
	cp -r ./assets  $(tmpdir)/assets
	cd $(tmpdir) && zip -r ../$(bin)-$(version).zip $(bin) start.sh assets
	rm -rf $(tmpdir)

.PHONY: dist
dist: clean build package

.PHONY: lint
lint: GO111MODULE=off
lint:
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install
	gometalinter ./... > $(lint_output); true