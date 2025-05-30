.EXPORT_ALL_VARIABLES:
# Common
BIN          := payment-reconciliation-consumer
SHELL		 :=	/bin/bash
VERSION		 ?= unversioned

# Go
CGO_ENABLED  = 1
XUNIT_OUTPUT = test.xml
LINT_OUTPUT  = lint.txt
TESTS      	 = ./...
COVERAGE_OUT = coverage.out
GO111MODULE  = on

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
ifeq ($(shell uname; uname -p), Darwin arm)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ go build --ldflags '-linkmode external -extldflags "-static"' -o ecs-image-build/app/$(BIN)
else
	go build -o ecs-image-build/app/$(BIN)
endif

.PHONY: test
test: test-unit test-integration

.PHONY: test-unit
test-unit:
	@go test $(TESTS) -run 'Unit'

.PHONY: test-integration
test-integration:
	@go test $(TESTS) -run 'Integration'

.PHONY: test-with-coverage
test-with-coverage:
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

.PHONY: package
package:
ifndef VERSION
	$(error No version given. Aborting)
endif
	$(eval tmpdir := $(shell mktemp -d build-XXXXXXXXXX))
	cp ./ecs-image-build/app/$(BIN) $(tmpdir)/$(BIN)
	cp ./ecs-image-build/docker_start.sh $(tmpdir)/docker_start.sh
	cp ./assets/product_code.yml $(tmpdir)/product_code.yml
	cp ./assets/product_code.yml ecs-image-build/product_code.yml
	cd $(tmpdir) && zip ../$(BIN)-$(VERSION).zip $(BIN) docker_start.sh product_code.yml
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
	@go get github.com/sonatype-nexus-community/nancy@latest
	@LOW=0 MED=0 HIGH=0 CRIT=0 res=$(go list -json -deps ./... | nancy sleuth -o json | jq -c '.vulnerable[].Vulnerabilities[].CvssScore'); for score in $$res; do if [ $${score:1:1} -ge 9 ]; then CRIT=$$(($$CRIT+1)); elif [ $${score:1:1} -ge 7 ]; then HIGH=$$(($$HIGH+1)); elif [ $${score:1:1} -ge 4 ]; then MED=$$(($$MED+1)); else LOW=$$(($$LOW+1)); fi; done; echo "CRITICAL=$$CRIT High=$$HIGH Medium=$$MED Low=$$LOW";
	@go build -o ${GOBIN} golang.org/x/vuln/cmd/govulncheck
	@govulncheck ./...

.PHONY: docker-image
docker-image: dist
	chmod +x build-docker-local.sh
	./build-docker-local.sh