# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS go
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY *.go ./
COPY binary ./binary
COPY valheim ./valheim
COPY cmd/fchedit/main.go cmd/fchedit/parsing.go ./cmd/fchedit/

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/fchedit ./cmd/fchedit

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=go /out/fchedit /usr/local/bin/fchedit
ENTRYPOINT ["/usr/local/bin/fchedit"]
