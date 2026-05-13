# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w \
      -X main.Version=${VERSION} \
      -X main.GitCommit=${GIT_COMMIT} \
      -X main.BuildTime=${BUILD_TIME}" \
    -o /build/payment ./cmd/payment

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy binary
COPY --from=builder /build/payment ./payment

# Copy CA cert for mTLS
COPY infra/certs/dev/ca.pem /etc/ssl/certs/parkir-pintar-ca.pem

USER appuser

EXPOSE 50053

HEALTHCHECK --interval=10s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50053/health || exit 1

ENTRYPOINT ["./payment"]
