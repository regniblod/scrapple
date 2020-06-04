#################################
# Base image
#################################
FROM golang as base

RUN apt-get update

RUN mkdir build

# Copy project
COPY . /build
WORKDIR /build

# Fetch dependencies.
RUN go mod vendor && go mod verify

# Create unprivileged user
#RUN adduser -S -D -H -h /build webserver
#USER webserver

#################################
# Dev image
#################################
FROM base as dev

ENV GO111MODULE=off
RUN go get github.com/cosmtrek/air
RUN go get github.com/go-delve/delve/cmd/dlv
RUN go get -tags 'postgres' -u github.com/golang-migrate/migrate/cmd/migrate
RUN go get github.com/vektra/mockery/.../
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.27.0
ENV GO111MODULE=on

CMD ["tail", "-f", "/dev/null"]

#################################
# Build app
#################################
FROM base as builder

# Build the binary
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -products -installsuffix cgo -ldflags '-s -w -extldflags "-static"' -o /build/main cmd/http-server/main.go

#################################
# Create products smaller image
#################################
FROM scratch as prod

# Import from builder.
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd

# Copy our static executable
COPY --from=builder /build/main /main

# Use an unprivileged user.
USER webserver

# Run the hello binary.
ENTRYPOINT ["/main"]
