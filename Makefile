include .env

.PHONY: vendor test

## help: Prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

## build: builds the binary
build: clean
	CGO_ENABLED=0 GOOS=linux go build --mod vendor -a -installsuffix cgo -ldflags '-s -w -extldflags "-static" -X main.build=${shell git rev-parse --short HEAD}' -o bin/ ./cmd/...

## clean: cleans the binary
clean:
	@go clean

## test: runs go test for all packages
test:
	go test ./pkg/... ./internal/...

## test-cover: runs go test for all packages with coverage
test-cover:
	go test -v -cover ./...

## vendor: setup go modules
vendor:
	go mod tidy && go mod vendor

## lint: lints all Go files
lint:
	docker exec -it ${APP_NAME}-app /go/bin/golangci-lint run cmd/... internal/... pkg/...

## watch: starts watching the code for changes
watch:
	set -o allexport && . .env && set +o allexport && air

## mock: creates mocks for an interface. ex - make mocks pkg/team Repository
mock:
	mockery -name=$(word 3, $(MAKECMDGOALS)) -case=underscore -dir $(word 2, $(MAKECMDGOALS)) -output $(word 2, $(MAKECMDGOALS))/mocks

## debug-ui: opens the debug ui (pprof)
debug-ui:
	open http://localhost:7778/debug/pprof

## upgrade-deps: upgrades dependencies
upgrade-deps:
	go get -u -t -d -v ./...
	go mod vendor
