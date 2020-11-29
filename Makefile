include .env

.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n" && sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

.PHONY: start
## start: starts the Docker containers
start:
	docker-compose up -d

.PHONY: stop
## stop: stops the Docker containers
stop:
	docker-compose stop

.PHONY: ssh
## ssh: ssh's into the Docker container
ssh:
	docker exec -it ${APP_NAME}-app bash

.PHONY: build
## build: builds the app for prod
build:
	CGO_ENABLED=0 GOOS=linux go build --mod vendor -a -installsuffix cgo -ldflags '-s -w -extldflags "-static" -X main.build=${shell git rev-parse --short HEAD}' -o /app ./cmd/${DEV_MAIN_CMD}

.PHONY: test
## test: runs go test for all packages
test:
	docker exec -it ${APP_NAME}-app go test ./pkg/... ./internal/...

.PHONY: test-cover
## test-cover: runs go test for all packages with coverage
test-cover:
	docker exec -it ${APP_NAME}-app go test -v -cover ./pkg/... ./internal/...

.PHONY: vendor
## vendor: setup go modules
vendor:
	docker exec -it ${APP_NAME}-app go mod tidy
	docker exec -it ${APP_NAME}-app go mod vendor

.PHONY: lint
## lint: lints all Go files
lint:
	docker exec -it ${APP_NAME}-app /go/bin/golangci-lint run cmd/... internal/... pkg/...

.PHONY: watch
## watch: starts watching the code for changes
watch:
	docker exec -it ${APP_NAME}-app /go/bin/CompileDaemon -log-prefix=false -graceful-kill=true --build="go build -mod vendor -o ./tmp/main ./cmd/${DEV_MAIN_CMD}" -command="./tmp/main"

.PHONY: debug
## watch-debug: starts watching the code for changes and starts the debugger
debug:
	docker exec -it ${APP_NAME}-app go build -mod vendor -o ./tmp/main ./cmd/${DEV_MAIN_CMD}
	docker exec -it ${APP_NAME}-app dlv debug /build/cmd/${DEV_MAIN_CMD} --headless --listen=:${DEV_DEBUG_PORT} --api-version=2

.PHONY: mock
## mock: creates mocks for an interface. ex - make mocks pkg/team Repository
mock:
	docker exec -it ${APP_NAME}-app mockery -name=$(word 3, $(MAKECMDGOALS)) -case=underscore -dir $(word 2, $(MAKECMDGOALS)) -output $(word 2, $(MAKECMDGOALS))/mocks

.PHONY: debug-ui
## debug-ui: opens the debug ui (pprof)
debug-ui:
	open http://localhost:7778/debug/pprof

.PHONY: upgrade-deps
## upgrade-deps: upgrades dependencies
upgrade-deps:
	docker exec -it ${APP_NAME}-app go get -u -t -d -v ./...
	docker exec -it ${APP_NAME}-app go mod vendor
