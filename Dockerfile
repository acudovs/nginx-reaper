# Stage 1. Build Binary

# See https://hub.docker.com/_/golang/tags
FROM golang:1.24.3-alpine AS build

# See https://pkg.go.dev/cmd/go#hdr-Environment_variables
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

ADD . /build/
WORKDIR /build/

# Test Go formatting, duplicate output to stderr, exit if not formatted
# See https://pkg.go.dev/cmd/gofmt
RUN test -z "$(gofmt -d -e -s cmd/ internal/ | tee /dev/stderr)"

# Vet examines Go source code and reports suspicious constructs
# See https://pkg.go.dev/cmd/vet
RUN go vet ./...

# Unit tests with coverage
RUN go test -v -coverprofile=cover.out ./...
RUN go tool cover -func=cover.out

# Build stripped binary
RUN go build -ldflags "-s -w" nginx-reaper/cmd/nginx-reaper


# Stage 2. Build Image

# Static Go executables have no dependencies, start from the smallest possible image
FROM scratch
COPY --from=build /build/nginx-reaper /
CMD ["/nginx-reaper"]
