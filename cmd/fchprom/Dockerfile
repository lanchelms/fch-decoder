# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS go
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY *.go ./
COPY binary ./binary
COPY valheim ./valheim
COPY cmd/fchprom/main.go ./cmd/fchprom/main.go

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/fchprom ./cmd/fchprom

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=go /out/fchprom /usr/local/bin/fchprom
EXPOSE 9108
ENTRYPOINT ["/usr/local/bin/fchprom"]
