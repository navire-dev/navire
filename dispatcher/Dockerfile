# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG NAVIRE_VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w -X github.com/navire-dev/navire/shared/version.Version=${NAVIRE_VERSION} -X github.com/navire-dev/navire/shared/version.Commit=${GIT_COMMIT} -X github.com/navire-dev/navire/shared/version.BuildDate=${BUILD_DATE}" \
    -o /out/navire-dispatcher ./cmd/dispatcher

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/navire-dispatcher /usr/local/bin/navire-dispatcher

ENTRYPOINT ["/usr/local/bin/navire-dispatcher"]
