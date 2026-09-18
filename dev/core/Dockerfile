# Development image only. Production uses core/Dockerfile and distroless.
FROM golang:1.27-alpine

ARG AIR_VERSION=v1.65.1

COPY go.mod go.sum ./
RUN go mod download

RUN apk add --no-cache ca-certificates git \
    && go install github.com/air-verse/air@${AIR_VERSION}

WORKDIR /workspace

ENTRYPOINT ["air", "-c", "/workspace/dev/core/air.toml"]
