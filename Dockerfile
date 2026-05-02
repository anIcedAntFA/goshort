# ---- Build stage ----
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Cache module layer — only invalidated when go.mod or go.sum change.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o goshort ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.21

# ca-certificates: HTTPS outbound; tzdata: correct time in SQLite datetime().
RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S goshort && adduser -S goshort -G goshort

WORKDIR /app

COPY --from=builder /build/goshort        /app/goshort
COPY --from=builder /build/docs/openapi.yaml /app/docs/openapi.yaml

RUN mkdir -p /app/data && chown -R goshort:goshort /app

USER goshort

EXPOSE 8080

VOLUME ["/app/data"]

ENTRYPOINT ["/app/goshort"]
