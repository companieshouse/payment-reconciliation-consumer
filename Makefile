bin     := payment-reconciliation-consumer

lint_output  := lint.txt

.PHONY: all
all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: deps
deps:
	go get ./...

.PHONY: build
build: deps fmt $(bin)

$(bin):
	CGO_ENABLED=0 go build -o ./$(bin)

.PHONY: test-deps
test-deps: deps
	go get -t ./...

.PHONY: test
test: test-unit

.PHONY: test-unit
test-unit: test-deps
	go test ./... -run 'Unit' -coverprofile=coverage.out

.PHONY: clean
clean:
	go mod tidy
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
lint:
	go get github.com/golang/lint/golint
	golint ./... > $(lint_output)

.EXPORT_ALL_VARIABLES:
    GO111MODULE = on
