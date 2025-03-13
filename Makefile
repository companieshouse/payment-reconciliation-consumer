.EXPORT_ALL_VARIABLES:
# Common
BIN          = payment-reconciliation-consumer
VERSION		 = unversioned
# Go
chs_envs      := $(CHS_ENV_HOME)/global_env $(CHS_ENV_HOME)/payment-reconciliation-consumer/env
source_env    := for chs_env in $(chs_envs); do test -f $$chs_env && . $$chs_env; done
CHS_ENV_HOME  ?= $(HOME)/.chs_env
CGO_ENABLED  = 0
XUNIT_OUTPUT = test.xml
LINT_OUTPUT  = lint.txt
TESTS      	 = ./...
COVERAGE_OUT = coverage.out
GO111MODULE  = on
ifeq ($(shell uname; uname -p)$(CGO_ENABLED), Darwin arm1)
	GOOS=linux
	GOARCH=amd64
	CC=x86_64-linux-musl-gcc
	CXX=x86_64-linux-musl-g++
else ifeq ($(shell uname; uname -p)$(CGO_ENABLED), Darwin arm0)
	GOOS=darwin
	GOARCH=arm64
else
	GOOS=linux
	GOARCH=amd64
endif

.PHONY:
arch:
	@echo OS: $(GOOS) ARCH: $(GOARCH)

.PHONY: all
all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: build
build: arch fmt
ifeq ($(shell uname; uname -p)$(CGO_ENABLED), Darwin arm1)
	go build --ldflags '-linkmode external -extldflags "-static"'
else
	go build -v
endif

.PHONY: test
test: test-unit test-integration

.PHONY: test-unit
test-unit:
	@go test $(TESTS) -run 'Unit'

.PHONY: test-integration
test-integration:
	@$(source_env); go test $(TESTS) -run 'Integration'

.PHONY: test-with-coverage
test-with-coverage:
ifeq ($(shell uname; uname -p), Darwin arm)
	@make clean
	@$(eval GOOS=darwin)
	@$(eval GOARCH=arm64)
	@go build
endif
	@go get github.com/hexira/go-ignore-cov
	@go build -o ${GOBIN} github.com/hexira/go-ignore-cov
	@go test -coverpkg=./... -coverprofile=$(COVERAGE_OUT) $(TESTS)
	@go-ignore-cov --file $(COVERAGE_OUT)
	@go tool cover -func $(COVERAGE_OUT)
	@make coverage-html

.PHONY: clean-coverage
clean-coverage:
	@rm -f $(COVERAGE_OUT) coverage.html

.PHONY: coverage-html
coverage-html: 
	@go tool cover -html=$(COVERAGE_OUT) -o coverage.html

.PHONY: clean
clean: clean-coverage
	go mod tidy
	rm -f ./$(BIN) ./$(BIN)-*.zip

.PHONY: package
package:
ifndef VERSION
	$(error No version given. Aborting)
endif
	$(eval tmpdir := $(shell mktemp -d build-XXXXXXXXXX))
	cp ./$(BIN) $(tmpdir)/$(BIN)
	cp ./start.sh $(tmpdir)/start.sh
	cd $(tmpdir) && zip ../$(BIN)-$(VERSION).zip $(BIN) start.sh
	rm -rf $(tmpdir)

.PHONY: dist
dist: clean build package

.PHONY: lint
lint:
	GO111MODULE=off
	go get -u github.com/lint/golint
	golint ./... > $(LINT_OUTPUT)

.PHONY: security-check
security-check dependency-check:
	@go get golang.org/x/vuln/cmd/govulncheck
	@go build -o ${GOBIN} golang.org/x/vuln/cmd/govulncheck
	@govulncheck ./...