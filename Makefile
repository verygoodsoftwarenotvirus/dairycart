GOPATH     := $(GOPATH)
GIT_HASH   := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')

all: tests run

clean:
	rm api/v1/example_files/plugins/mock_db.so
	rm api/v1/example_files/plugins/mock_img.so

.PHONY: coverage
coverage: | example-plugins
	if [ -f coverage.out ]; then rm coverage.out; fi
	docker build -t dairycoverage --file dockerfiles/coverage.Dockerfile .
	docker run --volume=$(GOPATH)/src/github.com/dairycart/dairycart:/output --rm -t dairycoverage
	go tool cover -html=coverage.out
	if [ -f coverage.out ]; then rm coverage.out; fi


.PHONY: ci-coverage
ci-coverage: | example-plugins
	docker build -t dairycoverage --file dockerfiles/coverage.Dockerfile .
	docker run --volume=$(GOPATH)/src/github.com/dairycart/dairycart:/output --rm -t dairycoverage

.PHONY: unit-tests
unit-tests: | example-plugins
	docker build -t api_test -f dockerfiles/unittest.Dockerfile .
	docker run --name api_test --rm api_test

.PHONY: integration-tests
integration-tests:
	docker-compose --file integration-tests.yml up --build --remove-orphans --force-recreate --abort-on-container-exit

.PHONY: tests
tests:
	make unit-tests integration-tests

.PHONY: vendor
vendor:
	dep ensure -update -v

.PHONY: revendor
revendor:
	rm -rf vendor
	rm Gopkg.*
	dep init -v

.PHONY: example-plugins
example-plugins:
	make api/v1/example_files/plugins/mock_db.so api/v1/example_files/plugins/mock_img.so

api/v1/example_files/plugins/mock_db.so api/v1/example_files/plugins/mock_img.so:
	docker build -t plugins --file dockerfiles/example_plugins.Dockerfile .
	docker run --volume=$(GOPATH)/src/github.com/dairycart/dairycart/api/v1/example_files/plugins:/output --rm -t plugins

.PHONY: storage
storage:
	(cd storage/database && gnorm gen)

.PHONY: run
run:
	docker-compose --file docker-compose.yml up --build --remove-orphans --force-recreate --abort-on-container-exit
