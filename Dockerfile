#################################
# Base image
#################################
FROM golang as base

# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apt-get update && apt-get install git ca-certificates && update-ca-certificates

# Create appuser
ENV USER=appuser
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735RUN
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

# Copy project
COPY . /build
WORKDIR /build

# Fetch dependencies.
RUN go mod tidy
RUN go mod vendor
RUN go mod verify

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

ENTRYPOINT ["tail", "-f", "/dev/null"]

#################################
# Build app
#################################
FROM base as builder

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build --mod vendor -a -installsuffix cgo -ldflags '-s -w -extldflags "-static"' -o /app ./cmd/...

#################################
# Create products smaller image
#################################
FROM scratch as prod

# Import from builder.
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy our static executable
COPY --from=builder /app/ /

# Use an unprivileged user.
USER appuser:appuser

# Run the hello binary.
ENTRYPOINT ["/main"]
