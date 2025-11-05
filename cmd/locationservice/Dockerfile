# syntax=docker/dockerfile:1
FROM golang:1.22-bookworm AS builder
WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod ./
COPY go.sum ./
COPY vendor ./vendor
COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    if [ -d vendor ]; then go build -mod=vendor -o /bin/locationservice ./cmd/locationservice; else go build -o /bin/locationservice ./cmd/locationservice; fi

FROM alpine:3.19
RUN addgroup -S app && adduser -S -G app app \
    && apk add --no-cache ca-certificates curl

USER app
WORKDIR /home/app
COPY --from=builder /bin/locationservice /usr/local/bin/locationservice
EXPOSE 8081 9090
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD curl -fsS http://127.0.0.1:8081/observability/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/locationservice"]
